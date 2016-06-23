package mandrill

import (
	"bytes"
	"net/mail"

	"bitbucket.org/chrj/smtpd"
	log "github.com/Sirupsen/logrus"
	"github.com/abo/rerate"
	"github.com/mattbaird/gochimp"
)

func MakeMailHandler(mandrillAPI *gochimp.MandrillAPI, sendLimiter *rerate.Limiter, spamLimiter *rerate.Limiter) func(smtpd.Peer, smtpd.Envelope) error {
	return func(peer smtpd.Peer, env smtpd.Envelope) error {
		// Validate data before sending them
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

			if err := spamLimiter.Inc(peer.HeloName); err != nil {
				log.WithFields(log.Fields{
					"heloName": peer.HeloName,
					"error":    err,
				}).Warn("Can't increment send")
			}
			return smtpd.Error{Code: 451, Message: "Spam filtered, increment rate limit"}
		}

		return nil
	}
}

func MakeHeloChecker(sendLimiter *rerate.Limiter, spamLimiter *rerate.Limiter) func(smtpd.Peer, string) error {
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
