package cache

import (
	"sync"

	"vanguard/core"
	"vanguard/logger"
)

const defaultTaskChanBuf = 1024

type PrefetchTask struct {
	ctx *core.Context
}

type Prefetcher struct {
	handler  core.DNSQueryHandler
	cache    *MessageCache
	taskChan chan *PrefetchTask
	stopChan chan struct{}
	taskKeys map[uint64]struct{}
	taskLock sync.Mutex
}

func newPrefetcher(handler core.DNSQueryHandler, cache *MessageCache) *Prefetcher {
	return &Prefetcher{
		handler:  handler,
		cache:    cache,
		stopChan: make(chan struct{}),
	}
}

func (p *Prefetcher) reloadConfig() {
	p.taskChan = make(chan *PrefetchTask, defaultTaskChanBuf)
	p.taskKeys = make(map[uint64]struct{})
}

func (p *Prefetcher) run() {
	var task *PrefetchTask
	for {
		select {
		case <-p.stopChan:
			p.stopChan <- struct{}{}
			return
		case task = <-p.taskChan:
			core.PassToNext(p.handler, task.ctx)
			if task.ctx.Client.Response != nil && task.ctx.Client.CacheAnswer {
				p.cache.Add(task.ctx.Client.Response)
				p.deletePrefetchTask(task.ctx.Client.QueryKey())
			}
		}
	}
}

func (p *Prefetcher) stop() {
	p.stopChan <- struct{}{}
	<-p.stopChan
	close(p.taskChan)
}

func (p *Prefetcher) addPrefetchTask(client *core.Client) {
	p.taskLock.Lock()
	defer p.taskLock.Unlock()
	key := client.QueryKey()
	if _, ok := p.taskKeys[key]; ok == false {
		select {
		case p.taskChan <- &PrefetchTask{
			ctx: core.NewContext().Clone(client),
		}:
			p.taskKeys[key] = struct{}{}
		default:
			logger.GetLogger().Warn("cache prefetch task chan is full and abandon %s with view %s",
				client.Request.Question.Name.String(false), client.View)
		}
	}
}

func (p *Prefetcher) deletePrefetchTask(key uint64) {
	p.taskLock.Lock()
	delete(p.taskKeys, key)
	p.taskLock.Unlock()
}
