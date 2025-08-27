package handler

import (
	"fmt"

	resterror "gorest/error"
	"gorest/resource"
	ca "pkg/clusteragent"
	"pkg/types"
)

type ServiceNetworkManager struct {
	clusters *ClusterManager
}

func newServiceNetworkManager(clusters *ClusterManager) *ServiceNetworkManager {
	return &ServiceNetworkManager{
		clusters: clusters,
	}
}

func (m *ServiceNetworkManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	var networks []*types.ServiceNetwork
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &networks); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list servicenetworks failed:%s", err.Error()))
	}
	return networks, nil
}
