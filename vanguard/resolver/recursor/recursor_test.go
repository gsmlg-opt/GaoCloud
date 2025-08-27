package recursor

import (
	"bufio"
	//"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"testing"

	ut "cement/unittest"
	"g53"
	"vanguard/config"
	"vanguard/core"
	"vanguard/logger"
	"vanguard/resolver/querysource"
	view "vanguard/viewselector"
)

func TestRecursive(t *testing.T) {
	logger.UseDefaultLogger("error")
	recurConf := config.RecursorInView{
		View:   "default",
		Enable: true,
	}
	conf := &config.VanguardConf{}
	conf.Recursor = []config.RecursorInView{recurConf}
	view.NewSelectorMgr(conf)
	querysource.NewQuerySourceManager(conf)

	f, err := os.Open("top_china_domains.txt")
	ut.Assert(t, err == nil, "open top 500 domain names failed")
	defer f.Close()

	allName, _ := ioutil.ReadAll(bufio.NewReader(f))
	namesToQuery := strings.Split(string(allName), "\n")
	r := NewRecursor(conf)
	r.ReloadConfig(conf)

	failedNames := []string{}
	var failedNameLock sync.Mutex
	var wg sync.WaitGroup

	nameChan := make(chan *g53.Name)
	for i := 0; i < 5; i++ {
		go func() {
			wg.Add(1)
			for {
				if name, ok := <-nameChan; ok == false {
					wg.Done()
					return
				} else {
					var client core.Client
					client.Request = g53.MakeQuery(name, g53.RR_A, 1024, false)
					client.Addr, _ = net.ResolveUDPAddr("udp", "127.0.0.1:0")
					client.View = "default"
					r.Resolve(&client)
					if client.Response == nil {
						failedNameLock.Lock()
						failedNames = append(failedNames, name.String(false))
						failedNameLock.Unlock()
					} else {
						//fmt.Printf("get:%s\n", client.Response.String())
					}
				}
			}
		}()
	}

	for _, name := range namesToQuery {
		qname, err := g53.NewName(strings.Trim(name, " "), false)
		if err != nil {
			continue
		}
		nameChan <- qname
	}
	close(nameChan)
	wg.Wait()

	ut.Assert(t, len(failedNames) == 0, "failed names is %v", failedNames)
}
