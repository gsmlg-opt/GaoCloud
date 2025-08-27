package forwarder

import (
	"sort"
	"sync"
	"time"
)

type SelectorBase struct {
	fwders []SafeFwder
}

func (s *SelectorBase) GetFwders() []SafeFwder {
	return s.fwders
}

func (s *SelectorBase) selectNextUsableFwder(fromIndex int) (SafeFwder, int) {
	fwderCount := len(s.fwders)
	for i := 0; i < fwderCount; i++ {
		f := s.fwders[fromIndex]
		if f.IsDown() == false {
			return f, fromIndex
		} else {
			fromIndex = (fromIndex + 1) % fwderCount
		}
	}
	return nil, -1
}

func (s *SelectorBase) HasUpFwder() bool {
	for _, fwder := range s.fwders {
		if fwder.IsDown() == false {
			return true
		}
	}
	return false
}

func (s *SelectorBase) SetQuerySource(ip string) error {
	for _, fwder := range s.fwders {
		if fwder.IsDown() == false {
			if err := fwder.SetQuerySource(ip); err != nil {
				return err
			}
		}
	}

	return nil
}

type FixOrderSelector struct {
	SelectorBase
	lock sync.Mutex
}

func newFixOrderSelector(fwders []SafeFwder) *FixOrderSelector {
	return &FixOrderSelector{
		SelectorBase: SelectorBase{
			fwders: fwders,
		},
	}
}

func (s *FixOrderSelector) SelectFwder() SafeFwder {
	s.lock.Lock()
	defer s.lock.Unlock()
	f, _ := s.selectNextUsableFwder(0)
	return f
}

type RoundRobinSelector struct {
	SelectorBase
	lock sync.Mutex
	next int
}

func newRoundRobinSelector(fwders []SafeFwder) *RoundRobinSelector {
	return &RoundRobinSelector{
		SelectorBase: SelectorBase{
			fwders: fwders,
		},
		next: 0,
	}
}

func (s *RoundRobinSelector) SelectFwder() SafeFwder {
	s.lock.Lock()
	defer s.lock.Unlock()

	f, target := s.selectNextUsableFwder(s.next)
	s.next = (target + 1) % len(s.fwders)
	return f
}

type RttBasedSelector struct {
	SelectorBase
	rttSnapShot []time.Duration
	lock        sync.Mutex
}

func newRttBasedSelector(fwders []SafeFwder) *RttBasedSelector {
	return &RttBasedSelector{
		SelectorBase: SelectorBase{
			fwders: fwders,
		},
		rttSnapShot: make([]time.Duration, len(fwders), len(fwders)),
	}
}

func (s *RttBasedSelector) SelectFwder() SafeFwder {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.sortForwarders()
	f, _ := s.selectNextUsableFwder(0)
	return f
}

func (s *RttBasedSelector) sortForwarders() {
	for i, f := range s.fwders {
		s.rttSnapShot[i] = f.GetLastRtt()
	}
	sort.Sort(s)
}

func (s *RttBasedSelector) Len() int {
	return len(s.fwders)
}

func (s *RttBasedSelector) Less(i, j int) bool {
	return s.rttSnapShot[i] < s.rttSnapShot[j]
}

func (s *RttBasedSelector) Swap(i, j int) {
	s.fwders[i], s.fwders[j] = s.fwders[j], s.fwders[i]
	s.rttSnapShot[i], s.rttSnapShot[j] = s.rttSnapShot[j], s.rttSnapShot[i]
}
