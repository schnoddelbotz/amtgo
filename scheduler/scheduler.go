// Package scheduler acts as "cron" for scheduled AMT tasks and
// continuous monitoring of clients.
package scheduler

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schnoddelbotz/amtgo/amt"
	"github.com/schnoddelbotz/amtgo/database"
)

// Ember submits Jobs differently from database.Job model. Work-around:
type emberJob struct {
	JobType        int      `json:"job_type"`
	AmtcCmd        string   `json:"amtc_cmd"`
	AmtcDelay      float64  `json:"amtc_delay"`
	AmtcHosts      []string `json:"hosts"`
	OuID           string   `json:"ou_id"`
	RepeatInterval *int     `json:"repeat_interval"`
	RepeatDays     *int     `json:"repeat_days"`
	LastStarted    *int     `json:"last_started"`
	LastDone       *int     `json:"last_done"`
	StartTime      int      `json:"start_time"`
	Description    *string  `json:"description"`
}
type newJob struct {
	SingleJob emberJob `json:"job"`
}

// map HostID -> state
var lastStateMap = map[int]amt.Laststate{}

// mutex for updating lastStateMap
var mutex = &sync.Mutex{}

// ScheduledJobsRunloop periodically checks DB for scheduled tasks.
func ScheduledJobsRunloop(verbose bool) {
	lastRunMinute := -1

	for {
		//log.Println("Looking for scheduled jobs...")
		time.Sleep(30 * time.Second) // sleep first -- db may not be open yet...
		now := time.Now()
		nowWeekday := now.Weekday() + 1 // amtc-web uses sunday=1, go sunday=0
		//nowMinuteOfDay := 480           //
		nowMinuteOfDay := now.Hour()*60 + now.Minute()

		if lastRunMinute != nowMinuteOfDay {
			jobs := database.GetScheduledJobs(int(nowWeekday), nowMinuteOfDay)
			for _, job := range jobs {
				ou := database.GetOu(*job.OuID)
				optionset := database.GetOptionset(*ou.OptionsetID)
				optionset.Username = "admin" // FIXME
				optionset.Password = getPasswordFromFile(optionset.OptPassfile)
				if optionset.SwUseTLS == 1 && optionset.SwSkipcertchk != 1 && optionset.OptCacertfile != "" {
					optionset.CaCertData = amt.LoadCaCertFile(optionset.OptCacertfile)
				}
				myhosts := database.GetHostsByOu(ou.ID)
				var hostsStringArr []string
				for _, h := range myhosts {
					hostsStringArr = append(hostsStringArr, h.Hostname)
				}
				if verbose {
					log.Printf("Scheduled command: %s, delay: %f, hosts: %s", *job.AmtcCmd, *job.AmtcDelay, hostsStringArr)
				}
				if *job.AmtcCmd == "U" {
					database.InsertNotification(database.NotificationTypePowerOn, fmt.Sprintf("Scheduled power-up %s", ou.Name))
				} else if *job.AmtcCmd == "D" {
					database.InsertNotification(database.NotificationTypePowerOff, fmt.Sprintf("Scheduled power-down %s", ou.Name))
				}

				go amt.SequentialCommand(amt.ShortCommandMap[*job.AmtcCmd], hostsStringArr, optionset, *job.AmtcDelay)
				//    ^^^^^^^^^ logs to notification (started, OK/FAIL done)
				//    ^^^^^^^^^ same is used for GUI submitted jobs
			}
			lastRunMinute = nowMinuteOfDay
		}
	}
}

// CreateJob accepts a web-GUI submitted job, scheduled or interactive
func CreateJob(body io.ReadCloser) string {
	var uncleanJob newJob
	decoder := json.NewDecoder(body)
	err := decoder.Decode(&uncleanJob)
	if err == nil {
		j := uncleanJob.SingleJob
		ouid, _ := strconv.Atoi(j.OuID)
		switch j.JobType {
		case 1: // interactive job
			ou := database.GetOu(ouid)
			optionset := database.GetOptionset(*ou.OptionsetID)
			optionset.Username = "admin" // to-do: jobs assume default AMT username admin
			optionset.Password = getPasswordFromFile(optionset.OptPassfile)
			if optionset.SwUseTLS == 1 && optionset.SwSkipcertchk != 1 && optionset.OptCacertfile != "" {
				optionset.CaCertData = amt.LoadCaCertFile(optionset.OptCacertfile)
			}
			// ember submits hostIDs as string. convert...
			hostnames := database.GetHostNamesByID(j.AmtcHosts)
			message := fmt.Sprintf("%s %d hosts in %s", amt.ShortCommandMap[j.AmtcCmd], len(hostnames), ou.Name)
			database.InsertNotification(database.NotificationTypeUser, message)
			go amt.SequentialCommand(amt.ShortCommandMap[j.AmtcCmd], hostnames, optionset, j.AmtcDelay)
			return "{}"
		default: // scheduled job
			var sjob database.Job
			// clean up!:
			defaultAmtCmd := "U"
			defaultAmtDelay := 2.5
			defaultOu := 1
			amtOu := &defaultOu
			amtCmd := &defaultAmtCmd
			amtDelay := &defaultAmtDelay
			if j.AmtcCmd != "" {
				amtCmd = &j.AmtcCmd
			}
			if j.AmtcDelay != 0 {
				amtDelay = &j.AmtcDelay
			}
			if ouid != 0 {
				amtOu = &ouid
			}
			// ^^^
			sjob.JobType = 2
			sjob.AmtcCmd = amtCmd
			sjob.AmtcDelay = amtDelay
			sjob.OuID = amtOu
			sjob.StartTime = j.StartTime
			sjob.RepeatDays = j.RepeatDays
			sjob.Description = j.Description
			return database.InsertJob(sjob)
		}
	}
	return `{ "error" : "` + err.Error() + `"}`
}

// UpdateJob updates a scheduled job in DB
func UpdateJob(id int, body io.ReadCloser) string {
	var uncleanJob newJob
	decoder := json.NewDecoder(body)
	err := decoder.Decode(&uncleanJob)
	if err == nil {
		j := uncleanJob.SingleJob
		ouid, _ := strconv.Atoi(j.OuID)
		var sjob database.Job
		sjob.ID = id
		sjob.JobType = 2
		sjob.AmtcCmd = &j.AmtcCmd
		sjob.AmtcDelay = &j.AmtcDelay
		sjob.OuID = &ouid
		sjob.StartTime = j.StartTime
		sjob.RepeatDays = j.RepeatDays
		sjob.Description = j.Description
		return database.UpdateJob(sjob)
	}
	return "{}"
}

// MonitoringRunloop periodically scans clients' powerstate via AMT.
func MonitoringRunloop(verbose bool) {
	lastStateMap = make(map[int]amt.Laststate)
	cmd := amt.CmdInfo
	// limit concurrent threads to...
	concurrency := 200
	for {
		time.Sleep(15 * time.Second)
		if verbose {
			log.Println("Host monitoring triggering scans...")
		}
		optionsets := database.GetOptionsets()
		hosts := database.GetHosts()
		ous := database.GetOus()
		toScan := 0
		sem := make(chan bool, concurrency)

		for _, optionsetX := range optionsets {
			//log.Printf(" Scan optionset %s", optionsetX.Name)
			for _, ouX := range ous {
				// FIXME: Web-GUI says "Log(ging)" in OU, but it's monitoring+logging!
				if ouX.Logging == 1 && *ouX.OptionsetID == optionsetX.ID {
					optionsetX.Password = getPasswordFromFile(optionsetX.OptPassfile)
					optionsetX.Username = "admin"
					if optionsetX.SwUseTLS == 1 && optionsetX.SwSkipcertchk != 1 && optionsetX.OptCacertfile != "" {
						optionsetX.CaCertData = amt.LoadCaCertFile(optionsetX.OptCacertfile)
					}
					//log.Printf("  Scan ou %s", ouX.Name)
					for _, hostX := range hosts {
						if hostX.Enabled == 1 && hostX.OuID == ouX.ID && *ouX.OptionsetID == optionsetX.ID {
							//log.Printf(" Scan optionset:%s OU:%s host:%s", optionsetX.Name, ouX.Name, hostX.Hostname)
							var client amt.Laststate // FIXME...?
							client.Hostname = hostX.Hostname
							client.HostID = hostX.ID
							client.ID = hostX.ID
							toScan = toScan + 1
							sem <- true
							go func() {
								defer func() { <-sem }()
								//log.Printf("Go command for: %s", client.Hostname)
								result := amt.Command(client, cmd, optionsetX)
								if verbose {
									log.Printf("%s %-15s OS:%-7d AMT:%02d HTTP:%03d %s\n", cmd, result.Hostname,
										result.OpenPort, result.StateAMT, result.StateHTTP, result.Usermessage)
								}
								updateLastStateMap(result)
							}()
						}
					}
				}
			}
		}
		for i := 0; i < cap(sem); i++ {
			sem <- true
		}
		if verbose {
			log.Printf("Host monitoring scans done -- sleeping")
		}
	}
}

func updateLastStateMap(stateNow amt.Laststate) {
	// diff lastStateMap with newstate
	mutex.Lock()
	if last, ok := lastStateMap[stateNow.HostID]; ok {
		if last.OpenPort == stateNow.OpenPort &&
			last.StateAMT == stateNow.StateAMT &&
			last.StateHTTP == stateNow.StateHTTP &&
			last.Usermessage == stateNow.Usermessage {
			mutex.Unlock()
			return
		}
	}
	// if it differs / doesnt exist yet, update and set state_begin
	stateNow.StateBegin = int(time.Now().Unix())
	lastStateMap[stateNow.HostID] = stateNow
	mutex.Unlock()

	database.InsertStatelog(stateNow.HostID, stateNow.StateHTTP, stateNow.StateAMT, stateNow.OpenPort)
}

func getPasswordFromFile(f string) (password string) {
	fileContents, err := ioutil.ReadFile(f)
	if err != nil {
		log.Printf("Error opening password file %s: %s", f, err)
		return
	}
	password = strings.TrimSpace(string(fileContents))
	return
}

// GetLaststatesJSON is consumed by webserver to report current client state.
func GetLaststatesJSON() string {
	data := []amt.Laststate{}
	mutex.Lock()
	for _, host := range lastStateMap {
		data = append(data, host)
	}
	mutex.Unlock()
	json, _ := json.Marshal(data)
	return "{\"laststates\":" + string(json) + "}"
}
