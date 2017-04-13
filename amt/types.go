package amt

// Laststate represents state reported by AMT, plus open TCP port
type Laststate struct {
	ID          int    `json:"id"`
	HostID      int    `json:"host_id"`
	Hostname    string `json:"hostname"`
	StateBegin  int    `json:"state_begin"`
	OpenPort    int    `json:"open_port"`
	StateAMT    int    `json:"state_amt"`
	StateHTTP   int    `json:"state_http"`
	Usermessage string `json:"usermessage"` // amtgo only
}

// Laststates is array of Laststate -- for ember
type Laststates struct {
	Laststates []Laststate `json:"laststates"`
}

// Optionset for AMT queries (TLS yes/no, CertCheck, timeout...)
type Optionset struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	SwV5           int    `json:"sw_v5" db:"sw_v5"`
	SwDash         int    `json:"sw_dash" db:"sw_dash"`
	SwScan22       int    `json:"sw_scan22" db:"sw_scan22"`
	SwScan3389     int    `json:"sw_scan3389" db:"sw_scan3389"`
	SwUseTLS       int    `json:"sw_usetls" db:"sw_usetls"`
	SwSkipcertchk  int    `json:"sw_skipcertchk" db:"sw_skipcertchk"`
	OptTimeout     int    `json:"opt_timeout" db:"opt_timeout"`
	OptPassfile    string `json:"opt_passfile" db:"opt_passfile"`
	OptCacertfile  string `json:"opt_cacertfile" db:"opt_cacertfile"`
	Username       string `json:"username"` // amtgo only
	Password       string `json:"-"`        // amtgo only
	CliDelay       int    `json:"-" db:"-"`
	CliUseTLS      bool   `json:"-" db:"-"` // amtgo cli (bool) vs db (int) hack
	CliSkipcertchk bool   `json:"-" db:"-"` // amtgo cli (bool) vs db (int) hack
	CaCertData     []byte `json:"-" db:"-"` // loaded contents of OptCacertfile
}

// Optionsets is ember array of Optionset
type Optionsets struct {
	Optionsets []Optionset `json:"optionsets"`
}

type cmdinfo struct {
	IsTwoStep  bool
	CommandOne string
	CommandTwo string
}

const (
	CmdBootcfgPxe  = "BOOTCFGPXE" // https://github.com/golang/lint/issues/274
	CmdBootcfgHdd  = "BOOTCFGHDD"
	CmdInfo        = "INFO"
	CmdUp          = "UP"
	CmdDown        = "DOWN"
	CmdReset       = "RESET"
	CmdReboot      = "REBOOT"
	CmdShutdown    = "SHUTDOWN"
	CmdPingEnable  = "PINGENABLE"
	CmdPingDisable = "PINGDISABLE"
	CmdWebEnable   = "WEBENABLE"
	CmdWebDisable  = "WEBDISABLE"
	CmdSolEnable   = "SOLENABLE"
	CmdSolDisable  = "SOLDISABLE"
)

// ShortCommandMap as used by jobs / scheduled jobs via GUI / in DB
var ShortCommandMap = map[string]string{
	// FIXME: C/power_cycle is EOI only or just lost in wsman?????
	"X": CmdBootcfgPxe,
	"H": CmdBootcfgHdd,
	"U": CmdUp,
	"D": CmdDown,
	"R": CmdReset,
	"B": CmdReboot,
	"S": CmdShutdown,
}

const (
	stateUnknown                   = 0
	stateOther                     = 1
	stateOn                        = 2
	stateSleepLight                = 3
	stateSleepDeep                 = 4
	stateCycleOffSoft              = 5
	stateOffHard                   = 6
	stateHibernate                 = 7
	stateOffSoft                   = 8
	stateCycleOffHard              = 9
	stateMasterBusReset            = 10
	stateNMI                       = 11
	stateNotApplicable             = 12
	stateOffSoftGraceful           = 13
	stateOffHardGraceful           = 14
	stateMasterBusResetGraceful    = 15
	statePowerCycleOffSoftGraceful = 16
	statePowerCycleOffHardGraceful = 17
	stateDiagnosticInterrupt       = 18
)

var cmdMap = map[string]cmdinfo{
	CmdBootcfgPxe:  {true, "wsman_pxeboot", "wsman_bootconfig"},
	CmdBootcfgHdd:  {true, "wsman_hddboot", "wsman_bootconfig"},
	CmdInfo:        {true, "wsman_info", "wsman_info_step2"},
	CmdUp:          {false, "wsman_up", ""},
	CmdDown:        {false, "wsman_down", ""},
	CmdReset:       {false, "wsman_reset", ""},
	CmdReboot:      {false, "wsman_reset_graceful", ""},
	CmdShutdown:    {false, "wsman_shutdown_graceful", ""},
	CmdPingEnable:  {false, "wsman_ping_enable", ""},
	CmdPingDisable: {false, "wsman_ping_disable", ""},
	CmdWebEnable:   {false, "wsman_webui_enable", ""},
	CmdWebDisable:  {false, "wsman_webui_disable", ""},
	CmdSolEnable:   {false, "wsman_solredir_enable", ""},
	CmdSolDisable:  {false, "wsman_solredir_disable", ""},
}

var powerstateTextMap = map[int]string{
	stateUnknown:                   "Unknown",
	stateOther:                     "Other",
	stateOn:                        "On",
	stateSleepLight:                "Sleep-Light",
	stateSleepDeep:                 "Sleep-Deep",
	stateCycleOffSoft:              "Cycle-Off-Soft",
	stateOffHard:                   "Off-Hard",
	stateHibernate:                 "Hibernate",
	stateOffSoft:                   "Off-Soft",
	stateCycleOffHard:              "Cycle-Off-Hard",
	stateMasterBusReset:            "Master-Bus-Reset",
	stateNMI:                       "NMI",
	stateNotApplicable:             "Not-Applicable",
	stateOffSoftGraceful:           "Off-Soft-Graceful",
	stateOffHardGraceful:           "Off-Hard-Graceful",
	stateMasterBusResetGraceful:    "Master-Bus-Reset-Graceful",
	statePowerCycleOffSoftGraceful: "Power-Cycle-Off-Soft-Graceful",
	statePowerCycleOffHardGraceful: "Power-Cycle-Off-Hard-Graceful",
	stateDiagnosticInterrupt:       "Diagnostic-Interrupt",
}

// mapping to amtc/EOI legacy powerstates + texts
var legacyPowerstateMap = map[int]int{
	0:  16, // error
	1:  9,  // undef
	2:  0,  // on
	3:  3,  // sleep
	4:  4,  // hibernate
	5:  9,
	6:  9,
	7:  9,
	8:  5, // soft-off
	9:  9,
	10: 9,
	11: 9,
	12: 9,
	13: 9,
	14: 9,
	15: 9,
	16: 9,
	17: 9,
	18: 9,
}

var legacyPowerstateTextMap = map[int]string{
	0:  "On",
	1:  "unimplemented",
	2:  "unimplemented",
	3:  "Sleep",
	4:  "Hibernate",
	5:  "Soft-Off",
	6:  "unimplemented",
	7:  "unimplemented",
	8:  "unimplemented",
	9:  "unimplemented",
	10: "unimplemented",
	11: "unimplemented",
	12: "unimplemented",
	13: "unimplemented",
	14: "unimplemented",
	15: "unimplemented",
	16: "unimplemented",
}

var httpReturncodeTextMap = map[int]string{
	200: "OK",
	400: "BadRequest",
	401: "Unauthorized",
	403: "Forbidden",
	404: "NotFound",
	408: "Timeout",
	500: "InternalError",
	// are more used in AMT...?
}
