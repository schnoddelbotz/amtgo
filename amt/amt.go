// Package amt provides Intel AMT interaction methods.
package amt

import (
	"fmt"
	"io/ioutil"
	"net"
	"regexp"
	"strconv"
	"time"

	dac "github.com/schnoddelbotz/amtgo/amt/digest_auth_client"
)

// Verbose bool controls verbosity
var Verbose bool

var portMap = map[int]string{
	0:    "none",
	22:   "SSH",
	3389: "RDP",
}

// CliCommand executes a single AMT command on a list of hosts.
func CliCommand(cmd string, hosts []string, options Optionset) {
	if len(hosts) == 0 {
		fmt.Println("Error: Expected list of hostnames as arguments")
		return
	}
	if Verbose {
		fmt.Printf("%s for %s ...\n", cmd, hosts)
	}

	// hack: support legacy db model for optionsets via cli
	if options.CliUseTLS {
		options.SwUseTLS = 1
	}
	if options.CliSkipcertchk {
		options.SwSkipcertchk = 1
	} else if options.OptCacertfile != "" {
		options.CaCertData = LoadCaCertFile(options.OptCacertfile)
	}

	stateChannel := make(chan Laststate)
	for _, host := range hosts {
		var client Laststate
		var result Laststate
		client.Hostname = host

		if cmd == CmdInfo {
			go func() {
				stateChannel <- Command(client, cmd, options)
			}()
		} else {
			result = Command(client, cmd, options)
			if result.StateHTTP != 0 {
				result.Usermessage = fmt.Sprintf("%s S%d (%s)", httpReturncodeTextMap[result.StateHTTP],
					result.StateAMT, legacyPowerstateTextMap[result.StateAMT])
			}
			fmt.Printf("%s %-15s OS:%-7s AMT:%02d HTTP:%03d %s\n", cmd, result.Hostname,
				portMap[result.OpenPort], result.StateAMT, result.StateHTTP, result.Usermessage)
			time.Sleep(time.Duration(options.CliDelay) * time.Millisecond)
		}
	}

	if cmd == CmdInfo {
		for range hosts {
			result := <-stateChannel
			if result.StateHTTP != 0 {
				result.Usermessage = fmt.Sprintf("%s S%d (%s)", httpReturncodeTextMap[result.StateHTTP],
					result.StateAMT, legacyPowerstateTextMap[result.StateAMT])
			}
			fmt.Printf("%s %-15s OS:%-7s AMT:%02d HTTP:%03d %s\n", cmd, result.Hostname,
				portMap[result.OpenPort], result.StateAMT, result.StateHTTP, result.Usermessage)
		}
	}
}

// Command executes a AMT command on a single host and returns execution result
func Command(host Laststate, cmd string, options Optionset) (result Laststate) {
	command := cmdMap[cmd]
	amtPort := "16992"
	amtProto := "http"
	amtTLSskipCertCheck := false
	if Verbose {
		fmt.Printf("start amt_command host %d: %s command: %s\n", host.HostID, host.Hostname, cmd)
	}
	if options.SwUseTLS == 1 {
		amtPort = "16993"
		amtProto = "https"
	}
	if options.SwSkipcertchk == 1 {
		amtTLSskipCertCheck = true
	}
	result = host

	cmdPayload, _ := Asset(command.CommandOne)
	uri := amtProto + "://" + host.Hostname + ":" + amtPort + "/wsman"
	dr := dac.NewRequest(options.Username, options.Password, "POST", uri,
		string(cmdPayload), time.Duration(options.OptTimeout)*time.Second,
		amtTLSskipCertCheck, options.CaCertData)

	response1, err := dr.Execute()
	var data string
	if err != nil {
		result.StateAMT = 16
		result.StateHTTP = 0
		result.Usermessage = err.Error()
		return
	}

	result.StateHTTP = response1.StatusCode
	bodyBytes, _ := ioutil.ReadAll(response1.Body)
	data = string(bodyBytes)
	response1.Body.Close()

	if command.IsTwoStep && result.StateHTTP == 200 {
		//fmt.Printf("STEP 2... start /w data: %s\n", data)
		cmdPayload, _ = Asset(command.CommandTwo)
		cmdPayload2 := string(cmdPayload)

		if cmd == CmdInfo {
			enumContext := getEnumContext(data)
			if Verbose {
				fmt.Printf("Using enumctx %s for %s\n", enumContext, host.Hostname)
			}
			cmdPayload2 = fmt.Sprintf(cmdPayload2, enumContext)
		}

		dr.UpdateRequest(options.Username, options.Password, "POST", uri,
			string(cmdPayload2), time.Duration(options.OptTimeout)*time.Second,
			amtTLSskipCertCheck, options.CaCertData)

		response2, err := dr.Execute()
		if err != nil {
			result.Usermessage = err.Error()
			result.StateAMT = 16
			result.StateHTTP = 0
			return
		}

		result.StateHTTP = response2.StatusCode
		bodyBytes, _ := ioutil.ReadAll(response2.Body)
		data = string(bodyBytes)
		response2.Body.Close()

		if cmd == CmdInfo && result.StateHTTP == 200 {
			powerState := getPowerstate(data)
			result.StateAMT = legacyPowerstateMap[powerState]
			if powerState == stateOn {
				var probePorts []int
				if options.SwScan3389 == 1 {
					probePorts = append(probePorts, 3389)
				}
				if options.SwScan22 == 1 {
					probePorts = append(probePorts, 22)
				}
				result.OpenPort = ProbeHostPorts(host.Hostname, probePorts)
			}
		}
	}

	return
}

// SequentialCommand executes a command on hosts in sequential/non-parallel fashion.
func SequentialCommand(cmd string, hosts []string, opt Optionset, delay float64) {
	fmt.Printf("Running command: %s with delay %f on hosts: %s\n", cmd, delay, hosts)

	for _, host := range hosts {
		fmt.Printf("Running command: %s on host: %s\n", cmd, host)

		var client Laststate
		client.Hostname = host
		Command(client, cmd, opt)
		// if result.StateHTTP != 0 {
		//  result.Usermessage = fmt.Sprintf("%s S%d (%s)", httpReturncodeTextMap[result.StateHTTP],
		//    result.StateAMT, legacyPowerstateTextMap[result.StateAMT])
		// }
		time.Sleep(time.Duration(delay) * time.Second)
	}

	// FIXME: add record to notification table /w fails+success
	fmt.Printf("Command completed.\n")
}

// ProbeHostPorts probes for given host ports.
// If none are open, 0 is returned.
func ProbeHostPorts(host string, ports []int) (openPort int) {
	openPort = 0
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 2*time.Second)
		if err == nil {
			openPort = port
			conn.Close()
			return
		}
	}
	return
}

func getEnumContext(xmldata string) string {
	// ...<g:EnumerationContext>06000000-0000-0000-0000-000000000000</g:EnumerationContext>...
	regex := regexp.MustCompile(`<g:EnumerationContext>([^<]+)</g:EnumerationContext>`)
	resultSlice := regex.FindStringSubmatch(xmldata)
	if resultSlice != nil {
		return resultSlice[1]
	}
	fmt.Println("ERROR: enum context regex -- no match in:", xmldata)
	return ""
}

func getPowerstate(xmldata string) int {
	// ...<h:PowerState>8</h:PowerState>
	regex := regexp.MustCompile(`<h:PowerState>([^<]+)</h:PowerState>`)
	resultSlice := regex.FindStringSubmatch(xmldata)
	if resultSlice != nil {
		if ps, err := strconv.Atoi(resultSlice[1]); err == nil {
			return ps
		}
		return -1
	}
	fmt.Println("ERROR: could not find powerstate in:", xmldata)
	return -2
}

// LoadCaCertFile loads a certificate from file and returns it as []byte
func LoadCaCertFile(filename string) []byte {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("ERROR: Cannot read CA cert from %s\n", filename)
		return []byte{}
	}
	return data
}
