package main

import (
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/dbogatov/dac-lib/dac"
	"github.com/dbogatov/fabric-amcl/amcl"
	"github.com/dbogatov/fabric-simulator/distributed"
	"github.com/dbogatov/fabric-simulator/helpers"
	"github.com/dbogatov/fabric-simulator/revocation"
	"github.com/dbogatov/fabric-simulator/simulator"
	"github.com/op/go-logging"
	"github.com/urfave/cli/v2"
)

var logger = logging.MustGetLogger("main")

func main() {

	commonFlags := []cli.Flag{
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
			Name:  "transactions",
			Value: 25,
			Usage: "total number of transactions per user",
		},
		&cli.IntFlag{
			Name:  "frequency",
			Value: 20,
			Usage: "max wait time in seconds for a user between transactions",
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
	}

	setSystemParameters := func(c *cli.Context, prg *amcl.RAND) (sysParams *helpers.SystemParameters, rootSk, auditSk dac.SK) {
		return helpers.MakeSystemParameters(
			logger,
			prg,
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
			c.Int("rpc-port"),
			c.String("root-address"),
			c.String("org-address"),
			c.String("revocation-address"),
			c.StringSlice("peer-addresses"),
		)
	}

	app := &cli.App{
		EnableBashCompletion: true,
		Version:              "v0.0.1",
		Authors: []*cli.Author{
			{
				Name:  "Dmytro Bogatov",
				Email: "dmytro@dbogatov.org",
			},
		},
		Copyright: "(c) 2020 Dmytro Bogatov",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "verbose",
				Value: "NOTICE",
				Usage: "verbosity level: CRIT, ERROR, WARN, NOTICE, INFO, DEBUG",
			},
			&cli.IntFlag{
				Name:  "seed",
				Value: 0x13,
				Usage: "seed to use for root PK",
			},
		},
		Name:        "fabric",
		Description: "Set of tools to do Fabric simulations",
		Before: func(c *cli.Context) error {
			configureLogging(strings.ToUpper(c.String("verbose")))

			logger.Criticalf("GOMAXPROCS: %d", runtime.GOMAXPROCS(0))

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "revocation",
				Usage: "revocation client-server simulation",
				Before: func(c *cli.Context) error {
					revocation.SetLogger(logger)
					return nil
				},
				Subcommands: []*cli.Command{
					{
						Name:  "server",
						Usage: "server part of the simulation",
						Action: func(c *cli.Context) error {
							revocation.RunServer()
							return nil
						},
					},
					{
						Name:  "client",
						Usage: "client part of the simulation",
						Flags: []cli.Flag{
							&cli.IntFlag{
								Name:  "runs",
								Value: 1000,
								Usage: "total number of requests",
							},
							&cli.IntFlag{
								Name:  "concurrent",
								Value: 100,
								Usage: "number of concurrent requests",
							},
							&cli.BoolFlag{
								Name:  "trust",
								Value: false,
								Usage: "if set, will not verify signatures",
							},
						},
						Action: func(c *cli.Context) error {
							revocation.StartRequests(
								c.Int("runs"),
								c.Int("concurrent"),
								c.Bool("trust"),
							)
							return nil
						},
					},
				},
			},
			{
				Flags: append(
					commonFlags,
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
					}),
				Name:  "simulator",
				Usage: "runs Fabric Idemix simulation tracking network statistics and crypto events",
				Action: func(c *cli.Context) error {

					simulator.SetLogger(logger)

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

					sys, rootSk, _ := setSystemParameters(c, helpers.NewRandSeed([]byte{byte(c.Int("seed"))}))

					return simulator.Simulate(rootSk, sys)
				},
			},
			{
				Flags: append(
					commonFlags,
					&cli.BoolFlag{
						Name:  "root",
						Value: false,
						Usage: "whether this instance shall run as RPC root",
					},
					&cli.BoolFlag{
						Name:  "revocation",
						Value: false,
						Usage: "whether this instance shall run as RPC revocation authority",
					},
					&cli.BoolFlag{
						Name:  "auditor",
						Value: false,
						Usage: "if set, will send call to all peers to do audit and clear ledger",
					},
					&cli.IntFlag{
						Name:  "organization",
						Value: 0,
						Usage: "if >0, this instance shall run as RPC organization with given ID",
					},
					&cli.IntFlag{
						Name:  "peer",
						Value: 0,
						Usage: "if >0, this instance shall run as RPC peer with given ID",
					},
					&cli.IntFlag{
						Name:  "user",
						Value: 0,
						Usage: "if >0, this instance shall run as RPC user with given ID",
					},
					&cli.IntFlag{
						Name:  "rpc-port",
						Value: 8000,
						Usage: "RPC port to listen to",
					},
					&cli.StringFlag{
						Name:  "root-address",
						Value: "localhost:8100",
						Usage: "the address (host:port) of the root credential authority RPC server",
					},
					&cli.StringFlag{
						Name:  "org-address",
						Value: "localhost:8200",
						Usage: "the address (host:port) of the user's organization RPC server",
					},
					&cli.StringFlag{
						Name:  "revocation-address",
						Value: "localhost:8300",
						Usage: "the address (host:port) of the revocation authority's RPC server",
					},
					&cli.StringSliceFlag{
						Name:  "peer-addresses",
						Value: cli.NewStringSlice("localhost:8400"),
						Usage: "the addresses (host:port) of the peers RPC servers",
					},
				),
				Name:  "distributed",
				Usage: "runs Fabric Idemix simulation in a fully distributed setting",
				Action: func(c *cli.Context) error {

					distributed.SetLogger(logger)

					sys, rootSk, auditSk := setSystemParameters(c, helpers.NewRandSeed([]byte{byte(c.Int("seed"))}))
					sys.Peers = len(c.StringSlice("peer-addresses"))

					return distributed.Simulate(rootSk, auditSk, sys, c.Bool("root"), c.Int("organization"), c.Int("peer"), c.Int("user"), c.Bool("revocation"), c.Bool("auditor"))
				},
			},
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
