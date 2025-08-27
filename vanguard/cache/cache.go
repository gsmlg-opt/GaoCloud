package cache

import (
	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/httpcmd"
	"vanguard/metrics"
	view "vanguard/viewselector"
)

type Cache struct {
	core.DefaultHandler
	cache map[string]*MessageCache
}

func NewCache(conf *config.VanguardConf) core.DNSQueryHandler {
	c := &Cache{}
	c.ReloadConfig(conf)
	httpcmd.RegisterHandler(c, []httpcmd.Command{&CleanCache{}, &CleanViewCache{}, &CleanDomainCache{}, &CleanRRsetsCache{}, &GetDomainCache{}, &GetMessageCache{}})
	return c
}

func (c *Cache) ReloadConfig(conf *config.VanguardConf) {
	cache := make(map[string]*MessageCache)
	for view, _ := range view.GetViewAndIds() {
		if _, exist := c.cache[view]; exist {
			cache[view] = c.cache[view]
			cache[view].reloadConfig(&conf.Cache)
		} else {
			cache[view] = newMessageCache(&conf.Cache, c)
		}
	}

	if defaultCache, exist := cache[view.DefaultView]; exist == false {
		cache[view.DefaultView] = newMessageCache(&conf.Cache, c)
	} else {
		defaultCache.reloadConfig(&conf.Cache)
	}

	c.cache = cache
}

func (c *Cache) HandleQuery(ctx *core.Context) {
	client := &ctx.Client
	message, found := c.get(client)
	client.CacheHit = found
	if found == true {
		metrics.RecordCacheHit(client.View)
		response := *message
		response.Header.Id = client.Request.Header.Id
		response.Header.SetFlag(g53.FLAG_AA, false)
		response.Question = client.Request.Question
		client.Response = &response
	} else {
		core.PassToNext(c, ctx)
		if client.Response != nil && client.CacheAnswer {
			c.AddMessage(client.View, client.Response)
		}
	}
}

func (c *Cache) AddMessage(view string, message *g53.Message) {
	if messageCache, ok := c.cache[view]; ok {
		messageCache.Add(message)
	}
}

func (c *Cache) get(client *core.Client) (*g53.Message, bool) {
	if messageCache, ok := c.cache[client.View]; ok {
		return messageCache.Get(client)
	} else {
		return nil, false
	}
}
