package handler

import (
	"fmt"

	resterror "gorest/error"
	"gorest/resource"
	ca "pkg/clusteragent"
	"pkg/types"
)

type OuterServiceManager struct {
	clusters *ClusterManager
}

func newOuterServiceManager(clusters *ClusterManager) *OuterServiceManager {
	return &OuterServiceManager{
		clusters: clusters,
	}
}

func (m *OuterServiceManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	var svcs []*types.OuterService
	if err := ca.GetAgent().ListResource(cluster.Name, genClusterAgentURL(ctx.Request.URL.Path, cluster.Name), &svcs); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list outerservices failed:%s", err.Error()))
	}
	return svcs, nil
}
