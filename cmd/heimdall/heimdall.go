package main

import (
	"os"
	"time"

	"github.com/themecloud/heimdall/config"
	"github.com/themecloud/heimdall/redis"
	"github.com/themecloud/heimdall/smtpd"

	log "github.com/Sirupsen/logrus"
	"github.com/abo/rerate"
	"github.com/urfave/cli"
)

var (
	name    = "Heimdall"
	version = "0.0.0"

	rateSendTime  = 1 * time.Hour
	rateSendLimit int

	rateSpamTime  = 24 * time.Hour
	rateSpamLimit = 5
)

func main() {
	app := cli.NewApp()
	app.Name = name
	app.Version = version
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Guilhem Lettron",
			Email: "guilhem@noadmin.io",
		},
	}
	app.Usage = "SMTP to mandrill gateway"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "listen, l",
			Usage: "Address to listen",
			Value: "127.0.0.1:25",
		},
		cli.StringFlag{
			Name: "output",
		},
		cli.StringFlag{
			Name:   "apikey, mandrill-key, mk",
			Usage:  "Mandrill API Key",
			EnvVar: "MANDRILL_KEY",
		},
		cli.IntFlag{
			Name:        "rateLimit, rl",
			Usage:       "# of message by minute allowed",
			Value:       10,
			Destination: &rateSendLimit,
			EnvVar:      "HEIMDALL_RATELIMIT",
		},
		cli.StringFlag{
			Name:   "redis",
			Value:  "127.0.0.1:6379",
			EnvVar: "REDIS_SERVER",
		},
		cli.StringFlag{
			Name:   "redis-password",
			EnvVar: "REDIS_PASSWORD",
		},
		cli.BoolFlag{
			Name: "verbose, V",
		},
	}

	app.Before = heimdallBefore

	app.Action = serve

	app.Run(os.Args)
}

func heimdallBefore(c *cli.Context) error {
	if c.Bool("verbose") {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

func serve(c *cli.Context) error {

	redisPool := redis.NewPool(c.String("redis"), c.String("redis-password"))

	config := config.NewConfig(c.String("output"))

	log.WithFields(log.Fields{
		"config": config,
	}).Debug("Configuration generated")

	log.WithFields(log.Fields{
		"limit": rateSendLimit,
		"time":  rateSendTime,
	}).Debug("rateLimit send")
	sendLimiter := rerate.NewLimiter(redisPool, name+":rateSend", rateSendTime, time.Minute, int64(rateSendLimit))

	log.WithFields(log.Fields{
		"limit": rateSpamLimit,
		"time":  rateSpamTime,
	}).Debug("rateLimit spam")
	spamLimiter := rerate.NewLimiter(redisPool, name+":rateSpam", rateSpamTime, time.Minute, int64(rateSpamLimit))

	server, err := smtpd.NewSMTP(config, sendLimiter, spamLimiter)
	if err != nil {
		return cli.NewExitError(err.Error(), 1)
	}

	if err := server.ListenAndServe(c.String("listen")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}
