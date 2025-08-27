package handler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8sstorage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"gok8s/client"
	resterror "gorest/error"
	"gorest/resource"
	"pkg/types"
)

const (
	annStorageProvisioner = "volume.beta.kubernetes.io/storage-provisioner"
)

type PersistentVolumeClaimManager struct {
	clusters *ClusterManager
}

func newPersistentVolumeClaimManager(clusters *ClusterManager) *PersistentVolumeClaimManager {
	return &PersistentVolumeClaimManager{clusters: clusters}
}

func (m *PersistentVolumeClaimManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sPersistentVolumeClaims, err := getPersistentVolumeClaims(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found pvcs")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list pvcs failed %s", err.Error()))
	}

	var pvcs []*types.PersistentVolumeClaim
	for _, item := range k8sPersistentVolumeClaims.Items {
		pvcs = append(pvcs, k8sPVCToSCPVC(&item))
	}
	if err := genUseInfoForPVCs(cluster.GetKubeClient(), namespace, pvcs); err != nil {
		return nil, nil
	}
	return pvcs, nil
}

func (m PersistentVolumeClaimManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	pvc := ctx.Resource.(*types.PersistentVolumeClaim)

	if err := isUsed(cluster.GetKubeClient(), namespace, pvc.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("get pvc %s info failed: %s", pvc.GetID(), err.Error()))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get pvc %s info failed %s", pvc.GetID(), err.Error()))
	}

	err := deletePersistentVolumeClaim(cluster.GetKubeClient(), namespace, pvc.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found pvc %s", pvc.GetID()))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete pvc %s failed %s", pvc.GetID(), err.Error()))
	}
	return nil
}

func getPersistentVolumeClaim(cli client.Client, namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	pvc := corev1.PersistentVolumeClaim{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &pvc)
	return &pvc, err
}

func getPersistentVolumeClaims(cli client.Client, namespace string) (*corev1.PersistentVolumeClaimList, error) {
	pvcs := corev1.PersistentVolumeClaimList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &pvcs)
	return &pvcs, err
}

func deletePersistentVolumeClaim(cli client.Client, namespace, name string) error {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), pvc)
}

func k8sPVCToSCPVC(k8sPersistentVolumeClaim *corev1.PersistentVolumeClaim) *types.PersistentVolumeClaim {
	var storageClassName string
	if k8sPersistentVolumeClaim.Spec.StorageClassName != nil {
		storageClassName = *k8sPersistentVolumeClaim.Spec.StorageClassName
	}

	var requestStorage string
	if quantity, ok := k8sPersistentVolumeClaim.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		requestStorage = quantity.String()
	}

	var actualStorage string
	if quantity, ok := k8sPersistentVolumeClaim.Status.Capacity[corev1.ResourceStorage]; ok {
		actualStorage = quantity.String()
	}

	var driver string
	if provisioner, ok := k8sPersistentVolumeClaim.Annotations[annStorageProvisioner]; ok {
		driver = provisioner
	}

	pvc := &types.PersistentVolumeClaim{
		Name:               k8sPersistentVolumeClaim.Name,
		RequestStorageSize: requestStorage,
		StorageClassName:   storageClassName,
		VolumeName:         k8sPersistentVolumeClaim.Spec.VolumeName,
		ActualStorageSize:  actualStorage,
		Status:             string(k8sPersistentVolumeClaim.Status.Phase),
		Driver:             driver,
	}
	pvc.SetID(k8sPersistentVolumeClaim.Name)
	pvc.SetCreationTimestamp(k8sPersistentVolumeClaim.CreationTimestamp.Time)
	if k8sPersistentVolumeClaim.GetDeletionTimestamp() != nil {
		pvc.SetDeletionTimestamp(k8sPersistentVolumeClaim.DeletionTimestamp.Time)
	}
	return pvc
}

func genUseInfoForPVCs(cli client.Client, namespace string, pvcs []*types.PersistentVolumeClaim) error {
	vas := k8sstorage.VolumeAttachmentList{}
	if err := cli.List(context.TODO(), nil, &vas); err != nil {
		return err
	}
	pods, err := getPods(cli, namespace, labels.Everything())
	if err != nil {
		return err
	}
	for _, pvc := range pvcs {
		if pvc.Status != "Bound" {
			return nil
		}
		if strings.HasSuffix(pvc.Driver, NfsDriverSuffix) {
			if err := genUseInfoForNfsPVC(cli, pvc, pods); err != nil {
				return err
			}
		} else {
			if err := genUseInfoForPVC(cli, pvc, pods, vas); err != nil {
				return err
			}
		}
	}
	return nil
}

func genUseInfoForPVC(cli client.Client, pvc *types.PersistentVolumeClaim, pods *corev1.PodList, vas k8sstorage.VolumeAttachmentList) error {
	pv, err := getPersistentVolume(cli, pvc.VolumeName)
	if err != nil {
		return err
	}
	for _, va := range vas.Items {
		if *va.Spec.Source.PersistentVolumeName == pv.Name {
			pvc.Used = va.Status.Attached
			if pvc.Used && strings.HasSuffix(va.Spec.Attacher, LvmDriverSuffix) {
				pvc.Node = va.Spec.NodeName
			}
		}
	}
	if pvc.Used {
		for _, p := range pods.Items {
			for _, volume := range p.Spec.Volumes {
				if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvc.Name {
					pvc.Pods = append(pvc.Pods, p.Name)
					break
				}
			}
		}
	}
	return nil
}

func isUsed(cli client.Client, namespace, name string) error {
	pods := corev1.PodList{}
	if err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &pods); err != nil {
		return err
	}
	for _, pod := range pods.Items {
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim != nil && name == v.PersistentVolumeClaim.ClaimName {
				return errors.New(fmt.Sprintf("the pvc %s is in used, can not delete it", name))
			}
		}
	}
	return nil
}

func genUseInfoForNfsPVC(cli client.Client, pvc *types.PersistentVolumeClaim, pods *corev1.PodList) error {
	for _, p := range pods.Items {
		for _, volume := range p.Spec.Volumes {
			if volume.PersistentVolumeClaim != nil && volume.PersistentVolumeClaim.ClaimName == pvc.Name {
				pvc.Pods = append(pvc.Pods, p.Name)
				pvc.Used = true
				break
			}
		}
	}
	return nil
}

func (m *PersistentVolumeClaimManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	pvc := ctx.Resource.(*types.PersistentVolumeClaim)
	if err := m.updatePersistentVolumeClaim(cluster.GetKubeClient(), namespace, pvc); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update persistentVolumeClaim failed, %s", err.Error()))
	}
	return pvc, nil
}

func (m *PersistentVolumeClaimManager) updatePersistentVolumeClaim(cli client.Client, namespace string, pvc *types.PersistentVolumeClaim) error {
	k8sPvc, err := getPersistentVolumeClaim(cli, namespace, pvc.Name)
	if err != nil {
		return err
	}
	if provisioner, ok := k8sPvc.Annotations[annStorageProvisioner]; ok {
		if strings.HasSuffix(provisioner, IscsiDriverSuffix) || strings.HasSuffix(provisioner, NfsDriverSuffix) {
			return errors.New(fmt.Sprintf("the driver %s or pvc %s unsupport expand", provisioner, pvc.Name))
		}
		if strings.HasSuffix(provisioner, LvmDriverSuffix) && pvc.Used {
			return errors.New(fmt.Sprintf("pvc %s can not expand online, you should detach first", pvc.Name))
		}
	}
	if pvc.ActualStorageSize != "" {
		quantity, err := apiresource.ParseQuantity(pvc.ActualStorageSize)
		if err != nil {
			return fmt.Errorf("parse storage size %s failed: %s", pvc.ActualStorageSize, err.Error())
		}
		expectedQuantity := &quantity
		if currentQuantity, ok := k8sPvc.Status.Capacity[corev1.ResourceStorage]; ok {
			if expectedQuantity.Value() <= currentQuantity.Value() {
				return errors.New(fmt.Sprintf("pvc %s unsupport reduce", pvc.Name))
			}
		}
		k8sPvc.Spec.Resources.Requests[corev1.ResourceStorage] = quantity
		return cli.Update(context.TODO(), k8sPvc)
	}
	return nil
}
