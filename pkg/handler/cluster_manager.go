package handler

import (
	"fmt"
	"time"

	"cement/slice"
	"gorest"
	resterr "gorest/error"
	restresource "gorest/resource"
	"pkg/authentication"
	"pkg/authorization"
	"pkg/types"
	"pkg/zke"

	"cement/log"
)

const (
	ZCloudNamespace        = "zcloud"
	ZCloudAdmin            = "zcloud-cluster-admin"
	ZCloudReadonly         = "zcloud-cluster-readonly"
	zcloudEventBufferCount = 10
)

type ClusterManager struct {
	authorizer    *authorization.Authorizer
	authenticator *authentication.Authenticator
	zkeManager    *zke.ZKEManager
}

func newClusterManager(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer) (*ClusterManager, error) {
	clusterMgr := &ClusterManager{
		authorizer:    authorizer,
		authenticator: authenticator,
	}

	storageNodeListener := &StorageNodeListener{
		clusters: clusterMgr,
	}

	zkeMgr, err := zke.New(storageNodeListener)
	if err != nil {
		log.Errorf("create zke-manager failed %s", err.Error())
		return nil, err
	}

	clusterMgr.zkeManager = zkeMgr
	return clusterMgr, nil
}

func (m *ClusterManager) GetAuthorizer() *authorization.Authorizer {
	return m.authorizer
}

func (m *ClusterManager) GetClusterForSubResource(obj restresource.Resource) *zke.Cluster {
	ancestors := restresource.GetAncestors(obj)
	clusterID := ancestors[0].GetID()
	return m.zkeManager.GetReady(clusterID)
}

func (m *ClusterManager) GetClusterByName(name string) *zke.Cluster {
	return m.zkeManager.GetReady(name)
}

func (m *ClusterManager) Create(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can create cluster")
	}
	return m.zkeManager.Create(ctx)
}

func (m *ClusterManager) Update(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can update cluster")
	}
	return m.zkeManager.Update(ctx)
}

func (m *ClusterManager) Get(ctx *restresource.Context) (restresource.Resource, *resterr.APIError) {
	id := ctx.Resource.GetID()
	if m.authorizer.Authorize(getCurrentUser(ctx), id, "") == false {
		return nil, resterr.NewAPIError(resterr.Unauthorized, "user has no permission to access the cluster")
	}

	cluster := m.zkeManager.Get(id)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	sc := cluster.ToScCluster()
	if cluster.IsReady() {
		return getClusterInfo(cluster, sc), nil
	}
	return sc, nil
}

func getClusterInfo(zkeCluster *zke.Cluster, sc *types.Cluster) *types.Cluster {
	if !zkeCluster.IsReady() {
		return sc
	}

	version, err := zkeCluster.GetKubeClient().ServerVersion()
	if err != nil {
		return sc
	}
	sc.Version = version.GitVersion

	nodes, err := getNodes(zkeCluster.GetKubeClient())
	if err != nil {
		return sc
	}
	sc.NodesCount = len(nodes)
	for _, n := range nodes {
		if n.HasRole(types.RoleControlPlane) {
			continue
		}
		sc.Cpu += n.Cpu
		sc.CpuUsed += n.CpuUsed
		sc.Memory += n.Memory
		sc.MemoryUsed += n.MemoryUsed
		sc.Pod += n.Pod
		sc.PodUsed += n.PodUsed
	}
	if sc.Cpu > 0 {
		sc.CpuUsedRatio = fmt.Sprintf("%.2f", float64(sc.CpuUsed)/float64(sc.Cpu))
	}
	if sc.Memory > 0 {
		sc.MemoryUsedRatio = fmt.Sprintf("%.2f", float64(sc.MemoryUsed)/float64(sc.Memory))
	}
	if sc.Pod > 0 {
		sc.PodUsedRatio = fmt.Sprintf("%.2f", float64(sc.PodUsed)/float64(sc.Pod))
	}

	return sc
}

func (m *ClusterManager) List(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	requestFlags := ctx.Request.URL.Query()
	user := getCurrentUser(ctx)
	var readyClusters []*types.Cluster
	var allClusters []*types.Cluster

	for _, c := range m.zkeManager.List() {
		if m.authorizer.Authorize(user, c.Name, "") {
			sc := c.ToScCluster()
			allClusters = append(allClusters, sc)
			if c.IsReady() {
				readyClusters = append(readyClusters, sc)
			}
		}
	}

	if onlyReady := requestFlags.Get("onlyready"); onlyReady == "true" {
		return readyClusters, nil
	}
	return allClusters, nil
}

func (m *ClusterManager) Delete(ctx *restresource.Context) *resterr.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resterr.NewAPIError(resterr.PermissionDenied, "only admin can delete cluster")
	}
	id := ctx.Resource.GetID()
	return m.zkeManager.Delete(id)
}

func (m *ClusterManager) Action(ctx *restresource.Context) (interface{}, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can call cluster action apis")
	}

	action := ctx.Resource.GetAction()
	id := ctx.Resource.GetID()

	switch action.Name {
	case types.CSCancelAction:
		return m.zkeManager.CancelCluster(id)
	default:
		return nil, nil
	}
}

func (m *ClusterManager) authorizationHandler(enableDebug bool) gorest.HandlerFunc {
	return func(ctx *restresource.Context) *resterr.APIError {
		if _, ok := ctx.Resource.(*types.User); ok {
			action := ctx.Resource.GetAction()
			if action != nil && action.Name == types.ActionLogin {
				return nil
			}
		}

		user := getCurrentUser(ctx)
		if user == "" {
			return resterr.NewAPIError(resterr.Unauthorized, fmt.Sprintf("user is unknowned"))
		}

		if m.authorizer.GetUser(user) == nil {
			newUser := &types.User{Name: user}
			newUser.SetID(user)
			newUser.SetCreationTimestamp(time.Now())
			m.authorizer.AddUser(newUser)
		}

		ancestors := restresource.GetAncestors(ctx.Resource)
		if len(ancestors) < 2 {
			return nil
		}

		if _, ok := ancestors[0].(*types.Cluster); ok {
			if _, ok := ancestors[1].(*types.Namespace); ok {
				cluster := ancestors[0].GetID()
				namespace := ancestors[1].GetID()
				if !IsNamespaceVisiable(namespace, enableDebug) {
					return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("user %s has no sufficient permission to work on namespace %s", user, namespace))
				}

				if m.authorizer.Authorize(user, cluster, namespace) == false {
					return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("user %s has no sufficient permission to work on cluster %s namespace %s", user, cluster, namespace))
				}
			}
		}
		return nil
	}
}

type StorageNodeListener struct {
	clusters *ClusterManager
}

func (m StorageNodeListener) IsStorageNode(cluster *zke.Cluster, node string) (bool, error) {
	if cluster.GetKubeClient() == nil {
		return false, fmt.Errorf("cluster %s kubeClient is nil", cluster.Name)
	}
	storageClusters, err := getStorageClusters(cluster.GetKubeClient())
	if err != nil {
		return true, err
	}
	for _, storageCluster := range storageClusters.Items {
		if slice.SliceIndex(storageCluster.Spec.Hosts, node) >= 0 {
			return true, nil
		}
	}
	return false, nil
}

var _ zke.NodeListener = StorageNodeListener{}
