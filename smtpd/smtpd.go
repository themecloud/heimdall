package smtpd

import (
	"bitbucket.org/chrj/smtpd"
	"github.com/abo/rerate"
	"github.com/mattbaird/gochimp"

	"github.com/themecloud/heimdall/config"
	"github.com/themecloud/heimdall/outputs/mandrill"
)

// NewSMTP TODO
func NewSMTP(c *config.Config, sendLimiter *rerate.Limiter, spamLimiter *rerate.Limiter) (*smtpd.Server, error) {
	mandrillAPI, err := gochimp.NewMandrill("")
	if err != nil {
		return &smtpd.Server{}, err
	}
	mailHandler := mandrill.MakeMailHandler(mandrillAPI, sendLimiter, spamLimiter)
	heloChecker := makeHeloChecker(sendLimiter, spamLimiter)
	return &smtpd.Server{
		WelcomeMessage: "Heimdall SMTP Server",
		Handler:        mailHandler,
		HeloChecker:    heloChecker,
	}, nil
}

func makeHeloChecker(sendLimiter *rerate.Limiter, spamLimiter *rerate.Limiter) func(smtpd.Peer, string) error {
	return func(peer smtpd.Peer, heloName string) error {
		if err := sendLimiter.Inc(heloName); err != nil {
			log.WithFields(log.Fields{
				"heloName": heloName,
				"error":    err,
			}).Warn("Can't increment send")
		}

		if exc, _ := sendLimiter.Exceeded(heloName); exc {
			log.WithFields(log.Fields{
				"rateLimit": "send",
				"peer":      peer,
			}).Warn("rateLimit exceeded")
			return smtpd.Error{Code: 451, Message: "Rate Limit exceeded"}
		}
		if exc, _ := spamLimiter.Exceeded(heloName); exc {
			log.WithFields(log.Fields{
				"rateLimit": "spam",
				"peer":      peer,
			}).Warn("rateLimit exceeded")
			return smtpd.Error{Code: 451, Message: "Rate Limit exceeded"}
		}
		return nil
	}
}
