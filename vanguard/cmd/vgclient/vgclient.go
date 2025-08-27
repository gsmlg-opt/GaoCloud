package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"vanguard/cache"
	"vanguard/httpcmd"
	"vanguard/resolver/auth"
	"vanguard/resolver/forwarder"
	"vanguard/server"
)

const (
	cmdReconfig        = "reconfig"
	cmdCleanCache      = "clean_cache"
	cmdCleanViewCache  = "clean_view_cache"
	cmdCleanRRsetCache = "clean_rrset_cache"
	cmdAddRr           = "add_rr"
	cmdDeleteRr        = "delete_rr"
	cmdAddForwarder    = "add_forwarder"
	cmdGetDomainCache  = "get_domain_cache"
	cmdGetMessageCache = "get_message_cache"
)

const cmdServiceName = "vanguard_cmd"
const cmdServiceIP = "127.0.0.1"
const cmdServicePort = 9000

var cmdServer string

func init() {
	flag.StringVar(&cmdServer, "s", "127.0.0.1:9009", "command server addr")
}

var supportedCommands = []httpcmd.Command{
	&server.Reconfig{},
	&cache.CleanCache{},
	&cache.CleanViewCache{},
	&cache.CleanDomainCache{},
	&cache.CleanRRsetsCache{},
	&cache.GetDomainCache{},
	&cache.GetMessageCache{},
	&auth.AddAuthRrs{},
	&auth.DeleteAuthRrs{},

	&forwarder.AddForwardZone{},
}

func main() {
	flag.Parse()
	args := flag.Args()
	task := httpcmd.NewTask()
	switch args[0] {
	case cmdReconfig:
		task.AddCmd(&server.Reconfig{})
	case cmdCleanCache:
		task.AddCmd(&cache.CleanCache{})
	case cmdCleanViewCache:
		task.AddCmd(&cache.CleanViewCache{View: args[1]})
	case cmdCleanRRsetCache:
		task.AddCmd(&cache.CleanRRsetsCache{View: args[1], Name: args[2]})
	case cmdAddRr:
		authRr := &auth.AuthRR{
			View:  args[1],
			Zone:  args[2],
			Name:  args[3],
			Ttl:   args[4],
			Type:  args[5],
			Rdata: args[6]}
		task.AddCmd(&auth.AddAuthRrs{
			auth.AuthRRs{authRr},
		})
	case cmdDeleteRr:
		authRr := &auth.AuthRR{
			View:  args[1],
			Zone:  args[2],
			Name:  args[3],
			Ttl:   args[4],
			Type:  args[5],
			Rdata: args[6]}
		task.AddCmd(&auth.DeleteAuthRrs{
			auth.AuthRRs{authRr},
		})
	case cmdAddForwarder:
		addForwarder := &forwarder.AddForwardZone{
			Zones: []forwarder.ForwardZoneParam{
				forwarder.ForwardZoneParam{
					View:         args[1],
					Name:         args[2],
					Forwarders:   strings.Split(args[3], ","),
					ForwardStyle: "order",
				},
			},
		}
		task.AddCmd(addForwarder)
	case cmdGetDomainCache:
		getDomainCache := &cache.GetDomainCache{
			Name: args[1],
			Type: args[2],
		}
		task.AddCmd(getDomainCache)
	case cmdGetMessageCache:
		getMessageCache := &cache.GetMessageCache{
			View: args[1],
			Name: args[2],
			Type: args[3],
		}
		task.AddCmd(getMessageCache)
	default:
		fmt.Printf("unknown cmd %v\n", args[0])
		return
	}

	ipAndPort := strings.Split(cmdServer, ":")
	port, err := strconv.Atoi(ipAndPort[1])
	if err != nil || port == 0 {
		fmt.Printf("cmd server port isn't valid\n")
		return
	}

	e := &httpcmd.EndPoint{
		Name: cmdServiceName,
		IP:   ipAndPort[0],
		Port: port,
	}

	proxy, err := httpcmd.GetProxy(e, supportedCommands)
	if err != nil {
		fmt.Printf("failed to get proxy:" + err.Error())
		return
	}

	if args[0] == cmdGetDomainCache || args[0] == cmdGetMessageCache {
		var rrsets []cache.RRInCache
		err = proxy.HandleTask(task, &rrsets)
		if err.(*httpcmd.Error) == nil {
			for _, rrset := range rrsets {
				fmt.Printf("%v\n", rrset)
			}
		}
	} else {
		err = proxy.HandleTask(task, nil)
	}

	if err.(*httpcmd.Error) != nil {
		fmt.Printf("failed to handle task:" + err.Error())
		return
	}

	task.ClearCmd()
	proxy.Close()
}
