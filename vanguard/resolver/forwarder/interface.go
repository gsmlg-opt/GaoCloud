package forwarder

import (
	"time"

	"g53"
)

type SafeFwder interface {
	GetLastRtt() time.Duration
	IsDown() bool
	RemoteAddr() string
	Forward(query *g53.Message) (*g53.Message, time.Duration, error)
	SetQuerySource(string) error
}

type FwderSelector interface {
	SelectFwder() SafeFwder
	HasUpFwder() bool
	GetFwders() []SafeFwder
	SetQuerySource(string) error
}
