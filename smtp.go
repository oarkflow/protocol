package protocol

import (
	"fmt"
	"github.com/oarkflow/protocol/smtp"
)

type SMTP struct {
	mailer *smtp.Mailer
	Config smtp.Config
}

func (s *SMTP) Setup() error {
	return nil
}

func (s *SMTP) SetService(service Service) {

}

func (s *SMTP) GetType() Type {
	return Smtp
}

func (s *SMTP) Handle(payload Payload) (Response, error) {
	from := payload.FromEmail
	if payload.From != "" {
		from = fmt.Sprintf("%s<%s>", payload.From, from)
	}
	err := s.mailer.Send(smtp.Mail{
		To:          []string{payload.To},
		From:        from,
		Subject:     payload.Subject,
		Body:        payload.Message,
		Cc:          []string{payload.Cc},
		Attachments: payload.Attachments,
	})
	if err != nil {
		return nil, err
	}
	return Response("email dispatched"), nil
}

func (s *SMTP) Queue(payload Payload) (Response, error) {
	return s.Handle(payload)
}
