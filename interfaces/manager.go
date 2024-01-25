package interfaces

import "time"

type IManager interface {
	Start() error
	AddConnection(noOfConnection ...int) error
	RemoveConnection(connectionID ...string) error
	GetConnection(conIds ...string) (any, error)
	SetupConnection() error
	GetPart(key string) (any, bool)
	UpdatePart(key, status, error string)
	GetMessages() []any
	DeletePart(key string)
	LastMessageAt() time.Time
	LastDeliveredMessageAt() time.Time
	SetLastDeliveredMessage()
	Rebind() error
	Send(payload any, connectionID ...string) (any, error)
	Close(connectionID ...string) error
}
