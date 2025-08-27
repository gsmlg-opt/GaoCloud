package resolver

import (
	"vanguard/config"
	"vanguard/core"
	"vanguard/resolver/auth"
	"vanguard/resolver/chain"
	"vanguard/resolver/fakeauth"
	"vanguard/resolver/forwarder"
	"vanguard/resolver/querysource"
	"vanguard/resolver/recursor"
	"vanguard/resolver/stub"
)

const (
	ModuleAuth      = "auth"
	ModuleStubZone  = "stub_zone"
	ModuleForwarder = "forwarder"
	ModuleRecursor  = "recursor"
	ModuleLocalData = "local_data"
)

var ResolverSubmodule = []string{
	ModuleAuth,
	ModuleStubZone,
	ModuleForwarder,
	ModuleRecursor,
	ModuleLocalData,
}

type ResolverManager struct {
	core.DefaultHandler

	resolver chain.Resolver
	Auth     *auth.AuthDataSource
}

func NewResolver(conf *config.VanguardConf) core.DNSQueryHandler {
	querysource.NewQuerySourceManager(conf)

	var resolvers []chain.Resolver
	if isModuleEnable(conf, ModuleStubZone) {
		resolvers = append(resolvers, stub.NewStubZoneManager(conf))
	}
	if isModuleEnable(conf, ModuleForwarder) {
		resolvers = append(resolvers, forwarder.NewForwarder(conf))
	}
	if isModuleEnable(conf, ModuleRecursor) {
		resolvers = append(resolvers, recursor.NewRecursor(conf))
	}

	if len(resolvers) > 0 {
		chain.BuildResolverChain(resolvers...)
		resolvers = []chain.Resolver{NewQueryLimit(resolvers[0], conf)}
	}

	var authResolvers []chain.Resolver
	var authResolver *auth.AuthDataSource
	if isModuleEnable(conf, ModuleLocalData) {
		authResolvers = append(authResolvers, fakeauth.NewFakeAuth(conf))
	}
	if isModuleEnable(conf, ModuleAuth) {
		authResolver = auth.NewAuth(conf)
		authResolvers = append(authResolvers, authResolver)
	}
	resolvers = append(authResolvers, resolvers...)
	if len(resolvers) == 0 {
		panic("empty resolver module is specified")
	}

	chain.BuildResolverChain(resolvers...)
	cnameHandler := NewCNameHandler(resolvers[0], conf)
	return &ResolverManager{
		resolver: cnameHandler,
		Auth:     authResolver,
	}
}

func (mgr *ResolverManager) ReloadConfig(conf *config.VanguardConf) {
	querysource.ReloadConfig(conf)
	mgr.resolver.ReloadConfig(conf)
}

func (mgr *ResolverManager) HandleQuery(ctx *core.Context) {
	client := &ctx.Client
	mgr.resolver.Resolve(client)
	if client.Response == nil {
		core.PassToNext(mgr, ctx)
	}
}

func IsResolverSubModule(mod string) bool {
	for _, m := range ResolverSubmodule {
		if m == mod {
			return true
		}
	}
	return false
}

func isModuleEnable(conf *config.VanguardConf, mod string) bool {
	for _, em := range conf.EnableModules {
		if em == mod {
			return true
		}
	}
	return false
}
