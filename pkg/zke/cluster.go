package zke

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gsmlg-opt/gaocloud/pkg/types"
	"github.com/gsmlg-opt/gaocloud/pkg/zke/zkelog"

	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	appv1beta1 "github.com/zdnscloud/application-operator/pkg/apis/app/v1beta1"
	"github.com/zdnscloud/cement/fsm"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/kvzoo"
	zketypes "github.com/zdnscloud/zke/types"
	"k8s.io/client-go/rest"
)

const (
	connectionCheckInterval = time.Second * 15
)

type Cluster struct {
	Name           string
	createTime     time.Time
	deleteTime     time.Time
	kubeClient     client.Client
	cache          cache.Cache
	k8sConfig      *rest.Config
	stopCh         chan struct{}
	config         *zketypes.ZKEConfig
	cancel         context.CancelFunc
	isCanceled     bool
	fsm            *fsm.FSM
	scVersion      string
	kubeHttpClient *http.Client
}

func (c *Cluster) GetCreationTimestamp() time.Time {
	return c.createTime
}

func (c *Cluster) GetDeletionTimestamp() time.Time {
	return c.deleteTime
}

func (c *Cluster) GetKubeClient() client.Client {
	return c.kubeClient
}

func (c *Cluster) GetKubeCache() cache.Cache {
	return c.cache
}

func (c *Cluster) GetKubeRestConfig() *rest.Config {
	return c.k8sConfig
}

func (c *Cluster) GetKubeHttpClient() *http.Client {
	return c.kubeHttpClient
}

type AlarmCluster struct {
	Cluster string
	Reason  string
	Message string
}

func newCluster(name string, initialStatus types.ClusterStatus) *Cluster {
	cluster := &Cluster{
		Name: name,
	}

	fsm := newClusterFsm(cluster, initialStatus)
	cluster.fsm = fsm
	return cluster
}

func (c *Cluster) IsReady() bool {
	status := c.getStatus()
	if status == types.CSRunning || status == types.CSUnreachable {
		return true
	}
	return false
}

func (c *Cluster) event(e string, zkeMgr *ZKEManager, state clusterState, errMessage string) {
	if err := c.fsm.Event(e, zkeMgr, state, errMessage); err != nil {
		log.Warnf("send cluster %s fsm %s event failed %s", c.Name, e, err.Error())
	}
}

func (c *Cluster) Event(e string) error {
	return c.fsm.Event(e)
}

func (c *Cluster) GetNodeIpsByRole(role types.NodeRole) []string {
	ips := []string{}
	cluster := c.ToScCluster()
	for _, n := range cluster.Nodes {
		if n.HasRole(role) {
			ips = append(ips, n.Address)
		}
	}
	return ips
}

func (c *Cluster) Cancel() error {
	status := c.getStatus()
	if status == types.CSCreating || status == types.CSUpdating {
		if c.isCanceled {
			return fmt.Errorf("cluster %s is caceling", c.Name)
		}
		c.isCanceled = true
		c.cancel()
		return nil
	}
	return fmt.Errorf("can't cancel cluster %s on %s status", c.Name, status)
}

func (c *Cluster) Can(operate string) bool {
	return c.fsm.Can(operate)
}

func (c *Cluster) getStatus() types.ClusterStatus {
	return types.ClusterStatus(c.fsm.Current())
}

func (c *Cluster) Init(kubeConfig string) error {
	k8sConfig, err := config.BuildConfig([]byte(kubeConfig))
	if err != nil {
		return fmt.Errorf("build cluster %s k8sconfig failed %s", c.Name, err.Error())
	}

	var options client.Options
	options.Scheme = client.GetDefaultScheme()
	storagev1.AddToScheme(options.Scheme)
	tektonv1alpha1.AddToScheme(options.Scheme)
	appv1beta1.AddToScheme(options.Scheme)

	kubeClient, err := client.New(k8sConfig, options)
	if err != nil {
		return fmt.Errorf("New cluster %s gok8s client failed %s", c.Name, err.Error())
	}
	c.kubeClient = kubeClient
	if err := c.setCache(k8sConfig); err != nil {
		return fmt.Errorf("set cluster %s cache failed %s", c.Name, err.Error())
	}
	go c.connectionCheckLoop()
	return nil
}

func (c *Cluster) connectionCheckLoop() {
	for {
		select {
		case <-c.stopCh:
			log.Debugf("cluster %s connectionCheckLoop exit", c.Name)
			return
		case <-time.After(connectionCheckInterval):
			if _, err := c.GetKubeClient().ServerVersion(); err != nil {
				c.Event(GetInfoFailedEvent)
			} else {
				c.Event(GetInfoSucceedEvent)
			}
		}
	}
}

func (c *Cluster) setCache(k8sConfig *rest.Config) error {
	httpClient, err := c.newKubeHttpClient(k8sConfig)
	if err != nil {
		return err
	}
	c.kubeHttpClient = httpClient
	c.stopCh = make(chan struct{})
	c.k8sConfig = k8sConfig
	cache, err := cache.New(k8sConfig, cache.Options{})
	if err != nil {
		return err
	}
	go cache.Start(c.stopCh)
	cache.WaitForCacheSync(c.stopCh)
	c.cache = cache
	return nil
}

func (c *Cluster) newKubeHttpClient(k8sConfig *rest.Config) (*http.Client, error) {
	secureTransport, err := rest.TransportFor(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("create kube http client failed: %s", err.Error())
	}

	return &http.Client{
		Transport: secureTransport,
	}, nil
}

func (c *Cluster) Create(ctx context.Context, state clusterState, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			err := log.Errorf("zke pannic info %s", r)
			c.event(CreateFailedEvent, mgr, state, err.Error())
		}
	}()

	logger, logCh := log.NewISO3339Log4jBufLogger(zkelog.MaxLogSize, log.Info)
	defer logger.Close()
	mgr.logger.AddOrUpdate(c.Name, logCh)
	zkeState, k8sConfig, kubeClient, err := upZKECluster(ctx, c.config, state.FullState, logger)
	state.FullState = zkeState
	if c.isCanceled {
		c.event(CreateCanceledEvent, mgr, state, "")
		return
	}
	if err != nil {
		log.Errorf("zke err info %s", err)
		logger.Error(err.Error())
		c.event(CreateFailedEvent, mgr, state, err.Error())
		return
	}

	c.kubeClient = kubeClient
	if err := c.setCache(k8sConfig); err != nil {
		c.event(CreateFailedEvent, mgr, state, err.Error())
		return
	}
	go c.connectionCheckLoop()
	state.Created = true
	c.event(CreateSucceedEvent, mgr, state, "")
}

func (c *Cluster) Update(ctx context.Context, state clusterState, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			err := log.Errorf("zke pannic info %s", r)
			c.event(UpdateCompletedEvent, mgr, state, err.Error())
		}
	}()

	logger, logCh := log.NewISO3339Log4jBufLogger(zkelog.MaxLogSize, log.Info)
	defer logger.Close()
	mgr.logger.AddOrUpdate(c.Name, logCh)

	zkeState, k8sConfig, k8sClient, err := upZKECluster(ctx, c.config, state.FullState, logger)
	state.FullState = zkeState
	if c.isCanceled {
		if state.Created {
			c.event(UpdateCanceledEvent, mgr, state, "")
		} else {
			c.event(CreateCanceledEvent, mgr, state, "")
		}
		return
	}
	if err != nil {
		log.Errorf("zke err info %s", err)
		logger.Error(err.Error())
		if state.Created {
			c.event(UpdateCompletedEvent, mgr, state, err.Error())
		} else {
			c.event(CreateFailedEvent, mgr, state, err.Error())
		}
		return
	}

	if state.Created {
		c.event(UpdateCompletedEvent, mgr, state, "")
	} else {
		c.kubeClient = k8sClient
		if err := c.setCache(k8sConfig); err != nil {
			c.event(CreateFailedEvent, mgr, state, err.Error())
		}
		state.Created = true
		c.event(CreateSucceedEvent, mgr, state, "")
	}
}

func (c *Cluster) Destroy(ctx context.Context, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			err := log.Errorf("zke pannic info %s", r)
			c.event(DeleteCompletedEvent, mgr, clusterState{}, err.Error())
		}
	}()

	logger, _ := log.NewISO3339Log4jBufLogger(zkelog.MaxLogSize, log.Info)
	defer logger.Close()

	err := removeZKECluster(ctx, c.config, logger)
	if err != nil {
		log.Errorf("zke err info %s", err)
		logger.Error(err.Error())
		c.event(DeleteCompletedEvent, mgr, clusterState{}, err.Error())
		return
	}
	c.event(DeleteCompletedEvent, mgr, clusterState{}, "")
}

func (c *Cluster) ToScCluster() *types.Cluster {
	sc := &types.Cluster{
		Name:                c.Name,
		SSHUser:             c.config.Option.SSHUser,
		SSHPort:             c.config.Option.SSHPort,
		ClusterCidr:         c.config.Option.ClusterCidr,
		ServiceCidr:         c.config.Option.ServiceCidr,
		ClusterDomain:       c.config.Option.ClusterDomain,
		ClusterDNSServiceIP: c.config.Option.ClusterDNSServiceIP,
		ClusterUpstreamDNS:  c.config.Option.ClusterUpstreamDNS,
		SingleCloudAddress:  c.config.SingleCloudAddress,
		ScVersion:           c.scVersion,

		Network: types.ClusterNetwork{
			Plugin: c.config.Network.Plugin,
			Iface:  c.config.Network.Iface,
		},
	}

	for _, node := range c.config.Nodes {
		n := types.Node{
			Name:    node.NodeName,
			Address: node.Address,
		}
		for _, role := range node.Role {
			if role == string(types.RoleEtcd) {
				continue
			}
			n.Roles = append(n.Roles, types.NodeRole(role))
		}
		sc.Nodes = append(sc.Nodes, n)
	}
	sc.NodesCount = len(sc.Nodes)

	sc.SetID(c.Name)
	sc.SetCreationTimestamp(c.createTime)
	sc.SetDeletionTimestamp(c.deleteTime)
	sc.Status = c.getStatus()
	sc.KubeProvider = c
	return sc
}

func (c *Cluster) GetKubeConfig(user string, table kvzoo.Table) (string, error) {
	state, err := getClusterFromDB(c.Name, table)
	if err != nil {
		return "", err
	}
	if state.FullState.CurrentState.CertificatesBundle != nil {
		kubeConfigCert, ok := state.CurrentState.CertificatesBundle[user]
		if !ok {
			return "", fmt.Errorf("cluster %s user %s cert doesn't exist", c.Name, user)
		}
		return kubeConfigCert.Config, nil
	}
	return "", nil
}
