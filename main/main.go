package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

var logger = logging.MustGetLogger("main")

func main() {

	app := &cli.App{
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "orgs",
				Value: 10,
				Usage: "number of organizations (all Idemix)",
			},
			&cli.IntFlag{
				Name:  "users",
				Value: 10,
				Usage: "number of users per organization",
			},
			&cli.IntFlag{
				Name:  "peers",
				Value: 5,
				Usage: "number of peers",
			},
			&cli.IntFlag{
				Name:  "epoch",
				Value: 60,
				Usage: "length of an epoch in seconds",
			},
			&cli.IntFlag{
				Name:  "seed",
				Value: 1305,
				Usage: "seed for PRG",
			},
			&cli.IntFlag{
				Name:  "bandwidth",
				Value: 1024 * 1024, // 1 MB/s
				Usage: "bandwidth in bytes per second",
			},
			&cli.BoolFlag{
				Name:  "revoke",
				Value: true,
				Usage: "whether to do occasional revocations",
			},
			&cli.BoolFlag{
				Name:  "audit",
				Value: true,
				Usage: "whether to do auditing of all transactions at the end",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Value: true,
				Usage: "verbose output",
			},
			&cli.GenericFlag{
				Name: "idemix",
				Value: &EnumValue{
					Enum:    []string{"none", "old", "new"},
					Default: "new",
				},
				Usage: "version of idemix: none, old or new",
			},
		},
		Name:     "simulator",
		Usage:    "runs Fabric Idemix simulation",
		Version:  "v0.0.1",
		Compiled: time.Now(),
		Authors: []*cli.Author{
			&cli.Author{
				Name:  "Dmytro Bogatov",
				Email: "dmytro@dbogatov.org",
			},
		},
		Copyright: "(c) 2020 Dmytro Bogatov",

		Action: func(c *cli.Context) error {
			configureLogging(c.Bool("verbose"))

			os.Remove("network-log.log")
			f, err := os.OpenFile("network-log.log", os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				logger.Fatalf("error opening file: %v", err)
			}
			defer f.Close()

			log.SetOutput(f)

			return simulate(
				c.Int("orgs"),
				c.Int("users"),
				c.Int("peers"),
				c.Int("epoch"),
				c.Int("seed"),
				c.Int("bandwidth"),
				c.Bool("revoke"),
				c.Bool("audit"),
				c.String("idemix"),
			)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}
}

func configureLogging(verbose bool) {
	logging.SetFormatter(
		logging.MustStringFormatter(`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} |	 %{message}`),
	)
	levelBackend := logging.AddModuleLevel(logging.NewLogBackend(os.Stdout, "", 0))
	if verbose {
		levelBackend.SetLevel(logging.DEBUG, "")
	} else {
		levelBackend.SetLevel(logging.ERROR, "")
	}
	logging.SetBackend(levelBackend)
}

// EnumValue for CLI
type EnumValue struct {
	Enum     []string
	Default  string
	selected string
}

// Set for CLI
func (e *EnumValue) Set(value string) error {
	for _, enum := range e.Enum {
		if enum == value {
			e.selected = value
			return nil
		}
	}

	return fmt.Errorf("allowed values are %s", strings.Join(e.Enum, ", "))
}

func (e EnumValue) String() string {
	if e.selected == "" {
		return e.Default
	}
	return e.selected
}
