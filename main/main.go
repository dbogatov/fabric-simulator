package main

import (
	"log"
	"os"
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
				Name:  "bandwidth",
				Value: 1024 * 1024, // 1 MB/s
				Usage: "bandwidth in bytes per second",
			},
			&cli.IntFlag{
				Name:  "transactions",
				Value: 25,
				Usage: "total number of transactions per user",
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
			&cli.BoolFlag{
				Name:  "global",
				Value: false,
				Usage: "whether the network is global; if yes, the bandwidth applies to all connection, otherwise to a single connection",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Value: false,
				Usage: "verbose output",
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

			logger.Critical(c.Int("bandwidth"))

			sys, rootSk := MakeSystemParameters(
				c.Int("orgs"),
				c.Int("users"),
				c.Int("peers"),
				c.Int("endorsements"),
				c.Int("epoch"),
				c.Int("bandwidth"),
				c.Int("conc-endorsements"),
				c.Int("conc-validations"),
				c.Int("conc-revocations"),
				c.Int("transactions"),
				c.Bool("revoke"),
				c.Bool("audit"),
				c.Bool("global"),
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

func configureLogging(verbose bool) {
	logging.SetFormatter(
		logging.MustStringFormatter(`%{color}%{time:15:04:05.000} %{shortfunc:22s} ▶ %{level:8s} %{id:03x}%{color:reset} |	 %{message}`),
	)
	levelBackend := logging.AddModuleLevel(logging.NewLogBackend(os.Stdout, "", 0))
	if verbose {
		levelBackend.SetLevel(logging.DEBUG, "")
	} else {
		levelBackend.SetLevel(logging.INFO, "")
	}
	logging.SetBackend(levelBackend)
}
