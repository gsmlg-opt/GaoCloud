package main

import (
	"flag"
	"fmt"

	"cement/shell"
	"cement/signal"
	"vanguard/acl"
	"vanguard/cache"
	"vanguard/config"
	"vanguard/core"
	"vanguard/dns64"
	"vanguard/failforwarder"
	"vanguard/filter"
	"vanguard/httpcmd"
	"vanguard/k8s"
	"vanguard/logger"
	"vanguard/metrics"
	"vanguard/querylog"
	"vanguard/resolver"
	"vanguard/responsetransfer"
	"vanguard/server"
	view "vanguard/viewselector"
	"vanguard/xfr"
)

var (
	version string
	build   string
)

var (
	configFile   string
	showVersion  bool
	maxOpenFiles uint64
)

const (
	ModuleView          = "view"
	ModuleFilter        = "filter"
	ModuleCache         = "cache"
	ModuleDNS64         = "dns64"
	ModuleResolver      = "resolver"
	ModuleKubernetes    = "kubernetes"
	ModuleFailForwarder = "fail_forwarder"
	ModuleQueryLog      = "query_log"
	ModuleAAAAFilter    = "aaaa_filter"
	ModuleHijack        = "hijack"
	ModuleSortList      = "sort_list"
)

func init() {
	flag.StringVar(&configFile, "c", "/etc/vanguard/vanguard.conf", "configure file path")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Uint64Var(&maxOpenFiles, "u", 0, "set max open files")
	if version == "" {
		version = "unknown"
	}
	if build == "" {
		build = "unknown"
	}
}

func main() {
	flag.Parse()
	if showVersion {
		fmt.Printf("vanguard %s (build at %s)\n", version, build)
		return
	}

	if maxOpenFiles > 0 {
		if err := shell.SetULimit(maxOpenFiles); err != nil {
			panic("set ulimit failed:" + err.Error())
		}
	}

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		panic("load configure file failed:" + err.Error())
	}

	if err := logger.InitLogger(conf); err != nil {
		panic("init logger failed:" + err.Error())
	}

	acl.NewAclManager(conf)
	queryHandler, xfrHandler := createHandler(conf)
	server, err := server.NewServer(conf, queryHandler, xfrHandler)
	if err != nil {
		panic("create server failed:" + err.Error())
	}

	go metrics.NewMetrics(conf).Run()
	go httpcmd.NewCmdService(conf).Run()
	server.Run()
	signal.WaitForInterrupt(func() {
		server.Shutdown()
		logger.GetLogger().Info("server get interrupted and going to exit")
		logger.GetLogger().Close()
	})
}

type ModuleCreator func(*config.VanguardConf) core.DNSQueryHandler

var supported_creators = map[string]ModuleCreator{
	ModuleView:          view.NewSelectorMgr,
	ModuleFilter:        filter.NewFilterChain,
	ModuleCache:         cache.NewCache,
	ModuleDNS64:         dns64.NewDNS64,
	ModuleKubernetes:    k8s.NewK8SAuth,
	ModuleFailForwarder: failforwarder.NewFailForwarder,
	ModuleQueryLog:      querylog.NewQuerylog,
	ModuleAAAAFilter:    responsetransfer.NewAAAAFilter,
	ModuleHijack:        responsetransfer.NewHijack,
	ModuleSortList:      responsetransfer.NewSortList,
}

var moduleInOrder = []string{
	ModuleQueryLog,
	ModuleKubernetes,
	ModuleView,
	ModuleFilter,
	ModuleAAAAFilter,
	ModuleSortList,
	ModuleHijack,
	ModuleDNS64,
	ModuleCache,
	ModuleResolver,
	ModuleFailForwarder,
}

func createHandler(conf *config.VanguardConf) (core.DNSQueryHandler, core.DNSQueryHandler) {
	creator := make(map[string]ModuleCreator)
	resolverEnable := false
	for _, m := range conf.EnableModules {
		c, ok := supported_creators[m]
		if ok {
			creator[m] = c
		} else {
			if resolver.IsResolverSubModule(m) == false {
				panic("unknown module:" + m)
			}
			resolverEnable = true
		}
	}

	var viewSelector *view.SelectorMgr
	var resol *resolver.ResolverManager

	var handlers []core.DNSQueryHandler
	for _, m := range moduleInOrder {
		var h core.DNSQueryHandler
		if m == ModuleResolver {
			if resolverEnable {
				h = resolver.NewResolver(conf)
			} else {
				continue
			}
		} else if c, ok := creator[m]; ok {
			h = c(conf)
		} else {
			continue
		}

		if m == ModuleView {
			viewSelector = h.(*view.SelectorMgr)
		} else if m == ModuleResolver {
			resol = h.(*resolver.ResolverManager)
		}
		handlers = append(handlers, h)
	}

	if len(handlers) == 0 {
		panic("no hanlder is created")
	}
	core.BuildQueryChain(handlers...)

	var xfrHandler core.DNSQueryHandler
	if viewSelector != nil && resol != nil && resol.Auth != nil {
		xfrHandler = xfr.NewXFRHandler(viewSelector, resol.Auth)
	}
	return handlers[0], xfrHandler
}
