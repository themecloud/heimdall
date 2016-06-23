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
	heloChecker := mandrill.MakeHeloChecker(sendLimiter, spamLimiter)
	return &smtpd.Server{
		WelcomeMessage: "Heimdall SMTP Server",
		Handler:        mailHandler,
		HeloChecker:    heloChecker,
	}, nil
}
