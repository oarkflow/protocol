package protocol

import (
	"encoding/json"
	"time"

	"github.com/oarkflow/frame/server/render"

	"github.com/oarkflow/protocol/http"
	"github.com/oarkflow/protocol/smpp"
	"github.com/oarkflow/protocol/smtp"
	"github.com/oarkflow/protocol/utils/template"
)

type Type string

const (
	Smpp Type = "smpp"
	Smtp Type = "smtp"
	Http Type = "http"
)

type Payload struct {
	ID               string            `json:"id"`
	From             string            `json:"from"`
	FromName         string            `json:"from_name"`
	To               string            `json:"to"`
	UserID           any               `json:"user_id"`
	Message          string            `json:"message"`
	Subject          string            `json:"subject"`
	Cc               string            `json:"cc"`
	Query            string            `json:"query"`
	Attachments      []smtp.Attachment `json:"attachments"`
	CallbackURL      string            `json:"callback_url"`
	URL              string            `json:"url"`
	Method           string            `json:"method"`
	RequestStructure string            `json:"request_structure"`
	Data             map[string]any    `json:"data"`
	Headers          map[string]string `json:"headers"`
	CreatedAt        time.Time         `json:"created_at"`
	SentAt           time.Time         `json:"sent_at"`
	DeliveredAt      time.Time         `json:"delivered_at"`
	FailedAt         time.Time         `json:"failed_at"`
}

func (p *Payload) Prepare() (err error) {
	if p.Data == nil && p.RequestStructure != "" {
		err = json.Unmarshal([]byte(p.RequestStructure), &p.Data)
		if err != nil {
			return
		}
	} else if p.Data != nil && p.RequestStructure != "" {
		var data map[string]any
		tmp := template.New(p.RequestStructure, "", "")
		p.RequestStructure = tmp.Parse(p.Data)
		err = json.Unmarshal([]byte(p.RequestStructure), &data)
		if err != nil {
			return
		}
		p.Data = data
	} else if p.Data != nil && p.Message != "" {
		tmp := template.New(p.Message, "", "")
		p.Message = tmp.Parse(p.Data)
	}
	return
}

type Response any

type Service interface {
	Setup() error
	GetType() Type
	SetService(service Service)
	Handle(payload Payload) (Response, error)
	Queue(payload Payload) (Response, error)
}

func NewSMTP(config smtp.Config, engine *render.HtmlEngine) (*SMTP, error) {
	return &SMTP{mailer: smtp.New(config, engine), Config: config}, nil
}

func NewHTTP(config *http.Options) (*HTTP, error) {
	return &HTTP{client: http.NewClient(config), Config: config}, nil
}

func NewSMPP(config smpp.Setting) (*SMPP, error) {
	manager, err := smpp.NewManager(config)
	if err != nil {
		return nil, err
	}
	return &SMPP{manager: manager, Config: config}, nil
}
