package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/gsmlg-opt/gaocloud/pkg/types"
)

type LimitRangeManager struct {
	clusters *ClusterManager
}

func newLimitRangeManager(clusters *ClusterManager) *LimitRangeManager {
	return &LimitRangeManager{clusters: clusters}
}

func (m *LimitRangeManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	limitRange := ctx.Resource.(*types.LimitRange)
	err := createLimitRange(cluster.GetKubeClient(), namespace, limitRange)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate limitRange name %s", limitRange.Name))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create limitRange failed %s", err.Error()))
	}

	limitRange.SetID(limitRange.Name)
	return limitRange, nil
}

func (m *LimitRangeManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sLimitRanges, err := getLimitRanges(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found limitRanges")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list limitRanges failed %s", err.Error()))
	}

	var limitRanges []*types.LimitRange
	for _, item := range k8sLimitRanges.Items {
		limitRanges = append(limitRanges, k8sLimitRangeToSCLimitRange(&item))
	}
	return limitRanges, nil
}

func (m *LimitRangeManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	limitRange := ctx.Resource.(*types.LimitRange)
	k8sLimitRange, err := getLimitRange(cluster.GetKubeClient(), namespace, limitRange.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found limitRange %s", limitRange.GetID()))
		}
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get limitRange %s failed %s", limitRange.GetID(), err.Error()))
	}

	return k8sLimitRangeToSCLimitRange(k8sLimitRange), nil
}

func (m *LimitRangeManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	limitRange := ctx.Resource.(*types.LimitRange)
	if err := deleteLimitRange(cluster.GetKubeClient(), namespace, limitRange.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found limitRange %s", limitRange.GetID()))
		}
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete limitRange %s failed %s", limitRange.GetID(), err.Error()))
	}

	return nil
}

func getLimitRange(cli client.Client, namespace, name string) (*corev1.LimitRange, error) {
	limitRange := corev1.LimitRange{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &limitRange)
	return &limitRange, err
}

func getLimitRanges(cli client.Client, namespace string) (*corev1.LimitRangeList, error) {
	limitRanges := corev1.LimitRangeList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &limitRanges)
	return &limitRanges, err
}

func createLimitRange(cli client.Client, namespace string, limitRange *types.LimitRange) error {
	var k8sLimitRangeItems []corev1.LimitRangeItem
	if len(limitRange.Max) == 0 && len(limitRange.Min) == 0 {
		return fmt.Errorf("limit range must set min or max")
	}

	max, err := scLimitResourceListToK8sResourceList(limitRange.Max)
	if err != nil {
		return fmt.Errorf("parse limitrange max failed: %v", err.Error())
	}

	min, err := scLimitResourceListToK8sResourceList(limitRange.Min)
	if err != nil {
		return fmt.Errorf("parse limitrange min failed: %v", err.Error())
	}

	k8sLimitRangeItems = append(k8sLimitRangeItems, corev1.LimitRangeItem{
		Type: corev1.LimitTypeContainer,
		Max:  max,
		Min:  min,
	})

	k8sLimitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      limitRange.Name,
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: k8sLimitRangeItems,
		},
	}
	return cli.Create(context.TODO(), k8sLimitRange)
}

func scLimitResourceListToK8sResourceList(resourceList map[string]string) (corev1.ResourceList, error) {
	k8sResourceList := make(map[corev1.ResourceName]apiresource.Quantity)
	for name, quantity := range resourceList {
		k8sResourceName, err := scResourceNameToK8sResourceName(name)
		if err != nil {
			return nil, fmt.Errorf("parse resource name %s failed: %s", name, err.Error())
		}

		k8sQuantity, err := apiresource.ParseQuantity(quantity)
		if err != nil {
			return nil, fmt.Errorf("parse resource %s quantity %s failed: %s", name, quantity, err.Error())
		}

		k8sResourceList[k8sResourceName] = k8sQuantity
	}
	return k8sResourceList, nil
}

func deleteLimitRange(cli client.Client, namespace, name string) error {
	limitRange := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), limitRange)
}

func k8sLimitRangeToSCLimitRange(k8sLimitRange *corev1.LimitRange) *types.LimitRange {
	limitRange := &types.LimitRange{
		Name: k8sLimitRange.ObjectMeta.Name,
	}
	for _, limit := range k8sLimitRange.Spec.Limits {
		if limit.Type == corev1.LimitTypeContainer {
			limitRange.Max = k8sResourceListToSCLimitResourceList(limit.Max)
			limitRange.Min = k8sResourceListToSCLimitResourceList(limit.Min)
			break
		}
	}

	limitRange.SetID(k8sLimitRange.Name)
	limitRange.SetCreationTimestamp(k8sLimitRange.CreationTimestamp.Time)
	if k8sLimitRange.GetDeletionTimestamp() != nil {
		limitRange.SetDeletionTimestamp(k8sLimitRange.DeletionTimestamp.Time)
	}
	return limitRange
}

func k8sResourceListToSCLimitResourceList(k8sResourceList corev1.ResourceList) map[string]string {
	resourceList := make(map[string]string)
	for name, quantity := range k8sResourceList {
		if name == corev1.ResourceCPU || name == corev1.ResourceMemory {
			resourceList[string(name)] = quantity.String()
		}
	}
	return resourceList
}
