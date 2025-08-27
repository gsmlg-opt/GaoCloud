package forwarder

import (
	"container/list"
	"sync"
	"time"

	"cement/set"
	"cement/workerpool"
	"g53"
	"vanguard/logger"
)

const probeRoutineCount = 10

type Target struct {
	fwder   SafeFwder
	request *g53.Message
}

type Prober struct {
	interval time.Duration

	targets      *list.List
	targetFwders *set.Set
	targetsLock  sync.Mutex

	proberPool *workerpool.WorkerPool
	stop       chan struct{}
}

func NewProber(interval time.Duration) *Prober {
	p := &Prober{
		interval:     interval,
		targets:      &list.List{},
		targetFwders: set.NewSet(),
		stop:         make(chan struct{}),
	}
	p.startProber()
	go p.run()

	return p
}

func (p *Prober) startProber() {
	prober := func(t workerpool.Task) {
		target := t.(*Target)
		_, _, err := target.fwder.Forward(target.request)
		if err != nil {
			p.targetsLock.Lock()
			p.targets.PushBack(target)
			p.targetsLock.Unlock()
		} else {
			p.targetsLock.Lock()
			p.targetFwders.Remove(target.fwder)
			p.targetsLock.Unlock()
			logger.GetLogger().Info(target.fwder.RemoteAddr() + " restored")
		}
	}
	p.proberPool = workerpool.NewWorkerPool(prober, probeRoutineCount)
}

func (p *Prober) AddProbe(f SafeFwder, request *g53.Message) {
	p.targetsLock.Lock()
	if p.targetFwders.Contains(f) == false {
		logger.GetLogger().Info(f.RemoteAddr() + " broken")
		p.targets.PushBack(&Target{fwder: f, request: request})
		p.targetFwders.Add(f)
	}
	p.targetsLock.Unlock()
}

func (p *Prober) run() {
	timer := time.NewTicker(p.interval)
	defer timer.Stop()

	for {
		select {
		case <-p.stop:
			goto done
		case <-timer.C:
			probeTargets := make([]*Target, 0, probeRoutineCount)
			p.targetsLock.Lock()
			for i := 0; i < probeRoutineCount; i++ {
				target := p.targets.Front()
				if target == nil {
					break
				} else {
					p.targets.Remove(target)
					probeTargets = append(probeTargets, target.Value.(*Target))
				}
			}
			p.targetsLock.Unlock()

			for _, target := range probeTargets {
				if err := p.proberPool.SendTask(target); err != nil {
					if err == workerpool.ErrWorkerPoolIsStopped {
						goto done
					} else if err == workerpool.ErrWorkerPoolIsTooBusy {
						logger.GetLogger().Warn("prober is very busy")
						p.targetsLock.Lock()
						p.targets.PushBack(target)
						p.targetsLock.Unlock()
					} else {
						panic("only two kinds of error should occur")
					}
				} else {
					logger.GetLogger().Debug("send %s to probe", target.fwder.RemoteAddr())
				}
			}
		}
	}

done:
	p.stop <- struct{}{}
}

func (p *Prober) Stop() {
	if p.proberPool.Stop() == nil {
		p.stop <- struct{}{}
		<-p.stop
	}
}
