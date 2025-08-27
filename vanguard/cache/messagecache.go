package cache

import (
	"container/list"
	"sync"
	"time"

	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
)

const (
	defaultNegativeCacheTtl uint32 = 60   //1 minute
	defaultPositiveCacheTtl uint32 = 3600 //1 Hour
	defaultMaxCacheSize     uint   = 0
	prefetchTime                   = 10
)

type Key uint64

type MessageCacheEntry struct {
	message    *g53.Message
	expireTime time.Time
}

func (e *MessageCacheEntry) Message() *g53.Message {
	return e.message
}

func (e *MessageCacheEntry) IsExpire() bool {
	return e.expireTime.Before(time.Now())
}

func (e *MessageCacheEntry) NeedPrefetch() bool {
	return e.expireTime.Before(time.Now().Add(prefetchTime * time.Second))
}

type MessageCache struct {
	positiveTtl  uint32
	negativeTtl  uint32
	maxSize      uint
	shortAnswer  bool
	needPrefetch bool

	ll         *list.List
	cache      map[Key]*list.Element
	lock       sync.RWMutex
	prefetcher *Prefetcher
}

func newMessageCache(conf *config.CacheConf, handler core.DNSQueryHandler) *MessageCache {
	c := &MessageCache{
		ll:    list.New(),
		cache: make(map[Key]*list.Element),
	}

	c.prefetcher = newPrefetcher(handler, c)
	c.reloadConfig(conf)
	return c
}

func (c *MessageCache) reloadConfig(conf *config.CacheConf) {
	if c.needPrefetch {
		c.prefetcher.stop()
	}

	positiveTtl := conf.PositiveTtl
	if positiveTtl == 0 {
		positiveTtl = defaultPositiveCacheTtl
	}

	negativeTtl := conf.NegativeTtl
	if negativeTtl == 0 {
		negativeTtl = defaultNegativeCacheTtl
	}

	c.positiveTtl = positiveTtl
	c.negativeTtl = negativeTtl
	c.maxSize = conf.MaxCacheSize
	c.shortAnswer = conf.ShortAnswer

	if conf.Prefetch {
		c.needPrefetch = true
		c.prefetcher.reloadConfig()
		go c.prefetcher.run()
	}
}

func (c *MessageCache) Add(message *g53.Message) {
	entry := c.messageToCache(message)
	if entry == nil {
		return
	}

	key := keyForMessage(message.Question.Name, message.Question.Type)
	c.lock.Lock()
	c.add(key, entry)
	c.lock.Unlock()
}

func (c *MessageCache) add(key Key, entry *MessageCacheEntry) {
	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		elem.Value = entry
	} else {
		elem := c.ll.PushFront(entry)
		c.cache[key] = elem
	}
	if c.maxSize != defaultMaxCacheSize && uint(c.ll.Len()) > c.maxSize {
		logger.GetLogger().Debug("cache messages size %v exceeded max size %v, will remove oldest one",
			c.ll.Len(), c.maxSize)
		c.removeOldest()
	}
}

func (c *MessageCache) messageToCache(message *g53.Message) *MessageCacheEntry {
	message.Header.SetFlag(g53.FLAG_RA, true)

	answers := message.Sections[g53.AnswerSection]
	ancount := len(answers)
	if ancount > 0 {
		return c.positiveMessageToCache(message)
	} else {
		return c.negativeMessageToCache(message)
	}
}

func (c *MessageCache) negativeMessageToCache(message *g53.Message) *MessageCacheEntry {
	auths := message.Sections[g53.AuthSection]
	minTtl := c.negativeTtl
	//auth section may includes soa
	if len(auths) == 1 && auths[0].Type == g53.RR_SOA && len(auths[0].Rdatas) == 1 {
		soa := auths[0]
		if rdata, ok := soa.Rdatas[0].(*g53.SOA); ok {
			if minTtl > rdata.Minimum {
				minTtl = rdata.Minimum
			}
			soaTtl := uint32(soa.Ttl)
			if soaTtl < minTtl {
				minTtl = soaTtl
			} else {
				soa.Ttl = g53.RRTTL(minTtl)
			}
		}
	}

	if c.shortAnswer {
		message.ClearSection(g53.AdditionalSection)
	}

	return &MessageCacheEntry{
		message:    message,
		expireTime: time.Now().Add(time.Duration(minTtl) * time.Second),
	}
}

func (c *MessageCache) positiveMessageToCache(message *g53.Message) *MessageCacheEntry {
	if c.shortAnswer {
		message.ClearSection(g53.AuthSection)
		message.ClearSection(g53.AdditionalSection)
	}

	answers := message.Sections[g53.AnswerSection]
	minTtl := c.positiveTtl
	ancount := len(answers)
	for i := 0; i < ancount; i++ {
		ttl := uint32(answers[i].Ttl)
		if ttl < minTtl {
			minTtl = ttl
		} else if ttl > c.positiveTtl {
			answers[i].Ttl = g53.RRTTL(c.positiveTtl)
		}
	}

	return &MessageCacheEntry{
		message:    message,
		expireTime: time.Now().Add(time.Second * time.Duration(minTtl)),
	}
}

func keyForMessage(name *g53.Name, typ g53.RRType) Key {
	hash := uint64(name.Hash(false))
	return Key((hash << 32) | uint64(typ))
}

func (c *MessageCache) Get(client *core.Client) (*g53.Message, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if entry, found := c.get(client.Request.Question.Name, client.Request.Question.Type); found {
		if c.needPrefetch && entry.NeedPrefetch() {
			c.prefetcher.addPrefetchTask(client)
		}
		return entry.Message(), true
	} else {
		return nil, false
	}
}

func (c *MessageCache) get(name *g53.Name, typ g53.RRType) (*MessageCacheEntry, bool) {
	key := keyForMessage(name, typ)
	if elem, hit := c.cache[key]; hit {
		entry := elem.Value.(*MessageCacheEntry)
		if entry.IsExpire() == false && entry.message.Question.Name.Equals(name) {
			c.ll.MoveToFront(elem)
			roundrobinAnswer(entry.message)
			return entry, true
		}
	}
	return nil, false
}

func (c *MessageCache) Remove(name *g53.Name, typ g53.RRType) {
	key := keyForMessage(name, typ)
	c.lock.Lock()
	defer c.lock.Unlock()
	if ele, hit := c.cache[key]; hit {
		c.removeElement(ele)
	}
}

func (c *MessageCache) removeOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *MessageCache) removeElement(e *list.Element) {
	c.ll.Remove(e)
	message := e.Value.(*MessageCacheEntry).message
	key := keyForMessage(message.Question.Name, message.Question.Type)
	delete(c.cache, key)
}

func (c *MessageCache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.ll.Len()
}

func (c *MessageCache) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.ll.Init()
	c.cache = make(map[Key]*list.Element)
}

func roundrobinAnswer(msg *g53.Message) {
	answers := msg.Sections[g53.AnswerSection]
	for _, rrset := range answers {
		if len(rrset.Rdatas) > 1 {
			rrset.RotateRdata()
		}
	}
}
