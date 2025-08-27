package main

import (
	"flag"
	"fmt"
	"net"

	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
	"vanguard/resolver"
	"vanguard/resolver/querysource"
	"vanguard/resolver/recursor"
	view "vanguard/viewselector"
)

var (
	name string
	typ  string
)

func init() {
	flag.StringVar(&name, "n", "www.zdns.cn.", "query name")
	flag.StringVar(&typ, "t", "a", "query type")
}

func main() {
	flag.Parse()

	logger.UseDefaultLogger("debug")
	conf := &config.VanguardConf{}
	conf.Recursor = []config.RecursorInView{
		config.RecursorInView{
			Enable: true,
			View:   "default",
		},
	}
	view.NewSelectorMgr(conf)
	querysource.NewQuerySourceManager(conf)

	r := resolver.NewCNameHandler(recursor.NewRecursor(conf), conf)
	r.ReloadConfig(conf)

	qname, err := g53.NameFromString(name)
	if err != nil {
		fmt.Printf("name isn't valid")
		return
	}

	qtype, err := g53.TypeFromString(typ)
	if err != nil {
		fmt.Printf("qtype isn't valid")
		return
	}

	var client core.Client
	client.Request = g53.MakeQuery(qname, qtype, 1024, false)
	client.Addr, _ = net.ResolveUDPAddr("udp", "127.0.0.1:0")
	client.View = "default"
	r.Resolve(&client)
	if client.Response != nil {
		fmt.Printf("%s\n", client.Response.String())
	}
}
