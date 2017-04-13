// Package database acts as storage backend for amtgo's webserver.
package database

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	// import SQLite 3 driver
	_ "github.com/mattn/go-sqlite3"
	// import MySQL driver
	_ "github.com/go-sql-driver/mysql"

	"github.com/schnoddelbotz/amtgo/amt"
)

var db *sqlx.DB

var (
	// DbDriver selects driver to use: sqlite3 | mysql
	DbDriver string
	// DbFile points a SQLite database file
	DbFile string
	// DbUser etc. below are MySQL only
	DbUser string
	// DbPassword for MySQL
	DbPassword string
	// DbName for MySQL
	DbName string
	// DbHost for MySQL
	DbHost string
	// DbPort for MySQL
	DbPort string
)

// OpenDB opens DB connection based on driver
func OpenDB() {
	switch DbDriver {
	case "mysql":
		OpenDBMySQL()
	default:
		OpenDBSQlite()
	}
}

// InitDB initializes DB based on driver
func InitDB() {
	switch DbDriver {
	case "mysql":
		InitDBMysql()
	default:
		InitDBSQlite()
	}
}

// OpenDBSQlite opens SQLite database
func OpenDBSQlite() {
	// try opening database file
	var err error
	// if verbose {
	//  log.Printf("Using database file: %s", dbFile)
	// }
	db, err = sqlx.Open("sqlite3", DbFile+"?cache=shared&mode=rwc&_busy_timeout=9999999")
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(1)

	// initialize database with default schema
	if _, err := os.Stat(DbFile); os.IsNotExist(err) {
		InitDBSQlite()
	}

	db.Exec("PRAGMA foreign_keys = ON")
}

// OpenDBMySQL opens MySQL database
func OpenDBMySQL() {
	// try opening database file
	var err error
	dsn := fmt.Sprintf("%s:%s@(%s:%s)/%s", DbUser, DbPassword, DbHost, DbPort, DbName)
	db, err = sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to open database: %s", err)
	}
	log.Printf("Successfully connected to MySQL: %s@%s", DbUser, DbHost)

	optionSets := GetOptionsets()
	if len(optionSets) == 0 {
		InitDBMysql()
	}
}

// InitDBSQlite initializes DB schema
func InitDBSQlite() {
	_, err := db.Exec(sqliteSchema)
	if err != nil {
		log.Fatalf("Fatal error with %s: %s", DbFile, err)
	}
	log.Printf("Successfully initialized new DB: %s", DbFile)
}

// InitDBMysql initializes DB schema
func InitDBMysql() {
	// multiStatements=true allows multiple SQL statements in one query
	dsn := fmt.Sprintf("%s:%s@(%s:%s)/%s?multiStatements=true", DbUser, DbPassword, DbHost, DbPort, DbName)
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to open database: %s", err)
	}
	_, err = db.Exec(mysqlSchema)
	if err != nil {
		log.Fatalf("Fatal error with DB: %s", err)
	}
	log.Printf("Successfully initialized new DB: %s @ %s", DbName, DbHost)
	db.Close()
}

// CloseDB closes SQLite database
func CloseDB() {
	db.Close()
}

/// GET ALL records

// GetOusJSON gets all OUs
func GetOusJSON() string {
	var data Ous
	db.Select(&data.Ous, "SELECT * FROM ou")
	json, _ := json.Marshal(data)
	return string(json)
}

// GetNotificationsJSON gets all notifications
func GetNotificationsJSON() string {
	var data Notifications
	db.Select(&data.Notifications, "SELECT * FROM notification ORDER BY tstamp DESC LIMIT 10")
	json, _ := json.Marshal(data)
	return string(json)
}

// GetUsersJSON gets all users
func GetUsersJSON() string {
	var data Users
	db.Select(&data.Users, "SELECT * FROM user")
	json, _ := json.Marshal(data)
	return string(json)
}

// GetHostsJSON gets all hosts
func GetHostsJSON() string {
	var data Hosts
	db.Select(&data.Hosts, "SELECT * FROM host ORDER BY hostname")
	json, _ := json.Marshal(data)
	return string(json)
}

// GetOptionsetsJSON gets optionsets
func GetOptionsetsJSON() string {
	var data amt.Optionsets
	db.Select(&data.Optionsets, "SELECT * FROM optionset")
	json, _ := json.Marshal(data)
	return string(json)
}

// GetJobsJSON gets all jobs
func GetJobsJSON() string {
	var data Jobs
	db.Select(&data.Jobs, "SELECT * FROM job")
	json, _ := json.Marshal(data)
	return string(json)
}

// GetOptionsets gets all optionsets
func GetOptionsets() (optionsets []amt.Optionset) {
	db.Select(&optionsets, "SELECT * FROM optionset")
	return
}

// GetHosts gets all hosts
func GetHosts() (hosts []Host) {
	db.Select(&hosts, "SELECT * FROM host")
	return
}

// GetUsers gets all Users
func GetUsers() (users []User) {
	db.Select(&users, "SELECT * FROM user")
	return
}

// GetHostNamesByID gets hostnames for hosts with given IDs
func GetHostNamesByID(ids []string) (hostnames []string) {
	var myhosts []Host
	myIds := strings.Join(ids, ",")
	// FIXME, this SUCKS:
	db.Select(&myhosts, "SELECT * FROM host WHERE id IN ("+myIds+")")
	for _, host := range myhosts {
		hostnames = append(hostnames, host.Hostname)
	}
	return
}

// GetHostsByOu gets all hosts of a OU.
func GetHostsByOu(ou int) (hosts []Host) {
	db.Select(&hosts, "SELECT * FROM host WHERE ou_id = ?", ou)
	return
}

// GetOus gets all OUs
func GetOus() (ous []Ou) {
	db.Select(&ous, "SELECT * FROM ou")
	return
}

// GetLogdaysJSON gets all Logdays
func GetLogdaysJSON() string {
	type logday struct {
		ID string `json:"id"`
	}
	type logdays struct {
		Logdays []logday `json:"logdays"`
	}
	var data logdays
	db.Select(&data.Logdays, "SELECT * FROM logday")
	json, _ := json.Marshal(data)
	return string(json)
}

// GetStatelogsJSON gets all statelogs for one ou at specific day
func GetStatelogsJSON(ouid int, unixtime int) []byte {
	var data []Statelog
	hosts := GetHostsByOu(ouid)
	for _, host := range hosts {
		var hostlog []Statelog
		db.Select(&hostlog, "SELECT * FROM statelog WHERE host_id=? AND state_begin < ? ORDER BY state_begin LIMIT 1", host.ID, unixtime)
		for _, entry := range hostlog {
			data = append(data, entry)
		}
		db.Select(&hostlog, "SELECT * FROM statelog WHERE host_id=? AND state_begin < ?", host.ID, unixtime+86400)
		for _, entry := range hostlog {
			data = append(data, entry)
		}
	}
	json, _ := json.Marshal(data)
	return json
}

// GET SINGLE record by id

// GetOuJSON gets a single OU
func GetOuJSON(id int) string {
	data := Ou{}
	db.Get(&data, "SELECT * FROM ou WHERE id=?", id)
	json, _ := json.Marshal(data)
	return "{\"ou\":" + string(json) + "}"
}

// GetOu gets a single OU.
func GetOu(id int) (o Ou) {
	db.Get(&o, "SELECT * FROM ou WHERE id=?", id)
	return
}

// GetNotificationJSON gets a single notification
func GetNotificationJSON(id int) string {
	data := Notification{}
	db.Get(&data, "SELECT * FROM notification WHERE id=?", id)
	json, _ := json.Marshal(data)
	return "{\"notification\":" + string(json) + "}"
}

// GetUserJSON gets a single user
func GetUserJSON(id int) string {
	data := User{}
	db.Get(&data, "SELECT * FROM user WHERE id=?", id)
	json, _ := json.Marshal(data)
	return "{\"user\":" + string(json) + "}"
}

// GetUser gets a single user
func GetUser(name string) (u User) {
	db.Get(&u, "SELECT * FROM user WHERE name=?", name)
	return
}

// GetHostJSON gets a single host
func GetHostJSON(id int) string {
	data := Host{}
	db.Get(&data, "SELECT * FROM host WHERE id=?", id)
	json, _ := json.Marshal(data)
	return "{\"host\":" + string(json) + "}"
}

// GetLaststateJSON get laststate entry -- UNUSED
func GetLaststateJSON(id int) string {
	data := amt.Laststate{}
	db.Get(&data, "SELECT * FROM laststate WHERE id=?", id)
	json, _ := json.Marshal(data)
	return "{\"laststate\":" + string(json) + "}"
}

// GetOptionsetJSON gets a single optionset
func GetOptionsetJSON(id int) string {
	data := amt.Optionset{}
	db.Get(&data, "SELECT * FROM optionset WHERE id=?", id)
	json, _ := json.Marshal(data)
	return "{\"optionset\":" + string(json) + "}"
}

// GetOptionset gets a single optionset
func GetOptionset(id int) (o amt.Optionset) {
	db.Get(&o, "SELECT * FROM optionset WHERE id=?", id)
	return
}

// GetJobJSON gets a single job
func GetJobJSON(id int) string {
	data := Job{}
	db.Get(&data, "SELECT * FROM job WHERE id=?", id)
	json, _ := json.Marshal(data)
	return "{\"job\":" + string(json) + "}"
}

// GetScheduledJobs gets all scheduled jobs for a given weekday and minute of day.
func GetScheduledJobs(weekDay int, minuteOfDay int) (myjobs []Job) {
	db.Select(&myjobs, "SELECT * FROM job WHERE job_type=2 AND repeat_days & ? == ? AND start_time = ?", weekDay, weekDay, minuteOfDay)
	return
}

// DELETE

// DeleteHost deletes a single host
func DeleteHost(id int) (string, bool) {
	_, err := db.Exec("DELETE FROM host WHERE id=?", id)
	if err != nil {
		log.Printf("Error deleting Host %d: %s", id, err)
		return `{"errors":[{"detail": "` + err.Error() + `"}]}`, false
	}
	return "{}", true
}

// DeleteOu deletes a single host
func DeleteOu(id int) (string, bool) {
	_, err := db.Exec("DELETE FROM ou WHERE id=?", id)
	if err != nil {
		log.Printf("Error deleting OU %d: %s", id, err)
		return `{"errors":[{"detail": "` + err.Error() + `"}]}`, false
	}
	return "{}", true
}

// DeleteOptionset deletes a single host
func DeleteOptionset(id int) (string, bool) {
	_, err := db.Exec("DELETE FROM optionset WHERE id=?", id)
	if err != nil {
		log.Printf("Error deleting Optionset %d: %s", id, err)
		return `{"errors":[{"detail": "` + err.Error() + `"}]}`, false
	}
	return "{}", true
}

// DeleteUser deletes a single user
func DeleteUser(id int) (string, bool) {
	_, err := db.Exec("DELETE FROM user WHERE id=?", id)
	if err != nil {
		log.Printf("Error deleting User %d: %s", id, err)
		return `{"errors":[{"detail": "` + err.Error() + `"}]}`, false
	}
	return "{}", true
}

// DeleteJob deletes a single job
func DeleteJob(id int) (string, bool) {
	_, err := db.Exec("DELETE FROM job WHERE id=?", id)
	if err != nil {
		log.Printf("Error deleting Job %d: %s", id, err)
		return `{"errors":[{"detail": "` + err.Error() + `"}]}`, false
	}
	return "{}", true
}

// CREATE

// InsertHost inserts a single host to DB
func InsertHost(body io.ReadCloser) string {
	decoder := json.NewDecoder(body)
	type singleHost struct {
		Host newHost `json:"host"`
	}
	var s singleHost
	err := decoder.Decode(&s)
	if err != nil {
		panic(err)
	}
	ouID, err := strconv.Atoi(s.Host.OuID)
	if err != nil {
		panic(err)
	}
	q, err := db.Exec("INSERT INTO host (ou_id,hostname,enabled) VALUES (?,?,?)", ouID, s.Host.Hostname, 1)
	if err == nil {
		id, _ := q.LastInsertId()
		return GetHostJSON(int(id))
	}
	log.Printf("DB ERROR @ insert host: %s", err)
	return "{}"
}

// InsertUser inserts a user record
func InsertUser(u User) {
	db.Exec("INSERT INTO user (name,fullname,password,passsalt,ou_id) VALUES (?,?,?,?,1)",
		u.Name, u.Fullname, u.Password, u.Passsalt)
}

// InsertNotification inserts a user record
func InsertNotification(ntype string, message string) {
	// hack: user_id refs valid user but GUI doesn't give it.
	users := GetUsers()
	userid := users[0].ID
	db.Exec("INSERT INTO notification (ntype,message,user_id) VALUES (?,?,?)",
		ntype, message, userid)
}

// InsertOu creates a single org unit
func InsertOu(body io.ReadCloser) string {
	decoder := json.NewDecoder(body)
	// emberJS sends some values with incorrect type. work-around...:
	type emberOu struct {
		ID          int     `json:"id"`
		ParentID    string  `json:"parent_id" db:"parent_id"`       // int
		OptionsetID string  `json:"optionset_id" db:"optionset_id"` // int
		Name        string  `json:"name"`
		Description string  `json:"description"`
		IdlePower   float32 `json:"idle_power" db:"idle_power"`
		Logging     bool    `json:"logging"` // int
	}
	type singleOu struct {
		Ou emberOu `json:"ou"`
	}
	var submittedOu singleOu
	decoder.Decode(&submittedOu)
	body.Close()
	var ouToSave Ou
	ouToSave.Name = submittedOu.Ou.Name
	ouToSave.Description = submittedOu.Ou.Description
	ouToSave.IdlePower = submittedOu.Ou.IdlePower
	if submittedOu.Ou.Logging {
		ouToSave.Logging = 1
	} else {
		ouToSave.Logging = 0
	}
	parentID, _ := strconv.Atoi(submittedOu.Ou.ParentID)
	ouToSave.ParentID = &parentID
	optionsetID, _ := strconv.Atoi(submittedOu.Ou.OptionsetID)
	ouToSave.OptionsetID = &optionsetID
	q, _ := db.Exec("INSERT INTO ou (parent_id, optionset_id, name, description, idle_power, logging) "+
		"VALUES (?,?,?,?,?,?)", ouToSave.ParentID, ouToSave.OptionsetID, ouToSave.Name,
		ouToSave.Description, ouToSave.IdlePower, ouToSave.Logging)
	// return last inserted OU
	id, _ := q.LastInsertId()
	return GetOuJSON(int(id))
}

// InsertOptionset creates an Optionset
func InsertOptionset(body io.ReadCloser) string {
	decoder := json.NewDecoder(body)
	var submittedOptionset singleOptionset
	e := decoder.Decode(&submittedOptionset)
	body.Close()
	if e != nil {
		log.Printf("ERR: %s", e)
		return "{}"
	}
	submitted := submittedOptionset.Optionset

	var opt amt.Optionset
	opt.Name = submitted.Name
	opt.Description = submitted.Description
	opt.OptPassfile = submitted.OptPassfile
	opt.OptCacertfile = submitted.OptCacertfile
	timeout, _ := strconv.Atoi(submitted.OptTimeout)
	opt.OptTimeout = timeout
	if submitted.SwScan22 {
		opt.SwScan22 = 1
	}
	if submitted.SwScan3389 {
		opt.SwScan3389 = 1
	}
	if submitted.SwUseTLS {
		opt.SwUseTLS = 1
	}
	if submitted.SwSkipcertchk {
		opt.SwSkipcertchk = 1
	}

	fields := "name,description,sw_scan22,sw_scan3389,sw_usetls," +
		"sw_skipcertchk,opt_timeout,opt_passfile,opt_cacertfile"
	q, _ := db.Exec("INSERT INTO optionset ("+fields+") VALUES (?,?,?,?,?,?,?,?,?)",
		opt.Name, opt.Description, opt.SwScan22, opt.SwScan3389, opt.SwUseTLS,
		opt.SwSkipcertchk, opt.OptTimeout, opt.OptPassfile, opt.OptCacertfile)
	id, _ := q.LastInsertId()
	return GetOptionsetJSON(int(id))
}

// InsertStatelog adds a record in statelog table
func InsertStatelog(hostid int, http int, amt int, port int) {
	db.Exec("INSERT INTO statelog (host_id, state_http, state_amt, open_port) VALUES (?,?,?,?)",
		hostid, http, amt, port)
}

// InsertJob inserts a (scheduled) job record
func InsertJob(j Job) string {
	// hack: user_id refs valid user but GUI doesn't give it.
	users := GetUsers()
	userid := users[0].ID
	q, e := db.Exec("INSERT INTO job (job_type,user_id,amtc_cmd,amtc_delay,ou_id,start_time,repeat_days,description) VALUES (?,?,?,?,?,?,?,?)",
		j.JobType, userid, j.AmtcCmd, j.AmtcDelay, j.OuID, j.StartTime, j.RepeatDays, j.Description)
	if e != nil {
		log.Printf("New scheduled job error: %s", e)
		return "{}"
	}
	id, _ := q.LastInsertId()
	return GetJobJSON(int(id))
}

// UPDATE

// UpdateJob updates a (scheduled) job record
func UpdateJob(j Job) string {
	_, e := db.Exec("UPDATE job SET job_type=?, amtc_cmd=?, amtc_delay=?, ou_id=?, start_time=?, repeat_days=?, description=? WHERE id=?",
		j.JobType, j.AmtcCmd, j.AmtcDelay, j.OuID, j.StartTime, j.RepeatDays, j.Description, j.ID)
	if e != nil {
		log.Printf("E: %s", e.Error())
	}
	return GetJobJSON(j.ID)
}

// UpdateOu updates a OU record
func UpdateOu(id int, body io.ReadCloser) string {
	decoder := json.NewDecoder(body)
	// emberJS sends some values with incorrect type. work-around...:
	type emberOu struct {
		ID          int     `json:"id"`
		ParentID    string  `json:"parent_id" db:"parent_id"`       // int
		OptionsetID string  `json:"optionset_id" db:"optionset_id"` // int
		Name        string  `json:"name"`
		Description string  `json:"description"`
		IdlePower   float32 `json:"idle_power" db:"idle_power"`
		Logging     bool    `json:"logging"` // int
	}
	type singleOu struct {
		Ou emberOu `json:"ou"`
	}
	var submittedOu singleOu
	decoder.Decode(&submittedOu)
	body.Close()
	var ouToSave Ou
	ouToSave.Name = submittedOu.Ou.Name
	ouToSave.Description = submittedOu.Ou.Description
	ouToSave.IdlePower = submittedOu.Ou.IdlePower
	if submittedOu.Ou.Logging {
		ouToSave.Logging = 1
	} else {
		ouToSave.Logging = 0
	}
	parentID, _ := strconv.Atoi(submittedOu.Ou.ParentID)
	ouToSave.ParentID = &parentID
	optionsetID, _ := strconv.Atoi(submittedOu.Ou.OptionsetID)
	ouToSave.OptionsetID = &optionsetID
	db.Exec("UPDATE ou SET parent_id=?, optionset_id=?, name=?, description=?, idle_power=?, logging=? "+
		"WHERE id=?", ouToSave.ParentID, ouToSave.OptionsetID, ouToSave.Name,
		ouToSave.Description, ouToSave.IdlePower, ouToSave.Logging, id)
	return GetOuJSON(int(id))
}

// UpdateOptionset updates an Optionset
func UpdateOptionset(id int, body io.ReadCloser) string {
	decoder := json.NewDecoder(body)
	var submittedOptionset singleOptionset
	decoder.Decode(&submittedOptionset)
	body.Close()
	submitted := submittedOptionset.Optionset

	var opt amt.Optionset
	opt.Name = submitted.Name
	opt.Description = submitted.Description
	opt.OptPassfile = submitted.OptPassfile
	opt.OptCacertfile = submitted.OptCacertfile
	timeout, _ := strconv.Atoi(submitted.OptTimeout)
	opt.OptTimeout = timeout
	if submitted.SwScan22 {
		opt.SwScan22 = 1
	}
	if submitted.SwScan3389 {
		opt.SwScan3389 = 1
	}
	if submitted.SwUseTLS {
		opt.SwUseTLS = 1
	}
	if submitted.SwSkipcertchk {
		opt.SwSkipcertchk = 1
	}

	db.Exec("UPDATE optionset SET name=?, description=?, sw_scan22=?, sw_scan3389=?, "+
		"sw_usetls=?, sw_skipcertchk=?, opt_timeout=?, opt_passfile=?, opt_cacertfile=? "+
		"WHERE id=?",
		opt.Name, opt.Description, opt.SwScan22, opt.SwScan3389, opt.SwUseTLS,
		opt.SwSkipcertchk, opt.OptTimeout, opt.OptPassfile, opt.OptCacertfile, id)
	return GetOptionsetJSON(id)
}
