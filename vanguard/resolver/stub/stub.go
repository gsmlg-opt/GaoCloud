package stub

import (
	"fmt"
	"sync"
	"time"

	"cement/domaintree"
	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/httpcmd"
	"vanguard/logger"
	"vanguard/resolver/chain"
	"vanguard/resolver/querysource"
	"vanguard/util"
	view "vanguard/viewselector"
)

const sendTimeout = 3 * time.Second

type StubZoneManager struct {
	chain.DefaultResolver
	stubZones map[string]*domaintree.DomainTree
	sender    *util.SafeUDPSender
	lock      sync.RWMutex
}

func NewStubZoneManager(conf *config.VanguardConf) *StubZoneManager {
	stub := &StubZoneManager{}
	stub.ReloadConfig(conf)
	httpcmd.RegisterHandler(stub, []httpcmd.Command{&AddStubZone{}, &DeleteStubZone{}, &UpdateStubZone{}})
	return stub
}

func (mgr *StubZoneManager) ReloadConfig(conf *config.VanguardConf) {
	stubZones := make(map[string]*domaintree.DomainTree)
	for view, _ := range view.GetViewAndIds() {
		stubZones[view] = domaintree.NewDomainTree()
	}

	for _, c := range conf.Stub {
		tree := stubZones[c.View]
		for _, zone := range c.Zones {
			origin, err := g53.NameFromString(zone.Name)
			if err != nil {
				panic("stub zone name" + zone.Name + " failed:" + err.Error())
			}

			if _, err := tree.Insert(origin, zone.Masters); err != nil {
				panic("load stub zone " + zone.Name + " failed:" + err.Error())
			}
		}
	}
	mgr.stubZones = stubZones
}

func (z *StubZoneManager) Resolve(client *core.Client) {
	if sender, err := util.NewSafeUDPSender(querysource.GetQuerySource(client.View), sendTimeout); err != nil {
		logger.GetLogger().Error("stub zone query %s failed: %s", client.Request.Question.String(), err.Error())
		return
	} else {
		z.sender = sender
	}

	if client.Response != nil && util.ClassifyResponse(client.Response) == util.REFERRAL {
		return
	}

	request := client.Request
	masters, result := z.getMasters(client.View, request.Question.Name)
	if result != domaintree.NotFound {
		response, err := z.handleQuery(request, masters)
		if err != nil {
			response = request.MakeResponse()
			response.Header.Rcode = g53.R_SERVFAIL
			logger.GetLogger().Error("stub zone query %s failed: %s",
				request.Question.Name.String(false), err.Error())
		} else {
			logger.GetLogger().Debug("stub zone query %s succeed with rcode: %s",
				request.Question.Name.String(false), response.Header.Rcode.String())
		}
		client.CacheAnswer = false
		client.Response = response
	} else {
		logger.GetLogger().Debug("no stub zone is found for rrset %s with view %s",
			request.Question.Name.String(false), client.View)
		chain.PassToNext(z, client)
	}
}

func (z *StubZoneManager) getMasters(viewName string, name *g53.Name) ([]string, domaintree.SearchResult) {
	z.lock.RLock()
	zones, ok := z.stubZones[viewName]
	z.lock.RUnlock()
	if ok == false {
		return nil, domaintree.NotFound
	}

	_, masters, result := zones.Search(name)
	if result != domaintree.NotFound {
		return masters.([]string), result
	} else {
		return nil, result
	}
}

func (z *StubZoneManager) handleQuery(request *g53.Message, masters []string) (response *g53.Message, err error) {
	masterLen := len(masters)
	if masterLen == 1 {
		response, _, err = z.sender.Query(masters[0], request)
	} else {
		resultChan := make(chan *g53.Message, masterLen)
		for _, master := range masters {
			go func(addr string) {
				resp, _, err := z.sender.Query(addr, request)
				if err == nil {
					resultChan <- resp
				} else {
					logger.GetLogger().Debug("stub zone query %s with nameserver %s failed:%s",
						request.Question.Name.String(false), addr, err.Error())
				}
			}(master)
		}

		select {
		case response = <-resultChan:
		case <-time.After(sendTimeout):
			err = fmt.Errorf("query timeout")
		}
	}
	return
}
