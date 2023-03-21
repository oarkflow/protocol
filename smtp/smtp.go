package smtp

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/sujit-baniya/log"
	"github.com/valyala/bytebufferpool"
	sMail "github.com/xhit/go-simple-mail/v2"
)

var maxBigInt = big.NewInt(math.MaxInt64)

type Config struct {
	Host        string `mapstructure:"MAIL_HOST" json:"host" yaml:"host" env:"MAIL_HOST"`
	Username    string `mapstructure:"MAIL_USERNAME" json:"username" yaml:"username" env:"MAIL_USERNAME"`
	Password    string `mapstructure:"MAIL_PASSWORD" json:"password" yaml:"password" env:"MAIL_PASSWORD"`
	Encryption  string `mapstructure:"MAIL_ENCRYPTION" json:"encryption" yaml:"encryption" env:"MAIL_ENCRYPTION"`
	FromAddress string `mapstructure:"MAIL_FROM_ADDRESS" json:"from_address" yaml:"from_address" env:"MAIL_FROM_ADDRESS"`
	FromName    string `mapstructure:"MAIL_FROM_NAME" json:"from_name" yaml:"from_name" env:"MAIL_FROM_NAME"`
	EmailLayout string `mapstructure:"MAIL_LAYOUT" json:"layout" yaml:"layout" env:"MAIL_LAYOUT"`
	Port        int    `mapstructure:"MAIL_PORT" json:"port" yaml:"port" env:"MAIL_PORT"`
}

type Mail struct {
	*sMail.SMTPServer
	*sMail.SMTPClient
	*html.Engine
	Config Config
}

type Attachment struct {
	Data     []byte
	File     string
	MimeType string
}

type Message struct {
	To          string       `json:"to,omitempty"`
	From        string       `json:"from,omitempty"`
	Subject     string       `json:"subject,omitempty"`
	Body        string       `json:"body,omitempty"`
	Cc          string       `json:"cc,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

var DefaultMailer *Mail
var TemplateEngine *html.Engine

func Default(cfg Config, templateEngine ...*html.Engine) {
	DefaultMailer = New(cfg, templateEngine...)
}

func New(cfg Config, templateEngine ...*html.Engine) *Mail {
	if len(templateEngine) > 0 {
		TemplateEngine = templateEngine[0]
	}
	m := &Mail{Config: cfg}
	m.SMTPServer = sMail.NewSMTPClient()
	m.SMTPServer.Host = cfg.Host
	m.SMTPServer.Port = cfg.Port
	m.SMTPServer.Username = cfg.Username
	m.SMTPServer.Password = cfg.Password
	if cfg.Encryption == "tls" {
		m.SMTPServer.Encryption = sMail.EncryptionSTARTTLS
	} else {
		m.SMTPServer.Encryption = sMail.EncryptionSSL
	}
	//Variable to keep alive connection
	m.SMTPServer.KeepAlive = false
	//Timeout for connect to SMTP Server
	m.SMTPServer.ConnectTimeout = 10 * time.Second
	//Timeout for send the data and wait respond
	m.SMTPServer.SendTimeout = 10 * time.Second
	return m
}

func (m *Mail) Send(msg Message, includeMessageID ...bool) error {
	var err error
	m.SMTPClient, err = m.SMTPServer.Connect()
	if err != nil {
		fmt.Println("Error on connection: " + err.Error())
		return err
	}
	defer m.SMTPClient.Close()
	//New email simple html with inline and CC
	email := sMail.NewMSG()
	if len(includeMessageID) > 0 && includeMessageID[0] {
		id, _ := generateMessageID()
		email.AddHeader("Message-Id", id)
	}
	if msg.From == "" {
		msg.From = fmt.Sprintf("%s<%s>", m.Config.FromName, m.Config.FromAddress)
	}
	email.SetFrom(msg.From).AddTo(msg.To).SetSubject(msg.Subject)
	if msg.Cc != "" { //nolint:wsl
		email.AddCc(msg.Cc)
	}
	// txt, _ := html2text.FromString(body, html2text.Options{PrettyTables: false})
	// email.AddAlternative(sMail.TextPlain, txt)
	email.SetBody(sMail.TextHTML, msg.Body) //nolint:wsl
	for _, attachment := range msg.Attachments {
		email.AddAttachmentData(attachment.Data, attachment.File, attachment.MimeType)
	}

	//Call Send and pass the client
	err = email.Send(m.SMTPClient)
	if err != nil {
		fmt.Println(err.Error())
		return err
	} else {
		log.Info().Msg("Email Sent to " + msg.To)
	}
	return nil
}

func View(view string, body fiber.Map) *Body {
	bodyContent := &Body{Content: Html(view, body)}
	return bodyContent
}

func Html(view string, body fiber.Map) string {
	buf := bytebufferpool.Get()
	defer bytebufferpool.Put(buf)
	if err := TemplateEngine.Render(buf, view, body, DefaultMailer.Config.EmailLayout); err != nil {
		panic(err)
	}
	return buf.String()
}

func Send(msg Message) error {
	return DefaultMailer.Send(msg)
}

type Body struct {
	Content string
}

func (t *Body) Send(msg Message) error {
	msg.Body = t.Content
	return DefaultMailer.Send(msg)
}

func generateMessageID() (string, error) {
	t := time.Now().UnixNano()
	pid := os.Getpid()
	rint, err := rand.Int(rand.Reader, maxBigInt)
	if err != nil {
		return "", err
	}
	h, err := os.Hostname()
	// If we can't get the hostname, we'll use localhost
	if err != nil {
		h = "localhost.localdomain"
	}
	msgid := fmt.Sprintf("<%d.%d.%d@%s>", t, pid, rint, h)
	return msgid, nil
}

func SendPasswordResetEmail(email string, formattedEmail string, baseURL string, serverKey string) error {
	resetLink := GeneratePasswordResetURL(formattedEmail, baseURL, serverKey)
	err := View("emails/password-reset", fiber.Map{
		"reset_link": resetLink,
	}).Send(Message{
		To:      email,
		Subject: "You asked to reset. Please click here.",
	})
	if err != nil {
		fmt.Println("Retrying sending password reset email: " + email)
		SendPasswordResetEmail(email, formattedEmail, baseURL, serverKey)
	}
	return err
}
func SendNewAccountEmail(email string, password string, baseURL string, serverKey string) error {
	resetEmail := fmt.Sprintf("%s-reset-%d", email, time.Now().Unix())
	resetLink := GeneratePasswordResetURL(resetEmail, baseURL, serverKey)
	err := View("emails/new-user", fiber.Map{
		"reset_link": resetLink,
		"password":   password,
	}).Send(Message{
		To:      email,
		Subject: "Your new account is here!",
	})
	if err != nil {
		fmt.Println("Retrying sending new account email: " + email)
		SendNewAccountEmail(email, password, baseURL, serverKey)
	}
	return err
}

func SendConfirmationEmail(email, baseURL, token string) error {
	confirmLink := GenerateConfirmURL(email, baseURL, token)
	err := View("emails/confirm", fiber.Map{
		"confirm_link": confirmLink,
	}).Send(Message{
		To:      email,
		Subject: "Please confirm if it is you.",
	})
	if err != nil {
		fmt.Println("Retrying sending confirmation email: " + email)
		SendConfirmationEmail(email, baseURL, token)
	}
	return err
}

func GenerateConfirmURL(email, baseURL, token string) string {
	uri := fmt.Sprintf("%s/verify-email?t=%s", baseURL, token)
	return uri
}

func GeneratePasswordResetURL(email, baseURL, token string) string {
	uri := fmt.Sprintf("%s/reset-password?t=%s", baseURL, token)
	return uri
}
