package main

import (
	"bytes"

	"net/mail"
	"os"
	"time"

	"bitbucket.org/chrj/smtpd"
	log "github.com/Sirupsen/logrus"
	limiter "github.com/lysu/go-rate-limiter"
	"github.com/mattbaird/gochimp"
	"github.com/urfave/cli"
)

var (
	name    = "Heimdall"
	version = "0.0.0"

	mandrillAPI *gochimp.MandrillAPI
	allowSend   limiter.Allow
	allowSpam   limiter.Allow

	rateSendTime  = 1 * time.Minute
	rateSendLimit int

	rateSpamTime  = 24 * time.Hour
	rateSpamLimit = 5
)

func mailHandler(peer smtpd.Peer, env smtpd.Envelope) error {
	if _, err := mail.ReadMessage(bytes.NewReader(env.Data)); err != nil {
		return err
	}

	response, err := mandrillAPI.MessageSendRaw(string(env.Data), env.Recipients, gochimp.Recipient{Email: env.Sender}, false)
	if err != nil {
		log.WithFields(log.Fields{
			"peer":  peer,
			"error": err,
		}).Info("Error sending message")
		return smtpd.Error{Code: 451, Message: "Error with Remote API"}
	}
	if response[0].Status == "rejected" && response[0].RejectedReason == "spam" {
		log.WithFields(log.Fields{
			"peer":           peer,
			"RejectedReason": response[0].RejectedReason,
		}).Info("Message filtered as SPAM")
		allowSpam(peer.HeloName)
		return smtpd.Error{Code: 451, Message: "Spam filtered, increment rate limit"}
	}

	return nil
}

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

	log.WithFields(log.Fields{
		"limit": rateSendLimit,
		"time":  rateSendTime,
	}).Debug("rateLimit send")
	allowSend = limiter.RateLimiter(limiter.UseMemory(), limiter.BucketLimit(rateSendTime, rateSendLimit))()

	log.WithFields(log.Fields{
		"limit": rateSpamLimit,
		"time":  rateSpamTime,
	}).Debug("rateLimit spam")
	allowSpam = limiter.RateLimiter(limiter.UseMemory(), limiter.BucketLimit(rateSpamTime, rateSpamLimit))()

	var err error
	mandrillAPI, err = gochimp.NewMandrill(c.String("apikey"))
	if err != nil {
		return err
	}

	server := &smtpd.Server{
		WelcomeMessage:    "Heimdall SMTP Server",
		Handler:           mailHandler,
		ConnectionChecker: connectionChecker,
	}

	if err := server.ListenAndServe(c.String("listen")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

func connectionChecker(peer smtpd.Peer) error {
	if peer.HeloName == "" {
		log.WithFields(log.Fields{
			"peer": peer,
		}).Warn("No HELO NAME provided")
		//return smtpd.Error{Code: 501, Message: "No HELO NAME provided"}
	}
	if !allowSend(peer.HeloName) {
		log.WithFields(log.Fields{
			"rateLimit": "send",
			"peer":      peer,
		}).Warn("rateLimit exceeded")
		return smtpd.Error{Code: 451, Message: "Rate Limit exceeded"}
	}
	if !allowSpam(peer.HeloName) {
		log.WithFields(log.Fields{
			"rateLimit": "spam",
			"peer":      peer,
		}).Warn("rateLimit exceeded")
		return smtpd.Error{Code: 451, Message: "Rate Limit exceeded"}
	}
	return nil
}
