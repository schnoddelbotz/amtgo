// amtgo can be used to control AMT-enabled PCs via command-line.
// It can also act as server for a web GUI, which supports scheduled tasks.
package main

import (
	"fmt"
	"os"
	"runtime"

	"gopkg.in/urfave/cli.v2"

	"github.com/schnoddelbotz/amtgo/amt"
	"github.com/schnoddelbotz/amtgo/database"
	"github.com/schnoddelbotz/amtgo/scheduler"
	"github.com/schnoddelbotz/amtgo/webserver"
)

// AppVersion is defined at build time using ldflags
var AppVersion string
var cliOptions amt.Optionset

func main() {
	webserver.AppVersion = Version()
	// override cli -version shortcut (for -verbose); use -V instead
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "version",
		Aliases: []string{"V"},
		Usage:   "print amtgo version",
	}

	app := &cli.App{
		Version: Version(),
		Name:    "amtgo",
		Usage: "Intel AMT & WS-Man OOB mass management tool\n   " +
			"https://github.com/schnoddelbotz/amtc/amtgo",
		EnableShellCompletion: true,

		Flags: []cli.Flag{
			// GLOBAL flags
			&cli.BoolFlag{
				Name:        "verbose",
				Aliases:     []string{"v"},
				Usage:       "produce verbose output",
				Value:       false,
				Destination: &amt.Verbose,
			},
			// INFO / CONTROL flags
			&cli.IntFlag{
				Name:        "wait",
				Value:       10,
				Aliases:     []string{"w"},
				Usage:       "CLI: wait/timeout for cURL requests",
				Destination: &cliOptions.OptTimeout,
			},
			&cli.IntFlag{
				Name:        "delay",
				Value:       1500,
				Aliases:     []string{"d"},
				Usage:       "CLI: delay between non-info commands in ms",
				Destination: &cliOptions.CliDelay,
			},
			&cli.BoolFlag{
				Name:        "tls",
				Value:       false,
				Aliases:     []string{"t"},
				Usage:       "CLI: use TLS (tcp port 16993)",
				Destination: &cliOptions.CliUseTLS,
			},
			&cli.BoolFlag{
				Name:        "no-verify",
				Value:       false,
				Aliases:     []string{"n"},
				Usage:       "CLI: disable TLS cert verification",
				Destination: &cliOptions.CliSkipcertchk,
			},
			&cli.StringFlag{
				Name:        "username",
				Value:       "admin",
				Aliases:     []string{"u"},
				Usage:       "CLI: AMT username",
				Destination: &cliOptions.Username,
				EnvVars:     []string{"AMT_USER"},
			},
			&cli.StringFlag{
				Name:        "password",
				Aliases:     []string{"p"},
				Usage:       "CLI: password file for AMT user or set",
				Destination: &cliOptions.Password,
				EnvVars:     []string{"AMT_PASSWORD"},
			},
			&cli.StringFlag{
				Name:        "cacert-file",
				Aliases:     []string{"c"},
				Usage:       "CLI: CA certificate file for TLS",
				Destination: &cliOptions.OptCacertfile,
			},
		},

		Commands: []*cli.Command{
			{
				Name:    "server",
				Aliases: []string{"s"},
				Usage:   "amtc-web server",
				Action: func(c *cli.Context) error {
					go scheduler.ScheduledJobsRunloop(amt.Verbose)
					go scheduler.MonitoringRunloop(amt.Verbose)
					webserver.Run(amt.Verbose)
					return nil
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "dbdriver",
						Value:       "sqlite3",
						Aliases:     []string{"d"},
						Usage:       "Database driver: sqlite3 or mysql",
						Destination: &database.DbDriver,
						EnvVars:     []string{"DB_DRIVER"},
					},
					&cli.StringFlag{
						Name:        "dbfile",
						Value:       "amtgo.db",
						Aliases:     []string{"F"},
						Usage:       "SQLite database file",
						Destination: &database.DbFile,
						EnvVars:     []string{"DB_FILE"},
					},
					&cli.StringFlag{
						Name:        "dbName",
						Value:       "amtgo",
						Aliases:     []string{"D"},
						Usage:       "MySQL database name",
						Destination: &database.DbName,
						EnvVars:     []string{"DB_NAME"},
					},
					&cli.StringFlag{
						Name:        "dbHost",
						Value:       "localhost",
						Aliases:     []string{"H"},
						Usage:       "MySQL database host",
						Destination: &database.DbHost,
						EnvVars:     []string{"DB_HOST"},
					},
					&cli.StringFlag{
						Name:        "dbUser",
						Value:       "amtgo",
						Aliases:     []string{"U"},
						Usage:       "MySQL database user name",
						Destination: &database.DbUser,
						EnvVars:     []string{"DB_USER"},
					},
					&cli.StringFlag{
						Name:        "dbPassword",
						Value:       "",
						Aliases:     []string{"P"},
						Usage:       "MySQL database password",
						Destination: &database.DbPassword,
						EnvVars:     []string{"DB_PASSWORD"},
					},
					&cli.StringFlag{
						Name:        "dbPort",
						Value:       "3306",
						Aliases:     []string{"p"},
						Usage:       "MySQL database port",
						Destination: &database.DbPort,
						EnvVars:     []string{"DB_PORT"},
					},

					&cli.BoolFlag{
						Name:        "tls",
						Value:       false,
						Aliases:     []string{"t"},
						Usage:       "use TLS for amtc-web server",
						Destination: &webserver.HttpdUseTLS,
					},
					&cli.StringFlag{
						Name:        "certpath",
						Value:       ".",
						Aliases:     []string{"c"},
						Usage:       "path to cert.pem/key.pem for TLS",
						Destination: &webserver.TLSCertDir,
					},
					&cli.StringFlag{
						Name:        "listenaddr",
						Value:       ":8080",
						Aliases:     []string{"l"},
						Usage:       "IP:PORT to listen on",
						Destination: &webserver.ListenAddr,
					},
				},

				Subcommands: []*cli.Command{
					{
						Name:    "createUser",
						Aliases: []string{"u"},
						Usage:   "create amtc-web user",
						Action: func(c *cli.Context) error {
							webserver.CreateUserDialog()
							return nil
						},
					},
				},
			},

			{
				Name:    "info",
				Aliases: []string{"i"},
				Usage:   "AMT: query power state",
				Action: func(c *cli.Context) error {
					amt.CliCommand(amt.CmdInfo, c.Args().Slice(), cliOptions)
					return nil
				},
			},

			{
				Name:    "control",
				Aliases: []string{"c"},
				Usage:   "AMT: control power state",
				Subcommands: []*cli.Command{
					{
						Name:    "powerup",
						Aliases: []string{"u"},
						Usage:   "AMT power up given hosts",
						Action: func(c *cli.Context) error {
							amt.CliCommand(amt.CmdUp, c.Args().Slice(), cliOptions)
							return nil
						},
					},
					{
						Name:    "powerdown",
						Aliases: []string{"d"},
						Usage:   "AMT power down given hosts",
						Action: func(c *cli.Context) error {
							amt.CliCommand(amt.CmdDown, c.Args().Slice(), cliOptions)
							return nil
						},
					},
					{
						Name:    "reset",
						Aliases: []string{"r"},
						Usage:   "AMT reset given hosts",
						Action: func(c *cli.Context) error {
							amt.CliCommand(amt.CmdReset, c.Args().Slice(), cliOptions)
							return nil
						},
					},
					{
						Name:    "reboot",
						Aliases: []string{"b"},
						Usage:   "AMT graceful reboot (AMT 9.0+ / Windows)",
						Action: func(c *cli.Context) error {
							amt.CliCommand(amt.CmdReboot, c.Args().Slice(), cliOptions)
							return nil
						},
					},
					{
						Name:    "shutdown",
						Aliases: []string{"s"},
						Usage:   "AMT graceful shutdown (AMT 9.0+ / Windows)",
						Action: func(c *cli.Context) error {
							amt.CliCommand(amt.CmdShutdown, c.Args().Slice(), cliOptions)
							return nil
						},
					},
				},
			},

			{
				Name:    "modify",
				Aliases: []string{"m"},
				Usage:   "AMT: modify host-side configuration",
				Subcommands: []*cli.Command{
					{
						Name:  "webui",
						Usage: "enable/disable AMT web UI",
						Subcommands: []*cli.Command{
							{
								Name:  "enable",
								Usage: "enable AMT web UI",
								Action: func(c *cli.Context) error {
									amt.CliCommand(amt.CmdWebEnable, c.Args().Slice(), cliOptions)
									return nil
								},
							},
							{
								Name:  "disable",
								Usage: "disable AMT web UI",
								Action: func(c *cli.Context) error {
									amt.CliCommand(amt.CmdWebDisable, c.Args().Slice(), cliOptions)
									return nil
								},
							},
						},
					},
					{
						Name:  "ping",
						Usage: "enable/disable AMT ping replies in power off state",
						Subcommands: []*cli.Command{
							{
								Name:  "enable",
								Usage: "enable AMT ping replies in power off state",
								Action: func(c *cli.Context) error {
									amt.CliCommand(amt.CmdPingEnable, c.Args().Slice(), cliOptions)
									return nil
								},
							},
							{
								Name:  "disable",
								Usage: "disable AMT ping replies in power off state",
								Action: func(c *cli.Context) error {
									amt.CliCommand(amt.CmdPingDisable, c.Args().Slice(), cliOptions)
									return nil
								},
							},
						},
					},
					{
						Name:  "sol",
						Usage: "enable/disable AMT serial-over-LAN",
						Subcommands: []*cli.Command{
							{
								Name:  "enable",
								Usage: "enable AMT serial-over-LAN",
								Action: func(c *cli.Context) error {
									amt.CliCommand(amt.CmdSolEnable, c.Args().Slice(), cliOptions)
									return nil
								},
							},
							{
								Name:  "disable",
								Usage: "disable AMT serial-over-LAN",
								Action: func(c *cli.Context) error {
									amt.CliCommand(amt.CmdSolDisable, c.Args().Slice(), cliOptions)
									return nil
								},
							},
						},
					},
				},
			},
		},

		Action: func(c *cli.Context) error {
			// default action if no command given
			cli.ShowAppHelp(c)
			return nil
		},
	}

	app.Run(os.Args)
}

// Version returns current amtgo version as string
func Version() string {
	if len(AppVersion) == 0 {
		AppVersion = "0.0.0-dev"
	}

	return fmt.Sprintf("%s (Go runtime %s)", AppVersion, runtime.Version())
}
