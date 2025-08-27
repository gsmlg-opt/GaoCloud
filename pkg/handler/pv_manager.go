package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/gsmlg-opt/gaocloud/pkg/types"
)

type PersistentVolumeManager struct {
	clusters *ClusterManager
}

func newPersistentVolumeManager(clusters *ClusterManager) *PersistentVolumeManager {
	return &PersistentVolumeManager{clusters: clusters}
}

func (m *PersistentVolumeManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	k8sPersistentVolumes, err := getPersistentVolumes(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found pvs")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list pvs failed %s", err.Error()))
	}

	var pvs []*types.PersistentVolume
	for _, item := range k8sPersistentVolumes.Items {
		pvs = append(pvs, k8sPVToSCPV(&item))
	}
	return pvs, nil
}

func (m PersistentVolumeManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	pv := ctx.Resource.(*types.PersistentVolume)
	k8sPersistentVolume, err := getPersistentVolume(cluster.GetKubeClient(), pv.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found pv %s", pv.GetID()))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get pv %s failed:%s", pv.GetID(), err.Error()))
	}

	return k8sPVToSCPV(k8sPersistentVolume), nil
}

func (m PersistentVolumeManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	pv := ctx.Resource.(*types.PersistentVolume)
	err := deletePersistentVolume(cluster.GetKubeClient(), pv.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found pv %s", pv.GetID()))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete pv %s failed:%s", pv.GetID(), err.Error()))
	}
	return nil
}

func getPersistentVolume(cli client.Client, name string) (*corev1.PersistentVolume, error) {
	pv := corev1.PersistentVolume{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &pv)
	return &pv, err
}

func getPersistentVolumes(cli client.Client) (*corev1.PersistentVolumeList, error) {
	pvs := corev1.PersistentVolumeList{}
	err := cli.List(context.TODO(), nil, &pvs)
	return &pvs, err
}

func deletePersistentVolume(cli client.Client, name string) error {
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Delete(context.TODO(), pv)
}

func k8sPVToSCPV(k8sPersistentVolume *corev1.PersistentVolume) *types.PersistentVolume {
	var ref types.ClaimRef
	if claim := k8sPersistentVolume.Spec.ClaimRef; claim != nil {
		ref = types.ClaimRef{
			Kind:      claim.Kind,
			Name:      claim.Name,
			Namespace: claim.Namespace,
		}
	}

	var capacity string
	if quantity, ok := k8sPersistentVolume.Spec.Capacity[corev1.ResourceStorage]; ok {
		capacity = quantity.String()
	}

	pv := &types.PersistentVolume{
		Name:             k8sPersistentVolume.Name,
		StorageSize:      capacity,
		StorageClassName: k8sPersistentVolume.Spec.StorageClassName,
		ClaimRef:         ref,
		Status:           string(k8sPersistentVolume.Status.Phase),
	}
	pv.SetID(k8sPersistentVolume.Name)
	pv.SetCreationTimestamp(k8sPersistentVolume.CreationTimestamp.Time)
	if k8sPersistentVolume.GetDeletionTimestamp() != nil {
		pv.SetDeletionTimestamp(k8sPersistentVolume.DeletionTimestamp.Time)
	}
	return pv
}
