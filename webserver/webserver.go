// Package webserver serves amtc-web, a web GUI for AMT management.
// Assets (html, css, js...) are included via go-bindata's Asset() functions.
package webserver

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"

	"github.com/schnoddelbotz/amtgo/database"
	"github.com/schnoddelbotz/amtgo/scheduler"
)

type assetInfo struct {
	ContentType string
	IsGzipped   bool
}

type apiFuncs struct {
	/* sequence: CR(R)UD */
	Create    func(io.ReadCloser) string
	GetAll    func() string
	GetSingle func(id int) string
	Update    func(id int, reader io.ReadCloser) string
	Delete    func(id int) (string, bool)
}

var (
	// HttpdUseTLS controls whether content is served via HTTPS.
	HttpdUseTLS bool
	// TLSCertDir points to a directory holding HTTPS/SSL webserver certificates.
	TLSCertDir string
	// ListenAddr controls the IP/port which the webserver listens on.
	ListenAddr string
	// AppVersion is set by main() by calling Version() (in main.go)
	AppVersion string
	// DisableSessions is used for testing only
	DisableSessions = false

	assetMap = map[string]assetInfo{
		"index.html":                      {"text/html", true},
		"amtc-favicon.png":                {"image/png", false},
		"js/jslibs.js":                    {"application/javascript", true},
		"css/styles.css":                  {"text/css", true},
		"fonts/fontawesome-webfont.woff":  {"font/woff", false},
		"fonts/fontawesome-webfont.woff2": {"font/woff2", false},
		"page/about.md":                   {"text/plain", false},
		"page/first-steps.md":             {"text/plain", false},
		"page/configure-amt.md":           {"text/plain", false},
	}

	funcMap = map[string]apiFuncs{
		"rest-config.js": {nil, getConfigJs, nil, nil, nil},
		"phptests":       {nil, pseudoPHPTests, nil, nil, nil},
		"systemhealth":   {nil, getSystemHealth, nil, nil, nil},
		"ous":            {database.InsertOu, database.GetOusJSON, database.GetOuJSON, database.UpdateOu, database.DeleteOu},
		"notifications":  {nil, database.GetNotificationsJSON, database.GetNotificationJSON, nil, nil},
		"users":          {nil, database.GetUsersJSON, database.GetUserJSON, nil, database.DeleteUser},
		"hosts":          {database.InsertHost, database.GetHostsJSON, database.GetHostJSON, nil, database.DeleteHost},
		"laststates":     {nil, scheduler.GetLaststatesJSON, database.GetLaststateJSON, nil, nil},
		"optionsets":     {database.InsertOptionset, database.GetOptionsetsJSON, database.GetOptionsetJSON, database.UpdateOptionset, database.DeleteOptionset},
		"jobs":           {scheduler.CreateJob, database.GetJobsJSON, database.GetJobJSON, scheduler.UpdateJob, database.DeleteJob},
		"logdays":        {nil, database.GetLogdaysJSON, nil, nil, nil},
	}

	startupTime = time.Now()
	aKey        = securecookie.GenerateRandomKey(64)
	eKey        = securecookie.GenerateRandomKey(32)
	store       = sessions.NewCookieStore(aKey, eKey)
)

func init() {
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 8, // 8 hours
		HttpOnly: true,
	}
}

// Run starts the webserver.
func Run(verbose bool) {
	serverProto := "http"
	serverPort := "8080"
	serverIP := "127.0.0.1"

	if HttpdUseTLS {
		serverProto = "https"
	}
	if !strings.Contains(ListenAddr, ":") {
		log.Fatal("Listen address must be of format [IP]:Port")
	}
	_tmp := strings.Split(ListenAddr, ":")
	serverPort = _tmp[1]
	if _, err := strconv.Atoi(serverPort); err != nil {
		log.Fatal("Invalid port for --listen argument")
	}
	if _tmp[0] != "" {
		serverIP = _tmp[0]
	}

	log.Printf("About to listen on %s ...", ListenAddr)
	log.Printf("Go to %s://%s:%s/", serverProto, serverIP, serverPort)
	// try to open DB; explicit -init-db once required for now
	database.OpenDB()
	defer database.CloseDB()

	r := mux.NewRouter()
	r.Handle("/", staticHandler)
	r.Handle("/index.html", staticHandler)
	r.Handle("/amtc-favicon.png", staticHandler)
	r.Handle("/js/{.*}", staticHandler)
	r.Handle("/css/{.*}", staticHandler)
	r.Handle("/fonts/{.*}", staticHandler)
	r.Handle("/page/{.*}", staticHandler)
	r.Handle("/rest-api.php/{.*}", restAPIHandler)
	r.Handle("/rest-api.php/{.*}/{.*}", restAPIHandler)
	r.Handle("/rest-api.php/statelogs/{.*}/{.*}", statelogAPIHandler)

	var err error
	if HttpdUseTLS {
		if !checkCertFilesAreReadable(TLSCertDir) {
			createSelfSignedCert(TLSCertDir)
		}
		err = http.ListenAndServeTLS(ListenAddr, TLSCertDir+"/cert.pem", TLSCertDir+"/key.pem", handlers.LoggingHandler(os.Stdout, r))
	} else {
		err = http.ListenAndServe(ListenAddr, handlers.LoggingHandler(os.Stdout, r))
	}
	if err != nil {
		log.Fatal(err)
	}
}

func getConfigJs() string {
	isConfigured := "true"
	if len(database.GetUsers()) == 0 {
		isConfigured = "false"
	}
	r := "DS.RESTAdapter.reopen({ namespace: '/rest-api.php' });\n"
	r = r + "var AMTCWEB_IS_CONFIGURED = " + isConfigured + ";"
	return r
}

func pseudoPHPTests() string {
	// legacy emberjs app: act as if we were running PHP to satisfy tests
	r := `{"phptests":[{"id":"php53","description":"","result":true,"remedy":""},{"id":"freshsetup","description":"","result":true,"remedy":""},{"id":"data","description":"","result":true,"remedy":""},{"id":"config","description":"","result":true,"remedy":""},{"id":"curl","description":"","result":true,"remedy":""},{"id":"pdo","description":"","result":true,"remedy":""},{"id":"pdo_sqlite","description":"","result":true,"remedy":""},{"id":"pdo_mysql","description":"","result":true,"remedy":""},{"id":"pdo_oci","description":"","result":false,"remedy":""},{"id":"pdo_pgsql","description":"","result":false,"remedy":""}],"authurl":""}`
	return r
}

func createInitialUser(username string, realname string, password string) string {
	createUser(username, realname, password)
	return `{"message":"Configuration written successfully"}`
}

func getSystemHealth() string {
	type systemhealth struct {
		Phpversion            string `json:"phpversion"`
		Uptime                string `json:"uptime"`
		Datetime              string `json:"datetime"`
		Diskfree              string `json:"diskfree"`
		Lastmonitoringstarted int    `json:"lastmonitoringstarted"`
		Lastmonitoringdone    int    `json:"lastmonitoringdone"`
		Activejobs            int    `json:"activejobs"`
		Activeprocesses       int    `json:"activeprocesses"`
		Monitorcount          int    `json:"monitorcount"`
		Amtcversion           string `json:"amtcversion"`
		Logsize               int    `json:"logsize"`
		Logmodtime            string `json:"logmodtime"` // amtgo only
	}
	type response struct {
		Systemhealth systemhealth `json:"systemhealth"`
	}
	var data response
	data.Systemhealth.Amtcversion = AppVersion
	data.Systemhealth.Phpversion = runtime.Version()
	data.Systemhealth.Activeprocesses = runtime.NumGoroutine()
	data.Systemhealth.Datetime = time.Now().String()
	upHours := time.Now().Sub(startupTime).Hours()
	data.Systemhealth.Uptime = fmt.Sprintf("%.2f hours (~ %.0f days)", upHours, upHours/24)
	runtime.GC()
	var memstats runtime.MemStats
	runtime.ReadMemStats(&memstats)
	data.Systemhealth.Diskfree = fmt.Sprintf("%.2f MB", float64(memstats.HeapSys)/1024/1024)
	// diskfree?...
	// https://github.com/StalkR/goircbot/blob/master/lib/disk/space_windows.go
	// https://github.com/StalkR/goircbot/blob/master/lib/disk/space_unix.go
	json, _ := json.Marshal(data)
	return string(json)
}

var staticHandler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
	requestPath := request.URL.Path[1:]
	if requestPath == "" {
		requestPath = "index.html"
	}
	if info, ok := assetMap[requestPath]; ok {
		// serve static / compiled-in assets (using go-bindata)
		gzipExtension := ""
		w.Header().Set("Content-Type", info.ContentType)
		if info.IsGzipped {
			w.Header().Set("Content-Encoding", "gzip")
			gzipExtension = ".gz"
		}
		data, _ := Asset(requestPath + gzipExtension)
		w.Write(data)
	} else {
		http.NotFound(w, request)
	}
})

var restAPIHandler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
	// http://thenewstack.io/make-a-restful-json-api-go/ ??
	pathComponents := strings.Split(request.URL.Path[1:], "/")
	componentCount := len(pathComponents)
	responsedata := "{}"
	contentType := "application/json"

	session, err := store.Get(request, "amtgo-session")
	if !DisableSessions {
		if pathComponents[1] != "rest-config.js" && pathComponents[1] != "authenticate" &&
			pathComponents[1] != "phptests" && pathComponents[1] != "submit-configuration" &&
			pathComponents[1] != "systemhealth" {
			if err != nil || session.Values["username"] == nil {
				w.Header().Set("Content-Type", contentType)
				w.Write([]byte(`{"notifications":[], "laststates":[], "error":"unauthenticated"}`))
				return
			}
		}
	}

	if afunc, ok := funcMap[pathComponents[1]]; ok {
		requestForID := false
		var err error
		var id int
		if componentCount == 3 {
			if id, err = strconv.Atoi(pathComponents[2]); err == nil {
				requestForID = true
			}
		}
		defer request.Body.Close()
		switch request.Method {
		// check nil...!
		case http.MethodPost:
			responsedata = afunc.Create(request.Body)
		case http.MethodGet:
			if requestForID {
				responsedata = afunc.GetSingle(id)
			} else {
				responsedata = afunc.GetAll()
			}
		case http.MethodPut:
			if requestForID {
				responsedata = afunc.Update(id, request.Body)
			}
		case http.MethodDelete:
			if requestForID {
				var success bool
				responsedata, success = afunc.Delete(id)
				if !success {
					w.WriteHeader(http.StatusForbidden)
				}
			}
		default:
			http.NotFound(w, request)
		}
	} else if pathComponents[1] == "authenticate" && request.Method == http.MethodPost {
		request.ParseForm()
		username := request.Form.Get("username")
		password := request.Form.Get("password")
		if authUser(username, password) {
			session.Values["username"] = username
			session.Save(request, w)
			responsedata = `{"result":"success","fullname":null}` + "\n"
		} else {
			responsedata = `{"result":"fail","fullname":null}` + "\n"
		}
	} else if pathComponents[1] == "submit-configuration" {
		if len(database.GetUsers()) == 0 {
			request.ParseForm()
			username := request.Form.Get("mysqlUser")
			realname := request.Form.Get("mysqlHost")
			password := request.Form.Get("mysqlPassword")
			responsedata = createInitialUser(username, realname, password)
		} else {
			responsedata = `{"errorMsg":"Forbidden, users already exist"}`
		}
	} else if pathComponents[1] == "logout" {
		session.Options.MaxAge = -1
		session.Save(request, w)
		responsedata = `{"message":"success"}`
	} else {
		http.NotFound(w, request)
	}

	if pathComponents[1] == "rest-config.js" && request.Method == http.MethodGet {
		contentType = "application/javascript"
	}

	w.Header().Set("Content-Type", contentType)
	w.Write([]byte(responsedata))
})

var statelogAPIHandler = http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
	pathComponents := strings.Split(request.URL.Path[1:], "/")
	ouID, _ := strconv.Atoi(pathComponents[2])
	unixtime, _ := strconv.Atoi(pathComponents[3])
	w.Header().Set("Content-Type", "application/json")
	w.Write(database.GetStatelogsJSON(ouID, unixtime))
})
