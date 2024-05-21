package mailer

import (
	"fmt"
	"net/smtp"
)

var (
	MessageTemplate = "From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n"
)

type Mailer struct {
	address string
	sender  string
}

func (m *Mailer) Mail(recipient, subject, text string) error {
	client, err := smtp.Dial(m.address)
	if err != nil {
		return fmt.Errorf("failed to connect to smtp server: %w", err)
	}

	if err := client.Mail(m.sender); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}

	_, err = fmt.Fprintf(writer, MessageTemplate, m.sender, recipient, subject, text)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("failed to quit: %w", err)
	}

	return nil
}

// func main() {
// 	// Connect to the remote SMTP server.
// 	c, err := smtp.Dial("localhost:25")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Set the sender and recipient first
// 	if err := c.Mail("root@hog26.urgu.org"); err != nil {
// 		log.Fatal(err)
// 	}
// 	if err := c.Rcpt("cmaster057@gmail.com"); err != nil {
// 		log.Fatal(err)
// 	}

// 	// Send the email body.
// 	wc, err := c.Data()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	_, err = fmt.Fprintf(wc, "From: root@hog26.urgu.org\r\n"+
// 		"To: cmaster057@gmail.com\r\n"+
// 		"Subject: Test mail\r\n\r\n"+
// 		"Email body\r\n")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	err = wc.Close()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Send the QUIT command and close the connection.
// 	err = c.Quit()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

func New(address string) *Mailer {
	sender := "root@hog26.urgu.org" //todo

	return &Mailer{
		address: address,
		sender:  sender,
	}
}
