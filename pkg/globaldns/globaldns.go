package globaldns

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"cement/log"
	"g53"
	"gok8s/cache"

	eb "pkg/eventbus"
	"pkg/types"
)

const (
	KubeSystemNamespace = "kube-system"
	ClusterConfig       = "cluster-config"
)

var (
	GetClusterConfigOption = k8stypes.NamespacedName{KubeSystemNamespace, ClusterConfig}
)

type GlobalDNS struct {
	clusterEventCh    <-chan interface{}
	clusterDNSSyncers map[string]*ClusterDNSSyncer
	proxy             *DnsProxy
	lock              sync.Mutex
}

func New(httpCmdAddr string) error {
	if httpCmdAddr == "" {
		return nil
	}

	proxy, err := newDnsProxy(httpCmdAddr)
	if err != nil {
		return err
	}

	gdns := &GlobalDNS{
		clusterEventCh:    eb.SubscribeResourceEvent(types.Cluster{}),
		clusterDNSSyncers: make(map[string]*ClusterDNSSyncer),
		proxy:             proxy,
	}

	go gdns.eventLoop()
	return nil
}

func (g *GlobalDNS) eventLoop() {
	for {
		event := <-g.clusterEventCh
		switch e := event.(type) {
		case eb.ResourceCreateEvent:
			cluster := e.Resource.(*types.Cluster)
			g.lock.Lock()
			err := g.newClusterDNSSyncer(cluster.Name, cluster.KubeProvider.GetKubeCache())
			if err != nil {
				log.Warnf("create globaldns syncer for cluster %s failed: %s", cluster.Name, err.Error())
			}
			g.lock.Unlock()
		case eb.ResourceDeleteEvent:
			clusterName := e.Resource.GetID()
			g.lock.Lock()
			syncer, ok := g.clusterDNSSyncers[clusterName]
			if ok {
				syncer.Stop()
				delete(g.clusterDNSSyncers, clusterName)
			} else {
				log.Warnf("globaldns syncer is unknown cluster %s", clusterName)
			}
			g.lock.Unlock()
		}
	}
}

func (g *GlobalDNS) newClusterDNSSyncer(clusterName string, c cache.Cache) error {
	if _, ok := g.clusterDNSSyncers[clusterName]; ok {
		return fmt.Errorf("duplicate cluster name %s", clusterName)
	}

	k8sconfigmap := &corev1.ConfigMap{}
	if err := c.Get(context.TODO(), GetClusterConfigOption, k8sconfigmap); err != nil {
		return fmt.Errorf("get full-cluster-state configmap failed: %s", err.Error())
	}

	var zkeConfig ZKEConfig
	if err := json.Unmarshal([]byte(k8sconfigmap.Data[ClusterConfig]), &zkeConfig); err != nil {
		return fmt.Errorf("unmarshal full-cluster-state configmap failed: %s", err.Error())
	}

	clusterDomain := zkeConfig.Option.ClusterDomain
	if clusterDomain == "" {
		return fmt.Errorf("cluster %s zone should not be empty", clusterName)
	}

	zoneName, err := g53.NameFromString(clusterDomain)
	if err != nil {
		return fmt.Errorf("parse cluster %s zone name %s failed: %v", clusterName, clusterDomain, err.Error())
	}

	for cluster, syncer := range g.clusterDNSSyncers {
		if syncer.GetZoneName().Equals(zoneName) {
			return fmt.Errorf("duplicate cluster zone %v, the zone has been belongs to cluster %v", clusterDomain, cluster)
		}
	}

	syncer, err := newClusterDNSSyncer(zoneName, c, g.proxy)
	if err != nil {
		return err
	}

	g.clusterDNSSyncers[clusterName] = syncer
	return nil
}
