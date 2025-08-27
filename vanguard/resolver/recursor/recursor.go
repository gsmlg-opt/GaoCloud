package recursor

import (
	"errors"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
	"vanguard/resolver/chain"
	"vanguard/resolver/querysource"
	"vanguard/util"
	view "vanguard/viewselector"
)

var errTooDepQuery = errors.New("query exceed depth limit")
var errInvalidResponse = errors.New("response is invalid")
var errQueryTimeout = errors.New("query time out")
var errDumbNameServer = errors.New("auth name server is dumb")

const maxQueryDep = 20
const maxInflightQuery = 100
const singleQueryTimeout = 3 * time.Second
const queryTimeout = 15 * time.Second
const batchQueryCount = 3 //max server to query in parallel
const memoryCheckInterval = 10 * time.Second

var rootServers = map[string]string{
	"a.root-servers.net.": "198.41.0.4:53",
	"b.root-servers.net.": "192.228.79.201:53",
	"c.root-servers.net.": "192.33.4.12:53",
	"d.root-servers.net.": "199.7.91.13:53",
	"e.root-servers.net.": "192.203.230.10:53",
	"f.root-servers.net.": "192.5.5.241:53",
	"g.root-servers.net.": "192.112.36.4:53",
	"h.root-servers.net.": "198.97.190.53:53",
	"i.root-servers.net.": "192.36.148.17:53",
	"j.root-servers.net.": "192.58.128.30:53",
	"k.root-servers.net.": "193.0.14.129:53",
	"l.root-servers.net.": "199.7.83.42:53",
	"m.root-servers.net.": "202.12.27.33:53",
}

type Recursor struct {
	chain.DefaultResolver
	nsasCache        *NsasCache
	ednsSubnetEnable map[string]bool
	resolverEnable   map[string]bool
	rootForView      map[string][]*NameServer
	ctxPool          *RecursorCtxPool
	stopCh           chan struct{}
}

func NewRecursor(conf *config.VanguardConf) *Recursor {
	r := &Recursor{
		ctxPool: newRecursorCtxPool(maxInflightQuery),
		stopCh:  make(chan struct{}),
	}
	r.ReloadConfig(conf)
	return r
}

func (r *Recursor) ReloadConfig(conf *config.VanguardConf) {
	r.stopMemoryEnforce()
	ednsSubnetEnable := make(map[string]bool)
	resolverEnable := make(map[string]bool)
	rootServers := make(map[string][]*NameServer)
	for _, c := range conf.Recursor {
		resolverEnable[c.View] = c.Enable
		ednsSubnetEnable[c.View] = c.EdnsSubnetEnable

		if c.RootHintFile != "" {
			f, err := os.OpenFile(c.RootHintFile, os.O_RDONLY, 0755)
			if err != nil {
				panic("open root hint file " + c.RootHintFile + " failed " + err.Error())
			}
			defer f.Close()
			content, err := ioutil.ReadAll(f)
			if err != nil {
				panic("read zone file " + c.RootHintFile + " failed " + err.Error())
			}
			nameServers, err := loadRootServer(string(content))
			if err != nil {
				panic("load root server failed:" + err.Error())
			}
			rootServers[c.View] = nameServers
		}
	}

	defaultRootServers := getDefaultRootServers()
	for view, _ := range view.GetViewAndIds() {
		if _, ok := rootServers[view]; ok == false {
			rootServers[view] = defaultRootServers
		}
	}
	r.ednsSubnetEnable = ednsSubnetEnable
	r.rootForView = rootServers
	r.resolverEnable = resolverEnable
	r.nsasCache = NewNsasCache(0)
	go r.enforceMemoryUsage(r.stopCh)
}

func (r *Recursor) Resolve(client *core.Client) {
	if enable, ok := r.resolverEnable[client.View]; ok && enable {
		r.resolver(client)
		if client.Response != nil {
			client.Response.Header.SetFlag(g53.FLAG_RA, true)
		}
	}

	if client.Response == nil {
		chain.PassToNext(r, client)
	}
}

func (r *Recursor) resolver(client *core.Client) {
	ctx := r.ctxPool.getCtx()
	if ctx == nil {
		logger.GetLogger().Error("out recusive query exceed limit")
		return
	}
	defer r.ctxPool.putCtx(ctx)

	clientAddress := ""
	if r.ednsSubnetEnable[client.View] {
		clientAddress = client.Addr.String()
	}

	ctx.init(singleQueryTimeout, querysource.GetQuerySource(client.View), clientAddress, client.Request.Question, r.getRootServers(client.View))

	var response *g53.Message
	var err error
	if client.Response != nil && util.ClassifyResponse(client.Response) == util.REFERRAL {
		//use root here to let recursor trust the response
		response, err = r.handleReferal(ctx, g53.Root, client.Response)
	} else {
		response, err = r.handleQuery(ctx)
	}

	if err == nil {
		finalResponse := *response
		finalResponse.Header.Id = client.Request.Header.Id
		client.Response = &finalResponse
		logger.GetLogger().Debug("query %s succeed and take %.1f milliseconds", client.Request.Question.String(), time.Since(ctx.startTime).Seconds()*1000)
	} else {
		logger.GetLogger().Error("query %s failed %s", client.Request.Question.String(), err.Error())
	}
}

func (r *Recursor) getRootServers(view string) []*NameServer {
	nameServers, ok := r.rootForView[view]
	if ok == false {
		panic("unkown view " + view)
	}
	return cloneNameServers(nameServers)
}

func (r *Recursor) handleQuery(ctx *RecursorCtx) (*g53.Message, error) {
	ctx.depth += 1
	if ctx.depth >= maxQueryDep || (ctx.depth > 1 && time.Since(ctx.startTime) > queryTimeout) {
		return nil, errTooDepQuery
	}

	nameServers := r.nsasCache.SelectNameServers(ctx.question.Name)
	if nameServers == nil {
		nameServers = ctx.nameServers
	}

	request := g53.MakeQuery(ctx.question.Name, ctx.question.Type, 4096, false)
	request.Edns.AddSubnetV4(ctx.clientAddress)
	request.Header.SetFlag(g53.FLAG_RD, false)
	request.RecalculateSectionRRCount()
	response, err := r.doQuery(ctx.sender, nameServers, request)
	if err == nil {
		return r.handleResponse(ctx, nameServers[0].zone, response)
	} else {
		return r.handleQuery(ctx)
	}
}

type Responder struct {
	server   *NameServer
	response *g53.Message
}

func (r *Recursor) doQuery(sender *util.SafeUDPSender, servers []*NameServer, request *g53.Message) (response *g53.Message, err error) {
	serverCount := len(servers)
	if serverCount == 1 {
		return r.doSingleQuery(sender, servers[0], request)
	} else {
		if serverCount > batchQueryCount {
			sort.Sort(ServerByRtt(servers))
			servers = servers[:batchQueryCount]
			serverCount = batchQueryCount
		}
		resultChan := make(chan Responder, serverCount)
		for _, server := range servers {
			go func(s *NameServer) {
				msg, err := r.doSingleQuery(sender, s, request)
				if err == nil {
					resultChan <- Responder{s, msg}
				}
			}(server)
		}

		select {
		case responder := <-resultChan:
			response = responder.response
			logger.GetLogger().Debug("from [%s] get response:\n%s", responder.server.String(), response.String())
		case <-time.After(singleQueryTimeout):
			err = errQueryTimeout
		}
	}
	return
}

func (r *Recursor) doSingleQuery(sender *util.SafeUDPSender, server *NameServer, request *g53.Message) (*g53.Message, error) {
	logger.GetLogger().Debug("send query %s to name server %s", request.Question.String(), server.String())

	response, rtt, err := sender.Query(server.addr, request)
	if err != nil {
		logger.GetLogger().Error("send query %s to name server %s get err %s", request.Question.String(), server.String(), err.Error())
	}

	if response != nil {
		if response.Header.Rcode == g53.R_FORMERR {
			requstWithoutEdns := *request
			requstWithoutEdns.Edns = nil
			requstWithoutEdns.RecalculateSectionRRCount()
			response, rtt, err = sender.Query(server.addr, &requstWithoutEdns)
		} else if isValidResponse(response) == false {
			rtt = queryTimeout
			err = errDumbNameServer
		}
	}

	r.nsasCache.UpdateRtt(server, rtt)
	return response, err
}

func getDefaultRootServers() []*NameServer {
	roots := make([]*NameServer, 0, len(rootServers))
	for name, addr := range rootServers {
		serverName, _ := g53.NameFromString(name)
		roots = append(roots, &NameServer{
			zone: g53.Root,
			name: serverName,
			addr: addr,
		})
	}
	return roots
}

func (r *Recursor) handleResponse(ctx *RecursorCtx, zone *g53.Name, response *g53.Message) (*g53.Message, error) {
	switch util.ClassifyResponse(response) {
	case util.ANSWER, util.NXDOMAIN, util.NXRRSET:
		return r.handleFinalAnswer(ctx, zone, response)
	case util.REFERRAL:
		return r.handleReferal(ctx, zone, response)
	default:
		return nil, errInvalidResponse
	}
}

func (r *Recursor) handleFinalAnswer(ctx *RecursorCtx, zone *g53.Name, response *g53.Message) (*g53.Message, error) {
	r.nsasCache.AddZoneNameServer(zone, response)
	response.Question = ctx.question
	return response, nil
}

func (r *Recursor) handleReferal(ctx *RecursorCtx, zone *g53.Name, response *g53.Message) (*g53.Message, error) {
	missingServers, knownServers := r.nsasCache.AddZoneNameServer(zone, response)
	if len(missingServers) > 0 {
		r.getMissingNameServer(ctx, missingServers, len(knownServers) == 0)
	}
	return r.handleQuery(ctx)
}

func (r *Recursor) getMissingNameServer(ctx *RecursorCtx, serverNames []*g53.Name, wait bool) {
	queryDepth := ctx.depth
	if queryDepth > maxQueryDep {
		return
	}

	doneChan := make(chan struct{})
	outQuery := 0
	for i := 0; i < len(serverNames); i++ {
		newCtx := r.ctxPool.getCtx()
		if newCtx == nil {
			logger.GetLogger().Error("out recusive query exceed limit")
			continue
		}
		newCtx.init(singleQueryTimeout, ctx.sender.GetQuerySource(), ctx.clientAddress,
			&g53.Question{
				Name:  serverNames[i],
				Type:  g53.RR_A,
				Class: g53.CLASS_IN,
			}, cloneNameServers(ctx.nameServers))
		newCtx.depth = queryDepth
		outQuery += 1
		go func(ctx_ *RecursorCtx) {
			defer r.ctxPool.putCtx(ctx_)
			response, err := r.handleQuery(ctx_)
			if err != nil {
				return
			}

			glue := getARRsetFromAnswer(response)
			if glue == nil {
				return
			}

			r.nsasCache.addNameServer(glue, FromAuth)
			select {
			case doneChan <- struct{}{}:
			default:
			}
		}(newCtx)
	}

	if wait && outQuery > 0 {
		select {
		case <-doneChan:
		case <-time.After(singleQueryTimeout):
		}
	}
}

func (r *Recursor) stopMemoryEnforce() {
	close(r.stopCh)
	r.stopCh = make(chan struct{})
}

func (r *Recursor) enforceMemoryUsage(stopCh <-chan struct{}) {
	ticker := time.NewTicker(memoryCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
		}
		r.nsasCache.EnforceMemoryLimit()
	}
}
