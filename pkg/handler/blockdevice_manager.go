package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resterr "github.com/zdnscloud/gorest/error"
	resource "github.com/zdnscloud/gorest/resource"
	"github.com/gsmlg-opt/gaocloud/pkg/clusteragent"
	"github.com/gsmlg-opt/gaocloud/pkg/types"
)

type BlockDeviceManager struct {
	clusters *ClusterManager
}

func newBlockDeviceManager(clusters *ClusterManager) *BlockDeviceManager {
	return &BlockDeviceManager{
		clusters: clusters,
	}
}

func (m *BlockDeviceManager) List(ctx *resource.Context) (interface{}, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	resp, err := getBlockDevices(cluster.Name, cluster.GetKubeClient(), clusteragent.GetAgent())
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get blockdevices failed %s", err.Error()))
	}
	return resp, nil
}

func getBlockDevices(cluster string, cli client.Client, agent *clusteragent.AgentManager) ([]*types.BlockDevice, error) {
	all, err := getAllDevices(cluster, agent)
	if err != nil {
		return nil, err
	}
	return cutInvalid(cli, all), nil
}

func getAllDevices(cluster string, agent *clusteragent.AgentManager) ([]types.ClusterAgentBlockDevice, error) {
	res := make([]types.ClusterAgentBlockDevice, 0)
	if err := agent.ListResource(cluster, "/blockdevices", &res); err != nil {
		return nil, err
	}
	return res, nil
}

func cutInvalid(cli client.Client, resp []types.ClusterAgentBlockDevice) []*types.BlockDevice {
	res := make([]*types.BlockDevice, 0)
	infos := getStorageUsed(cli)
	for _, h := range resp {
		if !isValidHost(cli, h.NodeName) || len(h.BlockDevices) == 0 {
			continue
		}
		var usedby string
		devs := make([]types.Dev, 0)
		for _, d := range h.BlockDevices {
			used := getUsed(h.NodeName, d, infos)
			if used == "other" {
				continue
			}
			if used != "" {
				usedby = used
			}
			dev := types.Dev{
				Name: d.Name,
				Size: d.Size,
			}
			devs = append(devs, dev)
		}
		if len(devs) == 0 {
			continue
		}
		host := &types.BlockDevice{
			NodeName:     h.NodeName,
			BlockDevices: devs,
			UsedBy:       usedby,
		}
		host.SetID(h.NodeName)
		res = append(res, host)
	}
	return res
}

func isValidHost(cli client.Client, name string) bool {
	node := corev1.Node{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &node); err != nil {
		return false
	}
	_, ok1 := node.Labels[zkeRoleLabelPrefix+string(types.RoleControlPlane)]
	_, ok2 := node.Labels[zkeRoleLabelPrefix+string(types.RoleEtcd)]
	if ok1 || ok2 {
		return false
	}
	return true
}

func getUsed(host string, dev types.ClusterAgentDev, infos map[string][]string) string {
	var used string
	info, ok := infos[host]
	if !ok {
		if dev.Parted || dev.Filesystem || dev.Mount {
			return "other"
		}
		return used
	}
	used = "other"
	for _, d := range info {
		if dev.Name != d {
			continue
		}
		used = info[0]
	}
	return used
}

func getStorageUsed(cli client.Client) map[string][]string {
	infos := make(map[string][]string)
	scs, _ := getStorageClusters(cli)
	for _, sc := range scs.Items {
		for _, info := range sc.Status.Config {
			devs := make([]string, 0)
			devs = append(devs, sc.Name)
			for _, d := range info.BlockDevices {
				devs = append(devs, d)
			}
			infos[info.NodeName] = devs
		}
	}
	return infos
}
