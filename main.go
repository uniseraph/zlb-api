package main

import (
	"os"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/zanecloud/zlb/api/daemon"
	"github.com/zanecloud/zlb/api/opts"
)

var (
	Version   string
	GitCommit string
	BuildTime string
)

func main() {

	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "zanecloud zlb api"
	app.Version = Version
	//	app.Author = "zhengtao.wuzt"
	//	app.Email = "zhengtao.wuzt@gmail.com"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "log-level, l",
			Value:  "info",
			EnvVar: "LOG_LEVEL",
			Usage:  "Log level (options: debug, info, warn, error, fatal, panic)",
		},
	}

	app.Before = func(c *cli.Context) error {
		logrus.SetOutput(os.Stderr)
		level, err := logrus.ParseLevel(c.String("log-level"))
		if err != nil {
			logrus.Fatalf(err.Error())
		}
		logrus.SetLevel(level)
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  "start",
			Usage: "start a zlb api ",
			Flags: []cli.Flag{

				cli.StringFlag{
					Name:   "consul-addr",
					Value:  "localhost:8500",
					EnvVar: "CONSUL_ADDR",
					Usage:  "consul addr",
				},
				cli.StringFlag{
					Name:   "addr",
					EnvVar: "ZLB_ADDR",
					Value:  "localhost:6300",
					Usage:  "zlb addr",
				},
			},
			Action: startCommand,
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func startCommand(cli *cli.Context) {

	opts := opts.Options{}
	opts.Consul = cli.String("consul-addr")
	opts.Address = cli.String("addr")

	daemon.Run(opts)

}
