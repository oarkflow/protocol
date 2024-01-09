package protocol

import "github.com/oarkflow/protocol/smpp"

type SMPP struct {
	manager *smpp.Manager
	Config  smpp.Setting
}

func (s *SMPP) Setup() error {
	return s.manager.Start()
}

func (s *SMPP) GetType() Type {
	return Smpp
}

func (s *SMPP) SetService(service Service) {

}

func (s *SMPP) Handle(payload Payload) (Response, error) {
	return s.manager.Send(smpp.Message{
		From:    payload.From,
		To:      payload.To,
		Message: payload.Message,
		UserID:  payload.UserID,
	})
}

func (s *SMPP) Queue(payload Payload) (Response, error) {
	return s.Handle(payload)
}
