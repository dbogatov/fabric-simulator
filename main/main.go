package main

import (
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
				Name:  "endorsements",
				Value: 2,
				Usage: "endorsement policy: number of endorsing peers per transaction",
			},
			&cli.IntFlag{
				Name:  "epoch",
				Value: 60,
				Usage: "length of an epoch in seconds",
			},
			&cli.IntFlag{
				Name:  "bandwidth-global",
				Value: 1024 * 1024, // 1 MB/s
				Usage: "global bandwidth in bytes per second",
			},
			&cli.IntFlag{
				Name:  "bandwidth-local",
				Value: 1024 * 1024 / 10, // 0.1 MB/s
				Usage: "local bandwidth in bytes per second",
			},
			&cli.IntFlag{
				Name:  "transactions",
				Value: 25,
				Usage: "total number of transactions per user",
			},
			&cli.IntFlag{
				Name:  "frequency",
				Value: 20,
				Usage: "max wait time in seconds for a user between transactions",
			},
			&cli.IntFlag{
				Name:  "conc-endorsements",
				Value: 3,
				Usage: "number of concurrent endorsements a peer can do",
			},
			&cli.IntFlag{
				Name:  "conc-validations",
				Value: 10,
				Usage: "number of concurrent validations a peer can do",
			},
			&cli.IntFlag{
				Name:  "conc-revocations",
				Value: 10,
				Usage: "number of concurrent revocations the authority can do",
			},
			&cli.BoolFlag{
				Name:  "revoke",
				Value: false,
				Usage: "whether to do occasional revocations",
			},
			&cli.BoolFlag{
				Name:  "audit",
				Value: false,
				Usage: "whether to do auditing of all transactions at the end",
			},
			&cli.StringFlag{
				Name:  "verbose",
				Value: "NOTICE",
				Usage: "verbosity level: CRIT, ERROR, WARN, NOTICE, INFO, DEBUG",
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
			configureLogging(strings.ToUpper(c.String("verbose")))

			os.Remove("network-log.json")
			f, err := os.OpenFile("network-log.json", os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				logger.Fatalf("error opening file: %v", err)
			}
			defer func() {
				log.Println("]")
				f.Close()
			}()

			log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime)) // to avoid timestamps
			log.SetOutput(f)
			log.Println("[")

			sys, rootSk := MakeSystemParameters(
				c.Int("orgs"),
				c.Int("users"),
				c.Int("peers"),
				c.Int("endorsements"),
				c.Int("epoch"),
				c.Int("bandwidth-global"),
				c.Int("bandwidth-local"),
				c.Int("conc-endorsements"),
				c.Int("conc-validations"),
				c.Int("conc-revocations"),
				c.Int("transactions"),
				c.Int("frequency"),
				c.Bool("revoke"),
				c.Bool("audit"),
			)
			sysParams = *sys

			return simulate(rootSk)
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logger.Fatal(err)
	}
}

func configureLogging(verbose string) {
	logging.SetFormatter(
		logging.MustStringFormatter(`%{color}%{time:15:04:05.000} %{shortfunc:22s} ▶ %{level:8s} %{id:03x}%{color:reset} |	 %{message}`),
	)
	levelBackend := logging.AddModuleLevel(logging.NewLogBackend(os.Stdout, "", 0))
	switch verbose {
	case "CRIT":
		levelBackend.SetLevel(logging.CRITICAL, "")
		break
	case "ERROR":
		levelBackend.SetLevel(logging.ERROR, "")
		break
	case "WARN":
		levelBackend.SetLevel(logging.WARNING, "")
		break
	case "NOTICE":
		levelBackend.SetLevel(logging.NOTICE, "")
		break
	case "INFO":
		levelBackend.SetLevel(logging.INFO, "")
		break
	case "DEBUG":
		levelBackend.SetLevel(logging.DEBUG, "")
		break
	default:
		levelBackend.SetLevel(logging.DEBUG, "")
		logger.Fatalf("Invalid verbosity level: %s", verbose)
	}
	logging.SetBackend(levelBackend)
}
