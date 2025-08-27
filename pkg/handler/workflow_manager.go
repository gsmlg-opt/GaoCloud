package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gsmlg-opt/gaocloud/pkg/types"

	tektonv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"
)

const (
	zcloudWorkFlowContentAnnotationKey      = "workflow.zdns.cn/content"
	zcloudWorkFlowIDLabelKey                = "workflow.zdns.cn/id"
	zcloudWorkFlowLatestTaskIDAnnotationKey = "workflow.zdns.cn/latest-task-id"

	zcloudWorkFlowClusterRoleBindingName = "zcloud-workflow-deployer"
)

type WorkFlowManager struct {
	clusters *ClusterManager
}

func newWorkFlowManager(clusters *ClusterManager) *WorkFlowManager {
	return &WorkFlowManager{
		clusters: clusters,
	}
}

func (m *WorkFlowManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	ns := ctx.Resource.GetParent().GetID()
	wf := ctx.Resource.(*types.WorkFlow)
	wf.SetCreationTimestamp(time.Now())
	wf.SetID(wf.Name)

	if err := preCheckDeploymentExist(cluster.GetKubeClient(), ns, wf.Name); err != nil {
		return nil, err
	}
	if err := createWorkFlow(cluster.GetKubeClient(), ns, wf); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, "workflow already exists")
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create workflow %s failed %s", wf.Name, err.Error()))
	}
	return wf, nil
}

func preCheckDeploymentExist(cli client.Client, namespace, name string) *resterror.APIError {
	_, err := getDeployment(cli, namespace, name)
	if err != nil {
		if err.ErrorCode == resterror.NotFound {
			return nil
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get deploy failed for pre check deploy name %s", err.Error()))
	}
	return resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("workflow deploy %s already exist", name))
}

func createWorkFlow(cli client.Client, namespace string, wf *types.WorkFlow) error {
	var gitSecretName string
	createdObjs := []runtime.Object{}
	gitSecret, err := genWorkFlowGitSecret(namespace, wf)
	if err != nil {
		return err
	}
	if gitSecret != nil {
		if err := cli.Create(context.TODO(), gitSecret); err != nil {
			return err
		}
		gitSecretName = gitSecret.Name
		createdObjs = append(createdObjs, gitSecret)
	}

	dockerSecret := genWorkFlowDockerSecret(namespace, wf)
	if err := cli.Create(context.TODO(), dockerSecret); err != nil {
		workFlowCreateFailBack(cli, namespace, wf.Name, createdObjs)
		return err
	}
	createdObjs = append(createdObjs, dockerSecret)

	sa := genWorkFlowServiceAccount(wf.Name, namespace, gitSecretName, dockerSecret.Name)
	if err := cli.Create(context.TODO(), sa); err != nil {
		workFlowCreateFailBack(cli, namespace, wf.Name, createdObjs)
		return err
	}
	createdObjs = append(createdObjs, sa)

	if err := addWorkFlowSaToCRB(cli, wf.Name, namespace); err != nil {
		workFlowCreateFailBack(cli, namespace, wf.Name, createdObjs)
		return err
	}

	pipelineResource, err := genGitPipelineResource(cli, namespace, wf)
	if err != nil {
		workFlowCreateFailBack(cli, namespace, wf.Name, createdObjs)
		return err
	}

	if err := cli.Create(context.TODO(), pipelineResource); err != nil {
		workFlowCreateFailBack(cli, namespace, wf.Name, createdObjs)
		return err
	}
	return nil
}

func workFlowCreateFailBack(cli client.Client, namespace, name string, objs []runtime.Object) {
	for _, obj := range objs {
		if err := cli.Delete(context.TODO(), obj); err != nil {
			log.Warnf("delete k8s object failed in workflow create failback %s", err.Error())
		}
	}
	if err := deleteWorkFlowSaFromCRB(cli, name, namespace); err != nil {
		log.Warnf("delete workflow %s_%s serviceaccount failed in workflow create failback %s", namespace, name, err.Error())
	}
}

func (m *WorkFlowManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	ns := ctx.Resource.GetParent().GetID()
	id := ctx.Resource.GetID()

	wf, err := getWorkFlow(cluster.GetKubeClient(), ns, id)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("workflow %s doesn't exist", id))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get workflow %s failed %s", id, err.Error()))
	}
	return wf, nil
}

func getWorkFlow(cli client.Client, namespace, name string) (*types.WorkFlow, error) {
	pr := tektonv1alpha1.PipelineResource{}
	if err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &pr); err != nil {
		return nil, err
	}
	return pipelineResourceToScWorkFlow(cli, namespace, pr)
}

func pipelineResourceToScWorkFlow(cli client.Client, namespace string, pr tektonv1alpha1.PipelineResource) (*types.WorkFlow, error) {
	wf := &types.WorkFlow{}
	wfContent, ok := pr.Annotations[zcloudWorkFlowContentAnnotationKey]
	if !ok {
		return nil, fmt.Errorf("workflowcontentannotation doesn't exist")
	}
	if err := json.Unmarshal([]byte(wfContent), wf); err != nil {
		return nil, err
	}

	wftID, ok := pr.Annotations[zcloudWorkFlowLatestTaskIDAnnotationKey]
	if wftID != "" && ok {
		subs, status, err := getWorkFlowSubTasksAndStatus(cli, namespace, wftID)
		if err != nil {
			return nil, err
		}
		wf.SubTasks = subs
		wf.Status = status
	} else {
		wf.SubTasks = nil
		wf.Status = types.WorkFlowTaskStatus{}
	}

	if pr.DeletionTimestamp != nil {
		wf.SetDeletionTimestamp(pr.DeletionTimestamp.Time)
	}
	return wf, nil
}

func getWorkFlowSubTasksAndStatus(cli client.Client, namespace, wftID string) ([]types.WorkFlowSubTask, types.WorkFlowTaskStatus, error) {
	wft, err := getWorkFlowTask(cli, namespace, wftID)
	if err != nil {
		return nil, types.WorkFlowTaskStatus{}, err
	}

	return wft.SubTasks, wft.Status, nil
}

func (m *WorkFlowManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}
	ns := ctx.Resource.GetParent().GetID()

	wfs, err := getWorkFlows(cluster.GetKubeClient(), ns)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list %s workflow failed %s", ns, err.Error()))
	}
	return wfs, nil
}

func getWorkFlows(cli client.Client, namespace string) ([]*types.WorkFlow, error) {
	prs := tektonv1alpha1.PipelineResourceList{}
	if err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &prs); err != nil {
		return nil, err
	}

	wfs := types.WorkFlows{}
	for _, pr := range prs.Items {
		wf, err := pipelineResourceToScWorkFlow(cli, namespace, pr)
		if err != nil {
			return nil, err
		}
		wfs = append(wfs, wf)
	}

	sort.Sort(wfs)
	return wfs, nil
}

func (m *WorkFlowManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, nil
	}

	ns := ctx.Resource.GetParent().GetID()
	newer := ctx.Resource.(*types.WorkFlow)

	older, err := getWorkFlow(cluster.GetKubeClient(), ns, newer.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "workflow doesn't exist")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get workflow %s failed %s", newer.GetID(), err.Error()))
	}
	newer.AutoDeploy = older.AutoDeploy
	newer.Name = older.Name

	if err := updateWorkFlow(cluster.GetKubeClient(), ns, newer); err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update workflow %s failed %s", newer.Name, err.Error()))
	}
	return newer, nil
}

func updateWorkFlow(cli client.Client, namespace string, wf *types.WorkFlow) error {
	if err := updateWorkFlowSecrets(cli, namespace, wf); err != nil {
		return err
	}

	return updateGitPipelineResource(cli, namespace, wf)
}

func (m *WorkFlowManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	ns := ctx.Resource.GetParent().GetID()
	id := ctx.Resource.GetID()

	if err := emptyWorkFlowTask(cluster.GetKubeClient(), ns, id); err != nil {
		return nil
	}

	if err := deleteWorkFlow(cluster.GetKubeClient(), ns, id); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, "workflow doesn't exist")
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete workflow %s failed %s", id, err.Error()))
	}
	return nil
}

func deleteWorkFlow(cli client.Client, namespace, name string) error {
	if err := deletePipelineResource(cli, namespace, name); err != nil {
		return err
	}

	if err := deleteWorkFlowSaFromCRB(cli, name, namespace); err != nil {
		return err
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace},
	}
	if err := cli.Delete(context.TODO(), sa); err != nil {
		return err
	}
	if err := deleteWorkFlowSecrets(cli, namespace, name); err != nil {
		return err
	}
	return deleteWorkFlowDeploymentAndPVCs(cli, namespace, name)
}

func (m *WorkFlowManager) Action(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	action := ctx.Resource.GetAction()
	ns := ctx.Resource.GetParent().GetID()
	id := ctx.Resource.GetID()

	switch action.Name {
	case types.WorkFlowEmptyTaskAction:
		return nil, emptyWorkFlowTask(cluster.GetKubeClient(), ns, id)
	default:
		return nil, nil
	}
}

func emptyWorkFlowTask(cli client.Client, namespace, name string) *resterror.APIError {
	if err := updateWorkFlowLastestIDAnnotation(cli, namespace, name, ""); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("update namespace %s workflow %s latest id annotation failed %s", namespace, name, err.Error()))
	}
	if err := deletePipelineRunsByWorkFlowName(cli, namespace, name); err != nil {
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete namespace %s workflow %s pipelineruns failed %s", namespace, name, err.Error()))
	}
	return nil
}
