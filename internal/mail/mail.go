package mail

import (
	"fmt"
	"net/smtp"
)

var (
	MessageTemplate = "Content-Type: text/html\r\nFrom: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n"
)

type Sender struct {
	serverAddress string
	senderName    string
}

func (s *Sender) Send(recipient, subject, text string) error {
	client, err := smtp.Dial(s.serverAddress)
	if err != nil {
		return fmt.Errorf("failed to connect to smtp server: %w", err)
	}

	if err := client.Mail(s.senderName); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to create writer: %w", err)
	}

	_, err = fmt.Fprintf(writer, MessageTemplate, s.senderName, recipient, subject, text)
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

func NewSender(address, senderName string) *Sender {
	return &Sender{
		serverAddress: address,
		senderName:    senderName,
	}
}
