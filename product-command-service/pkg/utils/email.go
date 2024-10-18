package utils

import "gopkg.in/gomail.v2"

func SendEmail(message *gomail.Message, sender string, password string, smtpServer string, smtpPort int) error {
	// Create a new SMTP client to send the email
	d := gomail.NewDialer(smtpServer, smtpPort, sender, password)

	// Send the email
	if err := d.DialAndSend(message); err != nil {
		return err
	}

	return nil
}
