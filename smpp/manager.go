package smpp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/oarkflow/errors"
	"github.com/oarkflow/log"
	"golang.org/x/time/rate"

	"github.com/oarkflow/protocol/smpp/balancer"
	"github.com/oarkflow/protocol/smpp/pdu"
	"github.com/oarkflow/protocol/smpp/pdu/pdufield"
	"github.com/oarkflow/protocol/smpp/pdu/pdutext"
	"github.com/oarkflow/protocol/utils/maps"
	"github.com/oarkflow/protocol/utils/str"
	"github.com/oarkflow/protocol/utils/xid"
)

type Auth struct {
	SystemID   string
	Password   string
	SystemType string
}

type Setting struct {
	Name                 string                   `json:"name,omitempty"`
	Slug                 string                   `json:"slug,omitempty"`
	ID                   string                   `json:"id,omitempty"`
	URL                  string                   `json:"url,omitempty"`
	Auth                 Auth                     `json:"auth"`
	ReadTimeout          time.Duration            `json:"read_timeout,omitempty"`
	WriteTimeout         time.Duration            `json:"write_timeout,omitempty"`
	EnquiryInterval      time.Duration            `json:"enquiry_interval,omitempty"`
	EnquiryTimeout       time.Duration            `json:"enquiry_timeout,omitempty"`
	MaxConnection        int                      `json:"max_connection,omitempty"`
	Balancer             balancer.Balancer        `json:"balancer,omitempty"`
	Throttle             int                      `json:"throttle,omitempty"`
	UseAllConnection     bool                     `json:"use_all_connection,omitempty"`
	AutoRebind           bool                     `json:"auto_rebind,omitempty"`
	Validity             time.Duration            `json:"validity,omitempty"`
	Register             pdufield.DeliverySetting `json:"register,omitempty"`
	ServiceType          string                   `json:"service_type,omitempty"`
	ESMClass             uint8                    `json:"esm_class,omitempty"`
	ProtocolID           uint8                    `json:"protocol_id,omitempty"`
	PriorityFlag         uint8                    `json:"priority_flag,omitempty"`
	ScheduleDeliveryTime string                   `json:"schedule_delivery_time,omitempty"`
	ReplaceIfPresentFlag uint8                    `json:"replace_if_present_flag,omitempty"`
	HandlePDU            func(p pdu.Body)
	OnPartReport         func(manager *Manager, parts []*Part)
	OnMessageReport      func(manager *Manager, sms *Message, parts []*Part)
}

type Manager struct {
	Name                   string
	Slug                   string
	ID                     string
	ctx                    context.Context
	setting                Setting
	connections            map[string]*Transceiver
	messagesToRetry        map[any]*Transceiver
	balancer               balancer.Balancer
	connIDs                []string
	mu                     sync.RWMutex
	messages               *maps.Map[string, *Message]
	parts                  *maps.Map[string, *Part]
	messageParts           *maps.Map[string, string]
	smsParts               *maps.Map[string, []string]
	lastMessageTS          time.Time
	lastDeliveredMessageTS time.Time
}

type Message struct {
	From           string    `json:"from,omitempty"`
	To             string    `json:"to,omitempty"`
	ID             string    `json:"id"`
	UserID         any       `json:"user_id"`
	Message        string    `json:"message,omitempty"`
	MessageID      string    `json:"message_id,omitempty"`
	MessageStatus  string    `json:"message_status,omitempty"`
	Error          string    `json:"error,omitempty"`
	TotalParts     int32     `json:"total_parts"`
	SentParts      int32     `json:"sent_parts"`
	FailedParts    int32     `json:"failed_parts"`
	DeliveredParts int32     `json:"delivered_parts"`
	CreatedAt      time.Time `json:"created_at"`
	SentAt         time.Time `json:"sent_at"`
	DeliveredAt    time.Time `json:"delivered_at"`
	FailedAt       time.Time `json:"failed_at"`
	totalParts     atomic.Int32
	sentParts      atomic.Int32
	failedParts    atomic.Int32
	deliveredParts atomic.Int32
}

type Part struct {
	ID            string    `json:"id"`
	SmsMessageID  string    `json:"sms_message_id"`
	Message       string    `json:"message,omitempty"`
	MessageID     string    `json:"message_id,omitempty"`
	MessageStatus string    `json:"message_status,omitempty"`
	Error         string    `json:"error,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	SentAt        time.Time `json:"sent_at"`
	DeliveredAt   time.Time `json:"delivered_at"`
	FailedAt      time.Time `json:"failed_at"`
}

const (
	DELIVERED string = "DELIVERED"
	FAILED    string = "FAILED"
)

func NewManager(setting Setting) (*Manager, error) {
	var id string
	{
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
	}
	manager := &Manager{
		Name:            setting.Name,
		Slug:            setting.Slug,
		ID:              id,
		ctx:             context.Background(),
		messagesToRetry: make(map[any]*Transceiver),
		connections:     make(map[string]*Transceiver),
		messages:        maps.New[string, *Message](10000),
		parts:           maps.New[string, *Part](10000),
		messageParts:    maps.New[string, string](10000),
		smsParts:        maps.New[string, []string](10000),
	}

	if setting.HandlePDU == nil {
		setting.HandlePDU = manager.DefaultPDUHandler
	}
	if setting.Balancer == nil {
		manager.balancer = &balancer.RoundRobin{}
	}
	manager.setting = setting
	return manager, nil
}

func (m *Manager) DefaultPDUHandler(p pdu.Body) {
	if msgStatus, ok := p.Fields()[pdufield.ShortMessage]; ok {
		response := Unmarshal(msgStatus.String())
		if response == nil {
			return
		}
		id := response["id"]
		status := response["stat"]
		if part, ok := m.parts.Get(id); ok {
			switch status {
			case "DELIVRD":
				status = DELIVERED
				m.SetLastDeliveredMessage()
				break
			}
			part.MessageStatus = status
			part.DeliveredAt = time.Now()
			part.Error = response["err"]
			m.parts.Set(id, part)
			if smsID, mExists := m.messageParts.Get(id); mExists {
				if sms, exists := m.messages.Get(smsID); exists {
					sms.sentParts.Add(-1)
					if status == DELIVERED {
						sms.deliveredParts.Add(1)
					} else {
						sms.failedParts.Add(1)
					}
					totalParts := sms.totalParts.Load()
					failedParts := sms.failedParts.Load()
					deliveredParts := sms.deliveredParts.Load()
					deleteAll := false
					if totalParts == (failedParts + deliveredParts) {
						deleteAll = true
						if failedParts > 0 {
							sms.MessageStatus = FAILED
							sms.FailedAt = time.Now()
						} else {
							sms.MessageStatus = DELIVERED
							sms.DeliveredAt = time.Now()
						}
					}
					m.messages.Set(smsID, sms)
					m.Report(sms)
					if smsParts, pExists := m.smsParts.Get(sms.ID); pExists && deleteAll {
						m.messages.Del(sms.ID)
						m.smsParts.Del(sms.ID)
						for _, p := range smsParts {
							m.parts.Del(p)
						}
					}
				}
			}
		}
	}
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

func (m *Manager) Report(sms *Message) {
	if sms.MessageStatus != "" && m.setting.OnMessageReport != nil {
		smsParts, pExists := m.smsParts.Get(sms.ID)
		var parts []*Part
		if pExists {
			for _, p := range smsParts {
				if part, e := m.parts.Get(p); e {
					parts = append(parts, part)
				}
			}
		}
		sms.TotalParts = sms.totalParts.Load()
		sms.SentParts = sms.sentParts.Load()
		sms.DeliveredParts = sms.deliveredParts.Load()
		sms.FailedParts = sms.failedParts.Load()
		m.setting.OnMessageReport(m, sms, parts)
	}
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
	return m.parts.Get(key)
}

func (m *Manager) UpdatePart(key, status, error string) {
	if v, ok := m.parts.Get(key); ok {
		v.MessageStatus = status
		v.Error = error
		m.parts.Set(key, v)
	}
}

func (m *Manager) DeletePart(key string) {
	m.parts.Del(key)
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
	m.messages.ForEach(func(key string, val *Message) bool {
		messages = append(messages, val)
		return true
	})
	return
}

func (m *Manager) SetupConnection() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// make persistent connection
	tx := &Transceiver{
		ID:                 xid.New().String(),
		Addr:               m.setting.URL,
		User:               m.setting.Auth.SystemID,
		Passwd:             m.setting.Auth.Password,
		Handler:            m.setting.HandlePDU,
		SystemType:         m.setting.Auth.SystemType,
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
	go func(m *Manager) {
		for c := range conn {
			if c.Status() == Connected && len(m.messagesToRetry) > 0 {
				for payload, t := range m.messagesToRetry {
					switch payload := payload.(type) {
					case Message:
						log.Warn().Bool("resend", true).Str("message_id", payload.ID).Msg("Resending message")
					case *Message:
						log.Warn().Bool("resend", true).Str("message_id", payload.ID).Msg("Resending message")
					}
					_, err := m.Send(payload, t.ID)
					if err == nil {
						m.mu.Lock()
						delete(m.messagesToRetry, payload)
						m.mu.Unlock()
					}
				}
			}
		}
	}(m)
	m.connIDs = append(m.connIDs, tx.ID)
	m.connections[tx.ID] = tx
	return nil
}

func (m *Manager) GetConnection(conIds ...string) (any, error) {
	var err error
	var pickedID string
	if len(conIds) > 0 { // pick among custom
		pickedID, err = m.balancer.Pick(conIds)
		if err != nil {
			return nil, err
		}
		if con, ok := m.connections[pickedID]; ok {
			return con, nil
		}
	}

	// pick among managing session
	pickedID, err = m.balancer.Pick(m.connIDs)
	if err != nil {
		return nil, err
	}
	if con, ok := m.connections[pickedID]; ok {
		return con, nil
	}
	return nil, errors.New("no connection")
}

func (m *Manager) Send(payload any, connectionId ...string) (any, error) {
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
	var sms *Message
	switch payload := payload.(type) {
	case Message:
		sms = &payload
	case *Message:
		sms = payload
	}
	if sms.ID == "" {
		sms.ID = xid.New().String()
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
	if isLongMsg {
		sm, err := tx.SubmitLongMsg(shortMessage)
		if err != nil {
			m.mu.Lock()
			m.messagesToRetry[sms] = tx
			m.mu.Unlock()
			sms.MessageStatus = FAILED
			sms.FailedAt = time.Now()
			sms.Error = err.Error()
			m.messages.Set(sms.ID, sms)
			m.Report(sms)
			return nil, err
		}
		sms.totalParts.Add(int32(len(sm)))
		m.lastMessageTS = time.Now()
		for _, s := range sm {
			part := &Part{
				ID:           xid.New().String(),
				SmsMessageID: sms.ID,
				Message:      str.FromByte(s.Text.Decode()),
				MessageID:    s.RespID(),
			}
			if s.Resp().Header().Status == pdu.ESME_ROK {
				part.MessageStatus = "SENT"
				part.SentAt = time.Now()
				sms.sentParts.Add(1)
			} else {
				part.MessageStatus = "FAILED"
				part.FailedAt = time.Now()
				part.Error = s.Resp().Header().Status.Error()
				sms.failedParts.Add(1)
			}
			m.messageParts.Set(s.RespID(), sms.ID)
			m.parts.Set(part.MessageID, part)
			if smsParts, s := m.smsParts.Get(sms.ID); s {
				smsParts = append(smsParts, part.MessageID)
				m.smsParts.Set(sms.ID, smsParts)
			} else {
				m.smsParts.Set(sms.ID, []string{part.MessageID})
			}
		}
		sms.Error = ""
		m.messages.Set(sms.ID, sms)
		m.Report(sms)
	} else {
		s, err := tx.Submit(shortMessage)
		if err != nil {
			m.mu.Lock()
			m.messagesToRetry[sms] = tx
			m.mu.Unlock()
			sms.MessageStatus = FAILED
			sms.FailedAt = time.Now()
			sms.Error = err.Error()
			m.messages.Set(sms.ID, sms)
			m.Report(sms)
			return nil, err
		}
		sms.totalParts.Add(1)
		part := &Part{
			ID:           xid.New().String(),
			SmsMessageID: sms.ID,
			Message:      string(s.Text.Encode()),
			MessageID:    s.RespID(),
		}
		if s.Resp().Header().Status == pdu.ESME_ROK {
			part.MessageStatus = "SENT"
			part.SentAt = time.Now()
			sms.sentParts.Add(1)
		} else {
			part.MessageStatus = "FAILED"
			part.FailedAt = time.Now()
			part.Error = s.Resp().Header().Status.Error()
			sms.failedParts.Add(1)
		}
		sms.Error = ""
		sms.MessageStatus = "SENT"
		sms.SentAt = time.Now()
		m.messageParts.Set(s.RespID(), sms.ID)
		m.parts.Set(part.MessageID, part)
		m.messages.Set(sms.ID, sms)
		m.smsParts.Set(sms.ID, []string{part.MessageID})
		m.Report(sms)
	}
	curSms, _ := m.messages.Get(sms.ID)
	return curSms, nil
}

func (m *Manager) Wait() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan bool, 1)
	go func() {
		<-sigs
		done <- true
	}()

	fmt.Println("Awaiting SMPP Manager")
	<-done
	m.Close()
	fmt.Println("Exiting SMPP Manager")
}

func (m *Manager) Close(connectionId ...string) error {
	if len(connectionId) > 0 {
		if con, ok := m.connections[connectionId[0]]; ok {
			err := con.Close()
			if err != nil {
				return err
			}
			log.Info().Str("conn_id", con.ID).Msg("SMPP Connection Closing")
		}
	} else {
		for _, conn := range m.connections {
			err := conn.Close()
			if err != nil {
				return err
			}
			log.Info().Str("conn_id", conn.ID).Msg("SMPP Connection Closing")
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

var (
	re = regexp.MustCompile(`id:(\w+) sub:(\d+) dlvrd:(\d+) submit date:(\d+) done date:(\d+) stat:(\w+) err:(\d+) text:(.+)`)
)

func Unmarshal(message string) map[string]string {
	matches := re.FindStringSubmatch(message)
	if len(matches) == 0 {
		return nil
	}
	var resultMap = make(map[string]string)
	keys := []string{"id", "sub", "dlvrd", "submit_date", "done_date", "stat", "err", "text"}
	for i, key := range keys {
		resultMap[key] = matches[i+1]
	}
	return resultMap
}

/*

type GenericResponse struct {
	ID         string `csv:"id" json:"id"`
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
			response.ID = v
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

*/
