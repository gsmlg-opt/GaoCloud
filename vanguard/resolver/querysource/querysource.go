package querysource

import (
	"sync"

	"vanguard/config"
	"vanguard/httpcmd"
)

var gQuerySourceManager *QuerySourceManager

type QuerySourceManager struct {
	querySources map[string]string
	lock         sync.RWMutex
}

func NewQuerySourceManager(conf *config.VanguardConf) {
	gQuerySourceManager = &QuerySourceManager{}
	ReloadConfig(conf)
	httpcmd.RegisterHandler(gQuerySourceManager, []httpcmd.Command{&AddQuerySource{}, &DeleteQuerySource{}, &UpdateQuerySource{}})
}

func ReloadConfig(conf *config.VanguardConf) {
	querySources := make(map[string]string)
	for _, c := range conf.QuerySource {
		querySources[c.View] = c.Address
	}
	gQuerySourceManager.querySources = querySources
}

func GetQuerySource(view string) string {
	gQuerySourceManager.lock.RLock()
	defer gQuerySourceManager.lock.RUnlock()
	if addr, ok := gQuerySourceManager.querySources[view]; ok {
		return addr
	} else if addr, ok := gQuerySourceManager.querySources[DefaultViewForQuery]; ok {
		return addr
	} else {
		return ""
	}
}
