package forwarder

import (
	"time"

	"g53"
)

type RecoverableFwder struct {
	SafeFwder
	prober *Prober
}

func NewRecoverableFwder(fwder SafeFwder, prober *Prober) *RecoverableFwder {
	return &RecoverableFwder{
		SafeFwder: fwder,
		prober:    prober,
	}
}

func (f *RecoverableFwder) Forward(query *g53.Message) (*g53.Message, time.Duration, error) {
	resp, rtt, err := f.SafeFwder.Forward(query)
	if f.SafeFwder.IsDown() {
		f.prober.AddProbe(f.SafeFwder, query)
	}
	return resp, rtt, err
}
