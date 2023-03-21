package smpp

import (
	"context"
	"fmt"
	"github.com/oarkflow/errors"
	"github.com/oarkflow/protocol/smpp/balancer"
	"github.com/oarkflow/protocol/smpp/pdu"
	"github.com/oarkflow/protocol/smpp/pdu/pdufield"
	"github.com/oarkflow/protocol/smpp/pdu/pdutext"
	"github.com/puzpuzpuz/xsync"
	"github.com/rs/xid"
	"golang.org/x/time/rate"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

type Auth struct {
	SystemID   string
	Password   string
	SystemType string
}

type Setting struct {
	Name             string
	Slug             string
	ID               string
	URL              string
	Auth             Auth
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	EnquiryInterval  time.Duration
	EnquiryTimeout   time.Duration
	MaxConnection    int
	Balancer         balancer.Balancer
	Throttle         int
	UseAllConnection bool
	HandlePDU        func(p pdu.Body)
	AutoRebind       bool

	Validity time.Duration
	Register pdufield.DeliverySetting

	// Other fields, normally optional.
	ServiceType          string
	ESMClass             uint8
	ProtocolID           uint8
	PriorityFlag         uint8
	ScheduleDeliveryTime string
	ReplaceIfPresentFlag uint8
}

type Manager struct {
	Name                   string
	Slug                   string
	ID                     string
	ctx                    context.Context
	setting                Setting
	connections            map[string]*Transceiver
	Balancer               balancer.Balancer
	connIDs                []string
	mu                     sync.RWMutex
	Messages               *xsync.MapOf[string, Message]
	Parts                  *xsync.MapOf[string, Parts]
	lastMessageTS          time.Time
	lastDeliveredMessageTS time.Time
}

type Message struct {
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
	ID            string `json:"id"`
	Message       string `json:"message,omitempty"`
	MessageID     string `json:"message_id,omitempty"`
	MessageStatus string `json:"message_status,omitempty"`
	Error         string `json:"error,omitempty"`
	Parts         *xsync.MapOf[string, Parts]
	MessageParts  []Parts
}

type Parts struct {
	ID            string `json:"id"`
	SmsMessageID  string `json:"sms_message_id"`
	Message       string `json:"message,omitempty"`
	MessageID     string `json:"message_id,omitempty"`
	MessageStatus string `json:"message_status,omitempty"`
	Error         string `json:"error,omitempty"`
}

func NewManager(setting Setting) (*Manager, error) {
	var id string
	if setting.MaxConnection == 0 {
		setting.MaxConnection = 1
	}
	if setting.ReadTimeout == 0 {
		setting.ReadTimeout = 10 * time.Second
	}
	if setting.WriteTimeout == 0 {
		setting.WriteTimeout = 10 * time.Second
	}
	if setting.EnquiryInterval == 0 {
		setting.EnquiryInterval = 10 * time.Second
	}
	if setting.EnquiryTimeout == 0 {
		setting.EnquiryTimeout = 10 * time.Second
	}
	if setting.ID != "" {
		id = setting.ID
	} else {
		id = xid.New().String()
	}
	manager := &Manager{
		Name:        setting.Name,
		Slug:        setting.Slug,
		ID:          id,
		ctx:         context.Background(),
		connections: make(map[string]*Transceiver),
		Messages:    xsync.NewMapOf[Message](),
		Parts:       xsync.NewMapOf[Parts](),
	}

	if setting.HandlePDU == nil {
		setting.HandlePDU = func(p pdu.Body) {
			if msgStatus, ok := p.Fields()[pdufield.ShortMessage]; ok {
				response := Unmarshal(msgStatus.String())
				if _, ok := p.Manager().GetPart(response.Id); ok {
					var status string
					switch response.Stat {
					case "DELIVRD":
						status = "DELIVERED"
						p.Manager().SetLastDeliveredMessage()
						break
					default:
						status = response.Stat
					}
					p.Manager().UpdatePart(response.Id, status, response.Err)
					p.Manager().DeletePart(response.Id)
				}
			}
		}
	}
	if setting.Balancer == nil {
		manager.Balancer = &balancer.RoundRobin{}
	}
	manager.setting = setting
	return manager, nil
}

func (m *Manager) Start() error {
	if m.setting.UseAllConnection {
		for i := 0; i < m.setting.MaxConnection; i++ {
			err := m.SetupConnection()
			if err != nil {
				return errors.NewE(err, "Unable to make SMPP connection", "manager:start")
			}
		}
		return nil
	}
	if len(m.connIDs) == 0 {
		err := m.SetupConnection()
		if err != nil {
			return errors.NewE(err, "Unable to make SMPP connection", "manager:start")
		}
	}
	return nil
}

func (m *Manager) AddConnection(noOfConnection ...int) error {
	con := 1
	if len(noOfConnection) > 0 {
		con = noOfConnection[0]
	}
	if con > m.setting.MaxConnection {
		return errors.New("Can't create more than allowed no of connections.")
	}
	if (len(m.connIDs) + con) > m.setting.MaxConnection {
		return errors.New("There are active sessions. Can't create more than allowed no of sessions.")
	}
	connLeft := m.setting.MaxConnection - len(m.connIDs)
	n := 0
	if connLeft >= con {
		n = con
	} else {
		n = connLeft
	}
	for i := 0; i < n; i++ {
		err := m.SetupConnection()
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) RemoveConnection(conID ...string) error {
	if len(conID) > 0 {
		for _, id := range conID {
			if con, ok := m.connections[id]; ok {
				err := con.Close()
				if err != nil {
					return err
				}
				m.connIDs = remove(m.connIDs, id)
				delete(m.connections, id)
			}
		}
	} else {
		for id, con := range m.connections {
			err := con.Close()
			if err != nil {
				return err
			}
			m.connIDs = remove(m.connIDs, id)
			delete(m.connections, id)
		}
	}
	return nil
}

func (m *Manager) Rebind() error {
	err := m.Close()
	if err != nil {
		return err
	}
	m.connections = make(map[string]*Transceiver)
	m.connIDs = []string{}
	return m.Start()
}

func (m *Manager) GetPart(key string) (any, bool) {
	return m.Parts.Load(key)
}

func (m *Manager) UpdatePart(key, status, error string) {
	if v, ok := m.Parts.Load(key); ok {
		v.MessageStatus = status
		v.Error = error
		m.Parts.Store(key, v)
		if va, o := m.Messages.Load(v.SmsMessageID); o {
			v.MessageStatus = status
			v.Error = error
			va.Parts.Store(key, v)
		}
	}
}

func (m *Manager) DeletePart(key string) {
	m.Parts.Delete(key)
}

func (m *Manager) LastMessageAt() time.Time {
	return m.lastMessageTS
}

func (m *Manager) LastDeliveredMessageAt() time.Time {
	return m.lastDeliveredMessageTS
}

func (m *Manager) SetLastDeliveredMessage() {
	m.lastDeliveredMessageTS = time.Now()
}

func (m *Manager) GetMessages() (messages []any) {
	f := func(key string, val Message) bool {
		var parts []Parts
		fp := func(key string, part Parts) bool {
			parts = append(parts, part)
			return true
		}
		val.Parts.Range(fp)
		val.MessageParts = parts
		messages = append(messages, val)
		return true
	}
	m.Messages.Range(f)
	return
}

func (m *Manager) SetupConnection() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// make persistent connection
	tx := &Transceiver{
		ID:         xid.New().String(),
		Addr:       m.setting.URL,
		User:       m.setting.Auth.SystemID,
		Passwd:     m.setting.Auth.Password,
		Handler:    m.setting.HandlePDU,
		SystemType: m.setting.Auth.SystemType,

		EnquireLink:        m.setting.EnquiryInterval,
		EnquireLinkTimeout: m.setting.EnquiryTimeout,
		RespTimeout:        15 * time.Minute,
		BindInterval:       10 * time.Second,
		manager:            m,
	}

	if m.setting.Throttle != 0 {
		rateLimiter := rate.NewLimiter(rate.Limit(m.setting.Throttle), 1)
		tx.RateLimiter = rateLimiter
	} else {
		rateLimiter := rate.NewLimiter(rate.Limit(100), 1)
		tx.RateLimiter = rateLimiter
	}
	conn := tx.Bind()
	// check initial connection status
	var status ConnStatus
	if status = <-conn; status.Error() != nil {
		return status.Error()
	}
	go func() {
		for c := range conn {
			log.Println("SMPP connection status:", c.Status(), tx.ID)
		}
	}()
	m.connIDs = append(m.connIDs, tx.ID)
	m.connections[tx.ID] = tx
	return nil
}

func (m *Manager) GetConnection(conIds ...string) (any, error) {
	var err error
	var pickedID string
	if len(conIds) > 0 { // pick among custom
		pickedID, err = m.Balancer.Pick(conIds)
		if err != nil {
			return nil, err
		}
		if con, ok := m.connections[pickedID]; ok {
			return con, nil
		}
	}

	// pick among managing session
	pickedID, err = m.Balancer.Pick(m.connIDs)
	if err != nil {
		return nil, err
	}
	if con, ok := m.connections[pickedID]; ok {
		return con, nil
	}
	return nil, errors.New("no connection")
}

func (m *Manager) Send(payload interface{}, connectionId ...string) (any, error) {
	if len(m.connIDs) == 0 {
		err := m.Start()
		if err != nil {
			return nil, err
		}
	}
	t, err := m.GetConnection(connectionId...)
	if err != nil {
		return nil, err
	}
	tx := t.(*Transceiver)
	sms := payload.(Message)
	if sms.ID == "" {
		sms.ID = xid.New().String()
	}
	if sms.Parts == nil {
		sms.Parts = xsync.NewMapOf[Parts]()
	}
	encodedText, isLongMsg := pdutext.FindCoding([]byte(sms.Message))
	srcTon, srcNpi := parseSrcPhone(sms.From)
	destTon, destNpi := parseDestPhone(sms.To)
	shortMessage := &ShortMessage{
		Src:           sms.From,
		SourceAddrTON: srcTon,
		SourceAddrNPI: srcNpi,

		Dst:         sms.To,
		DestAddrTON: destTon,
		DestAddrNPI: destNpi,

		Text: encodedText,

		Validity: m.setting.Validity,
		Register: m.setting.Register,

		ServiceType:          m.setting.ServiceType,
		ESMClass:             m.setting.ESMClass,
		ProtocolID:           m.setting.ProtocolID,
		PriorityFlag:         m.setting.PriorityFlag,
		ScheduleDeliveryTime: m.setting.ScheduleDeliveryTime,
		ReplaceIfPresentFlag: m.setting.ReplaceIfPresentFlag,
	}
	m.Messages.Store(sms.ID, sms)
	m.lastMessageTS = time.Now()
	if isLongMsg {
		sm, err := tx.SubmitLongMsg(shortMessage)
		if err != nil {
			sms.MessageStatus = "Unable to send: " + err.Error()
			m.Messages.Store(sms.ID, sms)
			return nil, err
		}
		for _, s := range sm {
			msg := Parts{
				ID:           xid.New().String(),
				SmsMessageID: sms.ID,
			}
			msg.MessageID = s.RespID()
			if s.Resp().Header().Status == pdu.ESME_ROK {
				msg.MessageStatus = "SENT"
			} else {
				msg.MessageStatus = "FAILED"
				msg.Error = s.Resp().Header().Status.Error()
			}
			m.Parts.Store(msg.MessageID, msg)
			if curSms, ok := m.Messages.Load(sms.ID); ok {
				curSms.Parts.Store(msg.MessageID, msg)
			}
		}
	} else {
		s, err := tx.Submit(shortMessage)
		if err != nil {
			sms.MessageStatus = "Unable to send: " + err.Error()
			m.Messages.Store(sms.ID, sms)
			return nil, err
		}
		msg := Parts{
			ID:           xid.New().String(),
			SmsMessageID: sms.ID,
			Message:      string(s.Text.Encode()),
		}
		msg.MessageID = s.RespID()
		if s.Resp().Header().Status == pdu.ESME_ROK {
			msg.MessageStatus = "SENT"
		} else {
			msg.MessageStatus = "FAILED"
			msg.Error = s.Resp().Header().Status.Error()
		}
		m.Parts.Store(msg.MessageID, msg)
		if curSms, ok := m.Messages.Load(sms.ID); ok {
			curSms.Parts.Store(msg.MessageID, msg)
		}
	}
	curSms, _ := m.Messages.Load(sms.ID)
	return curSms, nil
}

func (m *Manager) Close(connectionId ...string) error {
	if len(connectionId) > 0 {
		if con, ok := m.connections[connectionId[0]]; ok {
			err := con.Close()
			if err != nil {
				return err
			}
		}
	} else {
		for _, conn := range m.connections {
			err := conn.Close()
			fmt.Println("closing")
			if err != nil {
				fmt.Println("error on closing")
				return err
			}
		}
	}

	return nil
}

func parseSrcPhone(phone string) (ton uint8, npi uint8) {
	if strings.HasPrefix(phone, "+") {
		ton = 1
		npi = 1
		return
	}

	if utf8.RuneCountInString(phone) <= 5 {
		ton = 3
		npi = 0
		return
	}
	if isLetter(phone) {
		ton = 5
		npi = 0
		return
	}
	ton = 1
	npi = 1
	return
}

func parseDestPhone(phone string) (ton uint8, npi uint8) {
	if strings.HasPrefix(phone, "+") {
		ton = 1
		npi = 1
		return
	}
	ton = 0
	npi = 1
	return
}

func isLetter(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func remove(s []string, r string) []string {
	for i, v := range s {
		if v == r {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

type GenericResponse struct {
	Id         string `csv:"id" json:"id"`
	Sub        string `csv:"sub" json:"sub"`
	Dlvrd      string `csv:"dlvrd" json:"dlvrd"`
	SubmitDate string `csv:"submit date" json:"submit date"`
	DoneDate   string `csv:"done date" json:"done date"`
	Stat       string `csv:"stat" json:"stat"`
	Err        string `csv:"err" json:"err"`
	Text       string `csv:"text" json:"text"`
}

func Unmarshal(msg string) GenericResponse {
	fields := []string{"id", "sub", "dlvrd", "submit date", "done date", "stat", "err", "text", "Text"}
	var response GenericResponse
	for _, field := range fields {
		f := field + ":"
		ln1 := strings.Index(msg, f) + len(f)
		v := ""
		if len(msg) >= ln1 {
			if ln2 := strings.Index(msg[ln1:], " "); ln2 != -1 {
				v = msg[ln1:][:ln2]
			} else {
				v = msg[ln1:]
			}
		}
		switch field {
		case "id":
			response.Id = v
		case "sub":
			response.Sub = v
		case "dlvrd":
			response.Dlvrd = v
		case "submit date":
			response.SubmitDate = v
		case "done date":
			response.DoneDate = v
		case "stat":
			response.Stat = v
		case "err":
			response.Err = v
		case "text", "Text":
			response.Text = v
		}
	}
	return response
}
