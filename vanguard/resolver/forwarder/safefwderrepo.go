package forwarder

import (
	"time"

	"vanguard/config"
)

const (
	defaultProbeInterval  = 60
	defaultFwderTimeout   = 2
	defaultTimeoutLasting = 5
)

type SafeFwderRepo struct {
	probeInterval  time.Duration
	fwderTimeout   time.Duration
	timeoutLasting time.Duration

	fwders map[string]SafeFwder
	prober *Prober
}

func NewSafeFwderRepo(conf *config.ForwardProberConf) *SafeFwderRepo {
	repo := &SafeFwderRepo{}
	repo.ReloadConf(conf)
	return repo
}

func (repo *SafeFwderRepo) ReloadConf(conf *config.ForwardProberConf) {
	if repo.prober != nil {
		repo.prober.Stop()
	}

	probeInterval := conf.ProbeInterval
	if probeInterval == 0 {
		probeInterval = defaultProbeInterval
	}

	fwderTimeout := conf.Timeout
	if fwderTimeout == 0 {
		fwderTimeout = defaultFwderTimeout
	}

	timeoutLasting := conf.TimeoutLasting
	if timeoutLasting == 0 {
		timeoutLasting = defaultTimeoutLasting
	}

	repo.probeInterval = time.Duration(probeInterval) * time.Second
	repo.fwderTimeout = time.Duration(fwderTimeout) * time.Second
	repo.timeoutLasting = time.Duration(timeoutLasting) * time.Second
	repo.fwders = make(map[string]SafeFwder)
	repo.prober = NewProber(repo.probeInterval)
}

func (repo *SafeFwderRepo) GetOrCreateFwder(addr string) (fwder SafeFwder, err error) {
	if fwder, ok := repo.fwders[addr]; ok {
		return fwder, nil
	} else {
		udpFwder, err := NewSafeUDPFwder(addr, repo.fwderTimeout, repo.timeoutLasting)
		if err == nil {
			fwder := NewRecoverableFwder(udpFwder, repo.prober)
			repo.fwders[addr] = fwder
			return fwder, nil
		} else {
			return nil, err
		}
	}
}
