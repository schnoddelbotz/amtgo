package webserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/schnoddelbotz/amtgo/database"
)

func init() {
	ListenAddr = ":8080"
	AppVersion = "0.0.0-testing"
	DisableSessions = true

	tempdir, err := ioutil.TempDir("", "amtgo-web")
	if err != nil {
		fmt.Print("Error creating temp dir for DB")
		os.Exit(1)
	}
	database.DbFile = tempdir + "/test.db"
	database.DbDriver = "sqlite3"

	//createUser("tester", "tester", "tester")
	go Run(false)
	time.Sleep(1 * time.Second)
}

func Test404(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/rest-api.php/fooNonExistant")
	if err != nil {
		t.Errorf("Unable to GET a 404 API URL: %s", err)
	} else {
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Got a non-404 for an invalid API route")
		}
		resp.Body.Close()
	}

	resp, err = http.Get("http://localhost:8080/js/notExistant")
	if err != nil {
		t.Errorf("Unable to GET a 404 static URL: %s", err)
	} else {
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Got a non-404 for an invalid static route")
		}
		resp.Body.Close()
	}
}

func TestConfigJS(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/rest-api.php/rest-config.js")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Unable to GET rest-config.js: %s", err)
	} else {
		expect := "DS.RESTAdapter.reopen({ namespace: '/rest-api.php' });\n"
		expect = expect + "var AMTCWEB_IS_CONFIGURED = true;"
		if resp.Header.Get("Content-type") != "application/javascript" {
			t.Errorf("Wrong content-type for rest-config.js: %s", resp.Header.Get("Content-type"))
		}
		if string(body) != expect {
			t.Error("Wrong content for rest-config.js")
		}
		resp.Body.Close()
	}
}

func TestSystemhealth(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/rest-api.php/systemhealth")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Unable to GET systemhealth: %s", err)
	} else {
		if resp.Header.Get("Content-type") != "application/json" {
			t.Errorf("Wrong content-type for systemhealth: %s", resp.Header.Get("Content-type"))
		}
		if !strings.Contains(string(body), `"amtcversion":"0.0.0-testing"`) {
			t.Errorf("Didn't find expected amtcversion string in systemhealth")
		}
		resp.Body.Close()
	}
}

func TestIndex(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Unable to GET /: %s", err)
	} else {
		if resp.Header.Get("Content-type") != "text/html" {
			t.Errorf("Wrong content-type for /: %s", resp.Header.Get("Content-type"))
		}
		if !strings.Contains(string(body), `<title>amtc-web - AMT/DASH remote power management</title>`) {
			t.Errorf("Didn't find expected <title> in /")
		}
		resp.Body.Close()
	}
}

func TestCss(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/css/styles.css")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Unable to GET /css/styles.css: %s", err)
	} else {
		if resp.Header.Get("Content-type") != "text/css" {
			t.Errorf("Wrong content-type for /css/styles.css: %s", resp.Header.Get("Content-type"))
		}
		if !strings.Contains(string(body), `min-device-width: 1280px`) {
			t.Error("Didn't find expected min-device-width in /css/styles.css")
		}
		resp.Body.Close()
	}
}

func TestFont(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/fonts/fontawesome-webfont.woff")
	if err != nil {
		t.Errorf("Unable to GET fontawesome-webfont.woff: %s", err)
	} else {
		if resp.Header.Get("Content-type") != "font/woff" {
			t.Errorf("Wrong content-type for fontawesome-webfont.woff: %s", resp.Header.Get("Content-type"))
		}
	}
	// FIXME use assetMap and test all with one func...
}

func TestJslibs(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/js/jslibs.js")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Unable to GET /js/jslibs.js: %s", err)
	} else {
		if resp.Header.Get("Content-type") != "application/javascript" {
			t.Errorf("Wrong content-type for /js/jslibs.js: %s", resp.Header.Get("Content-type"))
		}
		if !strings.Contains(string(body), `jQuery`) {
			t.Error("Didn't find expected jQuery in /js/jslibs.js")
		}
		resp.Body.Close()
	}
}

func TestSession(t *testing.T) {
	// TBD ... in init()
	// resp, err := http.Get("http://localhost:8080/js/jslibs.js")
	// body, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	//  t.Errorf("Unable to GET /js/jslibs.js: %s", err)
	// } else {
	//  if resp.Header.Get("Content-type") != "application/javascript" {
	//    t.Errorf("Wrong content-type for /js/jslibs.js: %s", resp.Header.Get("Content-type"))
	//  }
	//  if !strings.Contains(string(body), `jQuery`) {
	//    t.Error("Didn't find expected jQuery in /js/jslibs.js")
	//  }
	//  resp.Body.Close()
	// }
}

func TestHostAPI(t *testing.T) {
	var jsonStr = []byte(`{"host":{"hostname":"my-test-host-00","enabled":false,"ou_id":"1","laststate":null}}`)
	req, err := http.NewRequest("POST", "http://localhost:8080/rest-api.php/hosts", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Cannot POST host: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Error("Received non-200 from /rest-api.php/hosts")
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	// parse/verify creation response
	type singleHost struct {
		Host database.Host `json:"host"`
	}
	var myHost singleHost
	//var hostJSON = /[]byte(string(body))
	err = json.Unmarshal(body, &myHost)
	if err != nil {
		t.Error("Failed to unmarshal JSON response for newly created Host")
	}
	if myHost.Host.Hostname != "my-test-host-00" {
		t.Error("Submitted host hasn't desired hostname")
	}

	// GET host just created
	getURL := fmt.Sprintf("http://localhost:8080/rest-api.php/hosts/%d", myHost.Host.ID)
	resp, err = http.Get(getURL)
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Unable to GET /hosts/1: %s", err)
	} else {
		if resp.Header.Get("Content-type") != "application/json" {
			t.Errorf("Wrong content-type for /hosts/1: %s", resp.Header.Get("Content-type"))
		}
		if !strings.Contains(string(body), `"ou_id":1,"hostname":"my-test-host-00","enabled":1`) {
			t.Errorf("Didn't find expected string in /hosts/1: %s", string(body))
		}
		resp.Body.Close()
	}

	// DELETE host
	req, err = http.NewRequest("DELETE", getURL, nil)
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("Cannot POST host: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("Received non-200 from DELETE /rest-api.php/hosts/<test-host-id>")
	}
}

func TestOu(t *testing.T) {
	resp, err := http.Get("http://localhost:8080/rest-api.php/ous/2")
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Unable to GET /ous/1: %s", err)
	} else {
		if resp.Header.Get("Content-type") != "application/json" {
			t.Errorf("Wrong content-type for /ous/2: %s", resp.Header.Get("Content-type"))
		}
		if !strings.Contains(string(body), `"name":"Student labs","description":"Computer rooms"`) {
			t.Errorf("Didn't find expected OU name string /ous/2: %s", string(body))
		}
		resp.Body.Close()
	}

	//UPDATE OU
	var jsonStr = []byte(`{"ou":{"name":"Student labsZZ","description":"Computer roomsXX","idle_power":35,"logging":true,"parent_id":"1","optionset_id":"1"}}`)
	req, err := http.NewRequest("PUT", "http://localhost:8080/rest-api.php/ous/2", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Errorf("Cannot PUT /ous/2: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Error("Received non-200 from /rest-api.php/ous")
	}
	body, _ = ioutil.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"name":"Student labsZZ","description":"Computer roomsXX"`) {
		t.Errorf("Updating OU /ous/2 failed: %s", string(body))
	}
	resp.Body.Close()
}
