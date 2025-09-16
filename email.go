package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/mail"
	"strings"

	"github.com/mhale/smtpd"
	"github.com/rs/zerolog/log"
)

type EmailData struct {
	To      []string
	From    string
	Subject string
	Body    string
	Date    string
	IP      string
}

func startServer() {
	log.Info().Msg(fmt.Sprintf("Listening on port %s", port))
	if err := smtpd.ListenAndServe(fmt.Sprintf(":%s", port), mailHandler, "", ""); err != nil {
		log.Error().Err(err).Msg("Error with the SMTP server")
	}
}

/*
mailHandler is called by smtpd when an email is received

Parameters:

	remoteIP - The IP address that the email was sent from
	from     - The email address that the email came from
	to       - The email address(es) that the email was sent to
	data     - The raw email data
*/
func mailHandler(remoteIP net.Addr, from string, to []string, data []byte) error {
	log.Info().Msg("Email Received")

	// Transform the IP into a string
	ip, _, err := net.SplitHostPort(remoteIP.String())
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve the ip")
	}

	log.Debug().Str("to", strings.Trim(fmt.Sprint(to), "[]")).Str("from", from).Str("ip", ip).Send()

	// Parse the email
	var emailSubject string
	var emailDate string
	var emailBody string
	msg, err := mail.ReadMessage(bytes.NewReader(data))
	if err != nil {
		log.Error().Err(err).Msg("Can't parse email")
		emailBody = "There was an error when parsing the email"
	} else {
		emailSubject = msg.Header.Get("Subject")
		emailDate = msg.Header.Get("Date")
		builder := &strings.Builder{}
		_, err = io.Copy(builder, msg.Body)
		if err != nil {
			log.Error().Err(err).Msg("Error with email body")
		}
		emailBody = builder.String()
	}

	// Determine which junction to use, or return if none found
	index := selectJunction(to, from, ip)
	if index < 0 {
		log.Error().Msg("No junction matches the received email")
		return nil
	}
	junction := junctions[index]

	// Prepare the title and body for the message
	for handler_index, junction_handler := range junction.Handlers {
		title, body, url := buildMessage(EmailData{
			to,
			from,
			emailSubject,
			emailBody,
			emailDate,
			ip,
		}, junction_handler)

		// Get the index or junction name for the logs
		id := func() string {
			var handler_name string
			if junction_handler.Name == "" {
				handler_name = fmt.Sprint(handler_index)
			} else {
				handler_name = junction_handler.Name
			}
			if junction.Name == "" {
				return fmt.Sprintf("%d.%s", index, handler_name)
			}
			return fmt.Sprintf("%s.%s", junction.Name, handler_name)
		}

		// Send it
		log.Info().Str("junction id", id()).Msg("Sending Notification")
		sendNotification(title, body, url)
	}
	return nil
}
