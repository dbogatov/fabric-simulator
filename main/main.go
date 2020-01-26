package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/dbogatov/fabric-amcl/amcl"
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
			&cli.IntFlag{
				Name:  "transactions",
				Value: 1000,
				Usage: "total number of transactions",
			},
			&cli.IntFlag{
				Name:  "peer-endorsements",
				Value: 3,
				Usage: "number of concurrent endorsements a peer can do",
			},
			&cli.IntFlag{
				Name:  "peer-validations",
				Value: 3,
				Usage: "number of concurrent validations a peer can do",
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

			prg := amcl.NewRAND()
			prg.Clean()
			prg.Seed(1, []byte(strconv.Itoa(c.Int("seed"))))

			sys, rootSk := MakeSystemParameters(
				prg,
				c.Int("orgs"),
				c.Int("users"),
				c.Int("peers"),
				c.Int("bandwidth"),
				c.Int("transactions"),
				c.Int("peer-endorsements"),
				c.Int("peer-validations"),
				c.Bool("revoke"),
				c.Bool("audit"),
			)
			sysParams = *sys

			return simulate(prg, rootSk)
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
