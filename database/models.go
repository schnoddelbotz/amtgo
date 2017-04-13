package database

const (
	// NotificationTypeUser maps to user symbol
	NotificationTypeUser = "user"
	// NotificationTypePowerOff maps to power-off symbol
	NotificationTypePowerOff = "toggle-off"
	// NotificationTypePowerOn maps to power-on symbol
	NotificationTypePowerOn = "toggle-on"
	// NotificationTypeWarning maps to warning symbol
	NotificationTypeWarning = "warning"
	// NotificationTypeComment maps to comment bubble symbol
	NotificationTypeComment = "comment"
)

// Ou describes an organizational unit (e.g. room)
type Ou struct {
	ID          int     `json:"id"`
	ParentID    *int    `json:"parent_id" db:"parent_id"`
	OptionsetID *int    `json:"optionset_id" db:"optionset_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	IdlePower   float32 `json:"idle_power" db:"idle_power"`
	Logging     int     `json:"logging"`
}

// Ous list multiple OUs for ember
type Ous struct {
	Ous []Ou `json:"ous"`
}

// Notification is a GUI notification (e.g. job started, completed...)
type Notification struct {
	ID      int    `json:"id"`
	Tstamp  int    `json:"tstamp"`
	UserID  int    `json:"user_id" db:"user_id"`
	Ntype   string `json:"ntype"`
	Message string `json:"message"`
}

// Notifications array for ember
type Notifications struct {
	Notifications []Notification `json:"notifications"`
}

// User represents a GUI user
type User struct {
	ID         int    `json:"id"`
	OuID       int    `json:"ou_id" db:"ou_id"`
	IsEnabled  int    `json:"is_enabled" db:"is_enabled"`
	IsAdmin    int    `json:"is_admin" db:"is_admin"`
	CanControl int    `json:"can_control" db:"can_control"`
	Name       string `json:"name"`
	Fullname   string `json:"fullname"`
	Password   string `json:"-"`
	Passsalt   string `json:"-"`
}

// Users array for ember
type Users struct {
	Users []User `json:"users"`
}

// Host is a AMT enabled client to be controlled.
type Host struct {
	ID       int    `json:"id"`
	OuID     int    `json:"ou_id" db:"ou_id"`
	Hostname string `json:"hostname"`
	Enabled  int    `json:"enabled"`
}

// Hosts array for ember
type Hosts struct {
	Hosts []Host `json:"hosts"`
}

// hack, for adding new hosts... ember-data sending ou as string.
type newHost struct {
	ID       int    `json:"id"`
	OuID     string `json:"ou_id" db:"ou_id"`
	Hostname string `json:"hostname"`
	Enabled  bool   `json:"enabled"`
}

// Statelog -- unused?
type Statelog struct {
	HostID     int `json:"host_id" db:"host_id"`
	StateBegin int `json:"state_begin" db:"state_begin"`
	OpenPort   int `json:"open_port" db:"open_port"`
	StateAMT   int `json:"state_amt" db:"state_amt"`
	StateHTTP  int `json:"state_http" db:"state_http"`
}

// Statelogs wraps statelogs for json/emberjs
type Statelogs struct {
	Statelogs []Statelog `json:"statelogs"`
}

// Job is a scheduled job
type Job struct {
	ID             int      `json:"id"`
	JobType        int      `json:"job_type" db:"job_type"`
	JobStatus      int      `json:"job_status" db:"job_status"`
	UserID         int      `json:"user_id" db:"user_id"`
	AmtcCmd        *string  `json:"amtc_cmd" db:"amtc_cmd"`
	AmtcDelay      *float64 `json:"amtc_delay" db:"amtc_delay"`
	AmtcBootdevice *string  `json:"amtc_bootdevice" db:"amtc_bootdevice"`
	AmtcHosts      *string  `json:"amtc_hosts" db:"amtc_hosts"`
	OuID           *int     `json:"ou_id" db:"ou_id"`
	StartTime      int      `json:"start_time" db:"start_time"`
	RepeatInterval *int     `json:"repeat_interval" db:"repeat_interval"`
	RepeatDays     *int     `json:"repeat_days" db:"repeat_days"`
	LastStarted    *int     `json:"last_started" db:"last_started"`
	LastDone       *int     `json:"last_done" db:"last_done"`
	ProcPid        *int     `json:"proc_pid" db:"proc_pid"`
	Description    *string  `json:"description"`
}

// Jobs for ember
type Jobs struct {
	Jobs []Job `json:"jobs"`
}

// emberJS sends some values with incorrect type. work-around...:
type emberOptionset struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	SwV5          bool   `json:"sw_v5"` // ints...
	SwDash        bool   `json:"sw_dash"`
	SwScan22      bool   `json:"sw_scan22"`
	SwScan3389    bool   `json:"sw_scan3389"`
	SwUseTLS      bool   `json:"sw_usetls"`
	SwSkipcertchk bool   `json:"sw_skipcertchk"`
	OptTimeout    string `json:"opt_timeout"` // int
	OptPassfile   string `json:"opt_passfile"`
	OptCacertfile string `json:"opt_cacertfile"`
}
type singleOptionset struct {
	Optionset emberOptionset `json:"optionset"`
}
