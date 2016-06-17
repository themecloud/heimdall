package main

import (
	"bytes"

	"net/mail"
	"os"
	"time"

	"bitbucket.org/chrj/smtpd"
	"github.com/bsm/ratelimit"
	"github.com/mattbaird/gochimp"
	"github.com/urfave/cli"
)

var (
	Name      = "Heimdall"
	Version   = "0.0.0"
	APIKEY    string
	Rate      *ratelimit.RateLimiter
	RateLimit int
)

func mailHandler(_ smtpd.Peer, env smtpd.Envelope) error {
	if _, err := mail.ReadMessage(bytes.NewReader(env.Data)); err != nil {
		return err
	}

	mandrillAPI, err := gochimp.NewMandrill(APIKEY)

	_, err = mandrillAPI.MessageSendRaw(string(env.Data), env.Recipients, gochimp.Recipient{Email: env.Sender}, false)
	if err != nil {
		return err
	}
	return nil
}

func init() {
	Rate = ratelimit.New(RateLimit, time.Minute)
}

func main() {
	app := cli.NewApp()
	app.Name = Name
	app.Version = Version
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
			Name:        "mandrill-key, mk",
			Usage:       "Mandrill API Key",
			Destination: &APIKEY,
			EnvVar:      "MANDRILL_KEY",
		},
		cli.IntFlag{
			Name:        "rateLimit, rl",
			Usage:       "# of message by minute allowed",
			Value:       10,
			Destination: &RateLimit,
			EnvVar:      "HEIMDALL_RATELIMIT",
		},
	}

	app.Action = serve

	app.Run(os.Args)
}

func serve(c *cli.Context) error {
	server := &smtpd.Server{

		WelcomeMessage: "Heimdall SMTP Server",

		Handler:           mailHandler,
		ConnectionChecker: rateLimit,
	}

	if err := server.ListenAndServe(c.String("listen")); err != nil {
		return cli.NewExitError(err.Error(), 1)
	}
	return nil
}

func rateLimit(_ smtpd.Peer) error {
	if Rate.Limit() {
		return smtpd.Error{Code: 451, Message: "Rate Limit exceeded"}
	}
	return nil
}
