package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"gok8s/client"
	resterror "gorest/error"
	"gorest/resource"
	eb "pkg/eventbus"
	"pkg/types"
)

const (
	ChangeCauseAnnotation       = "kubernetes.io/change-cause"
	LastAppliedConfigAnnotation = "kubectl.kubernetes.io/last-applied-configuration"
	RevisionAnnotation          = "deployment.kubernetes.io/revision"
	RevisionHistoryAnnotation   = "deployment.kubernetes.io/revision-history"
	DesiredReplicasAnnotation   = "deployment.kubernetes.io/desired-replicas"
	MaxReplicasAnnotation       = "deployment.kubernetes.io/max-replicas"
	DeprecatedRollbackTo        = "deprecated.deployment.rollback.to"
)

var AnnotationsToSkip = map[string]bool{
	LastAppliedConfigAnnotation: true,
	RevisionAnnotation:          true,
	RevisionHistoryAnnotation:   true,
	DesiredReplicasAnnotation:   true,
	MaxReplicasAnnotation:       true,
	DeprecatedRollbackTo:        true,
}

type DeploymentManager struct {
	clusters *ClusterManager
}

func newDeploymentManager(clusters *ClusterManager) *DeploymentManager {
	return &DeploymentManager{clusters: clusters}
}

func (m *DeploymentManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	if err := createDeployment(cluster.GetKubeClient(), namespace, deploy); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate deploy name %s", deploy.Name))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create deploy failed %s", err.Error()))
	}

	deploy.SetID(deploy.Name)
	return deploy, nil
}

func (m *DeploymentManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sDeploys, err := getDeployments(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found deploys")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list deploys failed %s", err.Error()))
	}

	var deploys []*types.Deployment
	for _, item := range k8sDeploys.Items {
		deploy, err := k8sDeployToSCDeploy(cluster.GetKubeClient(), &item)
		if err != nil {
			return nil, err
		}
		deploys = append(deploys, deploy)
	}

	return deploys, nil
}

func (m *DeploymentManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, err := getDeployment(cluster.GetKubeClient(), namespace, deploy.GetID())
	if err != nil {
		return nil, err
	}

	return k8sDeployToSCDeploy(cluster.GetKubeClient(), k8sDeploy)
}

func (m *DeploymentManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, apiErr := getDeployment(cluster.GetKubeClient(), namespace, deploy.GetID())
	if apiErr != nil {
		return nil, apiErr
	}

	k8sPodSpec, _, err := scPodSpecToK8sPodSpecAndPVCs(cluster.GetKubeClient(), deploy.Containers, deploy.PersistentVolumes)
	if err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update deployment failed %s", err.Error()))
	}

	k8sDeploy.Spec.Template.Spec.Containers = k8sPodSpec.Containers
	k8sDeploy.Spec.Template.Spec.Volumes = k8sPodSpec.Volumes
	k8sDeploy.Annotations = addWorkloadUpdateMemoToAnnotations(k8sDeploy.Annotations, deploy.Memo)
	if err := cluster.GetKubeClient().Update(context.TODO(), k8sDeploy); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update deployment failed %s", err.Error()))
	}

	return deploy, nil
}

func (m *DeploymentManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, err := getDeployment(cluster.GetKubeClient(), namespace, deploy.GetID())
	if err != nil {
		return err
	}

	if err := deleteDeployment(cluster.GetKubeClient(), namespace, deploy.GetID()); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete deployment failed %s", err.Error()))
	}

	if delete, ok := k8sDeploy.Annotations[AnnkeyForDeletePVsWhenDeleteWorkload]; ok && delete == "true" {
		deleteWorkLoadPVCs(cluster.GetKubeClient(), namespace, k8sDeploy.Spec.Template.Spec.Volumes)
	}
	eb.PublishResourceDeleteEvent(deploy)
	return nil
}

func (m *DeploymentManager) Action(ctx *resource.Context) (interface{}, *resterror.APIError) {
	switch ctx.Resource.GetAction().Name {
	case types.ActionGetHistory:
		return m.getDeploymentHistory(ctx)
	case types.ActionRollback:
		return nil, m.rollback(ctx)
	case types.ActionSetPodCount:
		return m.setPodCount(ctx)
	default:
		return nil, resterror.NewAPIError(resterror.InvalidAction, fmt.Sprintf("action %s is unknown", ctx.Resource.GetAction().Name))
	}
}

func getDeployment(cli client.Client, namespace, name string) (*appsv1.Deployment, *resterror.APIError) {
	deploy := appsv1.Deployment{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &deploy); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found deploy %s", name))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get deploy %s failed %s", name, err.Error()))
	}
	return &deploy, nil
}

func getDeployments(cli client.Client, namespace string) (*appsv1.DeploymentList, error) {
	deploys := appsv1.DeploymentList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &deploys)
	return &deploys, err
}

func createDeployment(cli client.Client, namespace string, deploy *types.Deployment) error {
	podTemplate, k8sPVCs, err := createPodTempateSpec(namespace, deploy, cli)
	if err != nil {
		return err
	}

	replicas := int32(deploy.Replicas)
	k8sDeploy := &appsv1.Deployment{
		ObjectMeta: generatePodOwnerObjectMeta(namespace, deploy),
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": deploy.Name},
			},
			Template: *podTemplate,
		},
	}

	if err := cli.Create(context.TODO(), k8sDeploy); err != nil {
		deletePVCs(cli, namespace, k8sPVCs)
		return err
	}

	return nil
}

func deleteDeployment(cli client.Client, namespace, name string) error {
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), deploy)
}

func k8sDeployToSCDeploy(cli client.Client, k8sDeploy *appsv1.Deployment) (*types.Deployment, *resterror.APIError) {
	containers, templates := k8sPodSpecToScContainersAndVCTemplates(k8sDeploy.Spec.Template.Spec.Containers,
		k8sDeploy.Spec.Template.Spec.Volumes)

	pvs, err := getPVCs(cli, k8sDeploy.Namespace, templates)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("get deploy %s pvc failed: %s", k8sDeploy.Name, err.Error()))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get deploy %s pvc failed: %s", k8sDeploy.Name, err.Error()))
	}

	var advancedOpts types.AdvancedOptions
	if opts, ok := k8sDeploy.Annotations[AnnkeyForWordloadAdvancedoption]; ok {
		json.Unmarshal([]byte(opts), &advancedOpts)
	}

	deploy := &types.Deployment{
		Name:              k8sDeploy.Name,
		Replicas:          int(*k8sDeploy.Spec.Replicas),
		Containers:        containers,
		PersistentVolumes: pvs,
		AdvancedOptions:   advancedOpts,
		Status:            types.WorkloadStatus{ReadyReplicas: int(k8sDeploy.Status.ReadyReplicas)},
	}

	if deployIsUpdating(k8sDeploy) {
		status, err := getDeploymentUpdateStatus(cli, k8sDeploy)
		if err != nil {
			return nil, resterror.NewAPIError(resterror.ServerError,
				fmt.Sprintf("get deploy %s status failed: %s", k8sDeploy.Name, err.Error()))
		}

		deploy.Status = status
	}

	deploy.Status.Conditions = k8sWorkloadConditionsToScWorkloadConditions(k8sDeploy.Status.Conditions, true)
	deploy.SetID(k8sDeploy.Name)
	deploy.SetCreationTimestamp(k8sDeploy.CreationTimestamp.Time)
	if k8sDeploy.GetDeletionTimestamp() != nil {
		deploy.SetDeletionTimestamp(k8sDeploy.DeletionTimestamp.Time)
	}
	deploy.AdvancedOptions.ExposedMetric = k8sAnnotationsToScExposedMetric(k8sDeploy.Spec.Template.Annotations)
	return deploy, nil
}

func getDeploymentUpdateStatus(cli client.Client, k8sDeploy *appsv1.Deployment) (types.WorkloadStatus, error) {
	version, ok := k8sDeploy.Annotations[RevisionAnnotation]
	if ok == false {
		return types.WorkloadStatus{}, fmt.Errorf("deploy %s should has annotation deployment.kubernetes.io/revision", k8sDeploy.Name)
	}

	currentVersion, _ := strconv.Atoi(version)
	rss, err := getReplicaSets(cli, k8sDeploy)
	if err != nil {
		return types.WorkloadStatus{}, err
	}

	if len(rss) < 2 {
		return types.WorkloadStatus{}, fmt.Errorf("deploy %s should has at least two replicasets when updating", k8sDeploy.Name)
	}

	workloadStatus := types.WorkloadStatus{
		ReadyReplicas: int(k8sDeploy.Status.ReadyReplicas),
		Updating:      true,
	}
	for _, rs := range rss {
		if v, ok := rs.Annotations[RevisionAnnotation]; ok {
			if v == version {
				workloadStatus.UpdatedReplicas = int(rs.Status.ReadyReplicas)
				workloadStatus.UpdatingReplicas = int(rs.Status.Replicas - rs.Status.ReadyReplicas)
			} else if v == strconv.Itoa(currentVersion-1) {
				if rs.Status.Replicas == 0 {
					return types.WorkloadStatus{ReadyReplicas: int(k8sDeploy.Status.ReadyReplicas)}, nil
				}

				workloadStatus.CurrentReplicas = int(rs.Status.Replicas)
			}
		}
	}

	return workloadStatus, nil
}

func deployIsUpdating(k8sDeploy *appsv1.Deployment) bool {
	if v, ok := k8sDeploy.Annotations[RevisionAnnotation]; ok {
		if v == "1" {
			return false
		}
	}

	if k8sDeploy.Status.ObservedGeneration == 1 ||
		(*k8sDeploy.Spec.Replicas == k8sDeploy.Status.Replicas &&
			k8sDeploy.Status.Replicas == k8sDeploy.Status.ReadyReplicas &&
			k8sDeploy.Status.ReadyReplicas == k8sDeploy.Status.UpdatedReplicas &&
			k8sDeploy.Status.UnavailableReplicas == 0) {
		return false
	}

	return true
}

func (m *DeploymentManager) getDeploymentHistory(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	_, replicasets, err := getDeploymentAndReplicaSets(cluster.GetKubeClient(), namespace, deploy.GetID())
	if err != nil {
		return nil, err
	}

	var versionInfos types.VersionInfos
	for _, rs := range replicasets {
		if v, ok := rs.Annotations[RevisionAnnotation]; ok {
			version, _ := strconv.Atoi(v)
			containers, _ := k8sPodSpecToScContainersAndVCTemplates(rs.Spec.Template.Spec.Containers, rs.Spec.Template.Spec.Volumes)
			versionInfos = append(versionInfos, types.VersionInfo{
				Name:         deploy.GetID(),
				Namespace:    namespace,
				Version:      version,
				ChangeReason: rs.Annotations[ChangeCauseAnnotation],
				Containers:   containers,
			})
		}
	}

	sort.Sort(versionInfos)
	return &types.VersionHistory{
		VersionInfos: versionInfos[:len(versionInfos)-1],
	}, nil
}

func (m *DeploymentManager) rollback(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	param, ok := ctx.Resource.GetAction().Input.(*types.RollBackVersion)
	if ok == false {
		return resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("action rollback version param is not valid"))
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, replicasets, apiErr := getDeploymentAndReplicaSets(cluster.GetKubeClient(), namespace, deploy.GetID())
	if apiErr != nil {
		return apiErr
	}

	if k8sDeploy.Spec.Paused {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("cannot rollback a paused deployment"))
	}

	var rsForVersion *appsv1.ReplicaSet
	for _, replicaset := range replicasets {
		if v, ok := replicaset.Annotations[RevisionAnnotation]; ok {
			if v == strconv.Itoa(param.Version) {
				rsForVersion = &replicaset
				break
			}
		}
	}

	if rsForVersion == nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback deployment failed no found version"))
	}

	delete(rsForVersion.Spec.Template.Labels, appsv1.DefaultDeploymentUniqueLabelKey)
	annotations := map[string]string{}
	for k := range AnnotationsToSkip {
		if v, ok := k8sDeploy.Annotations[k]; ok {
			annotations[k] = v
		}
	}
	for k, v := range rsForVersion.Annotations {
		if !AnnotationsToSkip[k] {
			annotations[k] = v
		}
	}

	annotations[ChangeCauseAnnotation] = param.Memo
	patch, err := marshalPatch(rsForVersion.Spec.Template, annotations)
	if err != nil {
		return resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("marshal deployment patch when rollback failed: %v", err.Error()))
	}

	if err := cluster.GetKubeClient().Patch(context.TODO(), k8sDeploy, k8stypes.JSONPatchType, patch); err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("rollback deployment failed: %v", err.Error()))
	}

	return nil
}

func (m *DeploymentManager) setPodCount(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	param, ok := ctx.Resource.GetAction().Input.(*types.SetPodCount)
	if ok == false {
		return nil, resterror.NewAPIError(resterror.InvalidFormat, "action set pod count param is not valid")
	}

	namespace := ctx.Resource.GetParent().GetID()
	deploy := ctx.Resource.(*types.Deployment)
	k8sDeploy, err := getDeployment(cluster.GetKubeClient(), namespace, deploy.GetID())
	if err != nil {
		return nil, err
	}

	if int(*k8sDeploy.Spec.Replicas) != param.Replicas {
		if err := cluster.GetKubeClient().Patch(context.TODO(), k8sDeploy, k8stypes.MergePatchType,
			[]byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, param.Replicas))); err != nil {
			return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("set deployment pod count failed: %v", err.Error()))
		}
	}

	return param, nil
}

func getDeploymentAndReplicaSets(cli client.Client, namespace, name string) (*appsv1.Deployment, []appsv1.ReplicaSet, *resterror.APIError) {
	k8sDeploy, apiErr := getDeployment(cli, namespace, name)
	if apiErr != nil {
		return nil, nil, apiErr
	}

	if k8sDeploy.Spec.Selector == nil {
		return nil, nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("deploy %v has no selector", name))
	}

	rss, err := getReplicaSets(cli, k8sDeploy)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found deploy %v replicasets", name))
		}
		return nil, nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get deploy %s replicasets failed: %s", name, err.Error()))
	}

	return k8sDeploy, rss, nil
}

func getReplicaSets(cli client.Client, k8sDeploy *appsv1.Deployment) ([]appsv1.ReplicaSet, error) {
	replicasets := appsv1.ReplicaSetList{}
	opts := &client.ListOptions{Namespace: k8sDeploy.Namespace}
	labels, err := metav1.LabelSelectorAsSelector(k8sDeploy.Spec.Selector)
	if err != nil {
		return nil, err
	}

	opts.LabelSelector = labels
	if err := cli.List(context.TODO(), opts, &replicasets); err != nil {
		return nil, err
	}

	var replicaSetsByDeployControled []appsv1.ReplicaSet
	for _, item := range replicasets.Items {
		if isControllerBy(item.OwnerReferences, k8sDeploy.UID) {
			replicaSetsByDeployControled = append(replicaSetsByDeployControled, item)
		}
	}

	return replicaSetsByDeployControled, nil
}
