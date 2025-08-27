package handler

import (
	"fmt"

	resterror "gorest/error"
	"gorest/resource"
	ca "pkg/clusteragent"
	"pkg/types"
)

type PodNetworkManager struct {
	clusters *ClusterManager
}

func newPodNetworkManager(clusters *ClusterManager) *PodNetworkManager {
	return &PodNetworkManager{
		clusters: clusters,
	}
}

func (m *PodNetworkManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	var networks []*types.PodNetwork
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &networks); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list podnetworks failed:%s", err.Error()))
	}
	return networks, nil
}
