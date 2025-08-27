package handler

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"gok8s/client"
	resterror "gorest/error"
	"gorest/resource"
	"pkg/types"
)

type CronJobManager struct {
	clusters *ClusterManager
}

func newCronJobManager(clusters *ClusterManager) *CronJobManager {
	return &CronJobManager{clusters: clusters}
}

func (m *CronJobManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	cronJob := ctx.Resource.(*types.CronJob)
	err := createCronJob(cluster.GetKubeClient(), namespace, cronJob)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate cronJob name %s", cronJob.Name))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create cronJob failed %s", err.Error()))
	}

	cronJob.SetID(cronJob.Name)
	return cronJob, nil
}

func (m *CronJobManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sCronJobs, err := getCronJobs(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found cronjobs")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list cronJobs failed %s", err.Error()))
	}

	var cronJobs []*types.CronJob
	for _, item := range k8sCronJobs.Items {
		cronJobs = append(cronJobs, k8sCronJobToScCronJob(&item))
	}
	return cronJobs, nil
}

func (m *CronJobManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	cronJob := ctx.Resource.(*types.CronJob)
	k8sCronJob, err := getCronJob(cluster.GetKubeClient(), namespace, cronJob.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found cronjob %s", cronJob.GetID()))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get cronJob %s failed %s", cronJob.GetID(), err.Error()))
	}

	return k8sCronJobToScCronJob(k8sCronJob), nil
}

func (m *CronJobManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	cronJob := ctx.Resource.(*types.CronJob)
	if err := deleteCronJob(cluster.GetKubeClient(), namespace, cronJob.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("cronJob %s doesn't exist", cronJob.GetID()))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete cronJob failed %s", err.Error()))
	}

	return nil
}

func getCronJob(cli client.Client, namespace, name string) (*batchv1beta1.CronJob, error) {
	cronJob := batchv1beta1.CronJob{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &cronJob)
	return &cronJob, err
}

func getCronJobs(cli client.Client, namespace string) (*batchv1beta1.CronJobList, error) {
	cronJobs := batchv1beta1.CronJobList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &cronJobs)
	return &cronJobs, err
}

func createCronJob(cli client.Client, namespace string, cronJob *types.CronJob) error {
	k8sPodSpec, _, err := scPodSpecToK8sPodSpecAndPVCs(nil, cronJob.Containers, nil)
	if err != nil {
		return err
	}

	policy, err := scRestartPolicyToK8sRestartPolicy(cronJob.RestartPolicy)
	if err != nil {
		return err
	}

	k8sPodSpec.RestartPolicy = policy
	k8sCronJob := &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cronJob.Name,
			Namespace: namespace,
		},
		Spec: batchv1beta1.CronJobSpec{
			Schedule: cronJob.Schedule,
			JobTemplate: batchv1beta1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: k8sPodSpec,
					},
				},
			},
		},
	}
	return cli.Create(context.TODO(), k8sCronJob)
}

func deleteCronJob(cli client.Client, namespace, name string) error {
	cronJob := &batchv1beta1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), cronJob, client.PropagationPolicy(metav1.DeletePropagationForeground))
}

func k8sCronJobToScCronJob(k8sCronJob *batchv1beta1.CronJob) *types.CronJob {
	containers, _ := k8sPodSpecToScContainersAndVCTemplates(k8sCronJob.Spec.JobTemplate.Spec.Template.Spec.Containers,
		k8sCronJob.Spec.JobTemplate.Spec.Template.Spec.Volumes)

	var objectReferences []types.ObjectReference
	for _, objectReference := range k8sCronJob.Status.Active {
		objectReferences = append(objectReferences, types.ObjectReference{
			Kind:            objectReference.Kind,
			Namespace:       objectReference.Namespace,
			Name:            objectReference.Name,
			UID:             string(objectReference.UID),
			APIVersion:      objectReference.APIVersion,
			ResourceVersion: objectReference.ResourceVersion,
			FieldPath:       objectReference.FieldPath,
		})
	}

	cronJobStatus := types.CronJobStatus{
		LastScheduleTime: k8sMetaV1TimePtrToISOTime(k8sCronJob.Status.LastScheduleTime),
		ObjectReferences: objectReferences,
	}

	cronJob := &types.CronJob{
		Name:          k8sCronJob.Name,
		Schedule:      k8sCronJob.Spec.Schedule,
		RestartPolicy: string(k8sCronJob.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy),
		Containers:    containers,
		Status:        cronJobStatus,
	}
	cronJob.SetID(k8sCronJob.Name)
	cronJob.SetCreationTimestamp(k8sCronJob.CreationTimestamp.Time)
	if k8sCronJob.GetDeletionTimestamp() != nil {
		cronJob.SetDeletionTimestamp(k8sCronJob.DeletionTimestamp.Time)
	}
	return cronJob
}
