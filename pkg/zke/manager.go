package zke

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"cement/log"
	resterr "gorest/error"
	restsource "gorest/resource"
	"kvzoo"
	"zke/core"
	"zke/core/pki"

	"pkg/db"
	"pkg/eventbus"
	"pkg/types"
	"pkg/zke/zkelog"
)

const (
	alarmEventBufferCount = 10
)

var gaoCloudVersion string

type ZKEManager struct {
	clusters     []*Cluster
	dbTable      kvzoo.Table
	lock         sync.Mutex
	scVersion    string       // add cluster gaocloud version for easy to confirm zcloud component version
	nodeListener NodeListener // for check storage node
	logger       *zkelog.LogManager
}

type NodeListener interface {
	IsStorageNode(cluster *Cluster, node string) (bool, error)
}

func New(nl NodeListener) (*ZKEManager, error) {
	return newZKEManager(db.GetGlobalDB(), nl)
}

func newZKEManager(db kvzoo.DB, nl NodeListener) (*ZKEManager, error) {
	tn, _ := kvzoo.TableNameFromSegments(ZKEManagerDBTable)
	table, err := db.CreateOrGetTable(tn)
	if err != nil {
		return nil, fmt.Errorf("create or get db table failed %s", err.Error())
	}

	mgr := &ZKEManager{
		clusters:     make([]*Cluster, 0),
		dbTable:      table,
		scVersion:    gaoCloudVersion,
		nodeListener: nl,
		logger:       zkelog.New(),
	}

	if err := mgr.loadDB(); err != nil {
		return mgr, err
	}
	return mgr, nil
}

func (m *ZKEManager) OpenLog(cluster string, r *http.Request, w http.ResponseWriter) {
	m.logger.OpenLog(cluster, r, w)
}

func (m *ZKEManager) Create(ctx *restsource.Context) (restsource.Resource, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()

	typesCluster := ctx.Resource.(*types.Cluster)
	typesCluster.TrimFieldSpace()

	existCluster := m.get(typesCluster.Name)
	if existCluster != nil {
		return nil, resterr.NewAPIError(resterr.DuplicateResource, "duplicate cluster")
	}

	if err := validateConfigForCreate(typesCluster); err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidOption, fmt.Sprintf("cluster config validate failed %s", err))
	}

	config := genZKEConfig(typesCluster)
	state := clusterState{
		ZKEConfig:  config,
		CreateTime: time.Now(),
		FullState:  &core.FullState{},
		Created:    false,
		ScVersion:  m.scVersion,
	}
	if err := createOrUpdateClusterFromDB(typesCluster.Name, state, m.dbTable); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	cluster := newCluster(typesCluster.Name, types.CSCreating)
	cluster.createTime = state.CreateTime
	cluster.config = config
	cluster.scVersion = m.scVersion
	m.add(cluster)

	cancelCtx, cancel := context.WithCancel(context.Background())
	cluster.cancel = cancel
	go cluster.Create(cancelCtx, state, m)
	typesCluster.SetID(typesCluster.Name)
	typesCluster.SetCreationTimestamp(state.CreateTime)
	return typesCluster, nil
}

func (m *ZKEManager) Update(ctx *restsource.Context) (restsource.Resource, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()

	typesCluster := ctx.Resource.(*types.Cluster)
	typesCluster.TrimFieldSpace()

	existCluster := m.get(typesCluster.Name)
	if existCluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", typesCluster.Name))
	}

	if err := validateConfigForUpdate(existCluster.ToScCluster(), typesCluster, m.nodeListener, existCluster); err != nil {
		return nil, resterr.NewAPIError(resterr.InvalidOption, fmt.Sprintf("cluster config validate failed %s", err))
	}
	config := genZKEConfigForUpdate(existCluster.config, typesCluster)

	state, err := getClusterFromDB(typesCluster.Name, m.dbTable)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	if state.Created && !existCluster.Can(UpdateEvent) {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("cluster %s can't update on %s status", existCluster.Name, existCluster.getStatus()))
	}
	state.ZKEConfig = config
	existCluster.config = config

	if err := createOrUpdateClusterFromDB(typesCluster.Name, state, m.dbTable); err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	if state.Created {
		if err := existCluster.Event(UpdateEvent); err != nil {
			return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("send cluster %s fsm %s event failed %s", existCluster.Name, UpdateEvent, err.Error()))
		}
	} else {
		if err := existCluster.Event(ContinuteCreateEvent); err != nil {
			return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("send cluster %s fsm %s event failed %s", existCluster.Name, ContinuteCreateEvent, err.Error()))
		}
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	existCluster.cancel = cancel
	go existCluster.Update(cancelCtx, state, m)
	return typesCluster, nil
}

func (m *ZKEManager) Get(id string) *Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()

	cluster := m.get(id)
	if cluster != nil {
		return cluster
	}
	return nil
}

func (m *ZKEManager) GetReady(id string) *Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()

	cluster := m.get(id)
	if cluster != nil && cluster.IsReady() {
		return cluster
	}
	return nil
}

func (m *ZKEManager) get(id string) *Cluster {
	for _, c := range m.clusters {
		if c.Name == id {
			return c
		}
	}
	return nil
}

func (m *ZKEManager) List() []*Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.clusters
}

func (m *ZKEManager) ListReady() []*Cluster {
	m.lock.Lock()
	defer m.lock.Unlock()

	clusters := []*Cluster{}
	for _, c := range m.clusters {
		if c.IsReady() {
			clusters = append(clusters, c)
		}
	}
	return clusters
}

func (m *ZKEManager) Delete(id string) *resterr.APIError {
	m.lock.Lock()
	defer m.lock.Unlock()

	toDelete := m.get(id)
	if toDelete == nil {
		return resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}

	if !toDelete.Can(DeleteEvent) {
		return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("cluster %s can't delete when on %s status", id, toDelete.getStatus()))
	}

	state, err := getClusterFromDB(toDelete.Name, m.dbTable)
	if err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}

	if toDelete.Event(DeleteEvent); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("send cluster %s fsm %s event failed %s", toDelete.Name, DeleteEvent, err.Error()))
	}

	if state.Created {
		close(toDelete.stopCh)
		eventbus.PublishResourceDeleteEvent(toDelete.ToScCluster())
	}

	tm := time.Now()
	toDelete.deleteTime = tm
	state.DeleteTime = tm
	if err := createOrUpdateClusterFromDB(id, state, m.dbTable); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("%s", err))
	}
	go toDelete.Destroy(context.TODO(), m)
	return nil
}

func (m *ZKEManager) CancelCluster(id string) (interface{}, *resterr.APIError) {
	m.lock.Lock()
	defer m.lock.Unlock()

	c := m.get(id)
	if c == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("cluster %s desn't exist", id))
	}
	if err := c.Cancel(); err != nil {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, err.Error())
	}
	return nil, nil
}

func (m *ZKEManager) loadDB() error {
	states, err := getClustersFromDB(m.dbTable)
	if err != nil {
		return err
	}

	for k, v := range states {
		if v.Created {
			cluster := newCluster(k, types.CSRunning)
			cluster.config = v.ZKEConfig
			cluster.createTime = v.CreateTime
			cluster.scVersion = v.ScVersion
			if err := cluster.Init(v.CurrentState.CertificatesBundle[pki.KubeAdminCertName].Config); err != nil {
				log.Warnf("init cluster %s failed %s", k, err.Error())
				continue
			}
			m.add(cluster)
			eventbus.PublishResourceCreateEvent(cluster.ToScCluster())
		} else {
			cluster := newCluster(k, types.CSCreateFailed)
			cluster.config = v.ZKEConfig
			cluster.createTime = v.CreateTime
			cluster.scVersion = v.ScVersion
			m.add(cluster)
		}
	}
	return nil
}

func (m *ZKEManager) add(c *Cluster) {
	m.clusters = append(m.clusters, c)
}

func (m *ZKEManager) GetDBTable() kvzoo.Table {
	return m.dbTable
}

func (m *ZKEManager) Remove(cluster *Cluster) {
	m.lock.Lock()
	defer m.lock.Unlock()

	for i, c := range m.clusters {
		if c.Name == cluster.Name {
			m.clusters = append(m.clusters[:i], m.clusters[i+1:]...)
			break
		}
	}
}
