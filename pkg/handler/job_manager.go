package handler

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/gsmlg-opt/gaocloud/pkg/types"
)

type JobManager struct {
	clusters *ClusterManager
}

func newJobManager(clusters *ClusterManager) *JobManager {
	return &JobManager{clusters: clusters}
}

func (m *JobManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	job := ctx.Resource.(*types.Job)
	err := createJob(cluster.GetKubeClient(), namespace, job)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate job name %s", job.Name))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create job failed %s", err.Error()))
	}

	job.SetID(job.Name)
	return job, nil
}

func (m *JobManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sJobs, err := getJobs(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found jobs")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list jobs failed %s", err.Error()))
	}

	var jobs []*types.Job
	for _, item := range k8sJobs.Items {
		if len(item.OwnerReferences) == 0 {
			jobs = append(jobs, k8sJobToSCJob(&item))
		}
	}
	return jobs, nil
}

func (m *JobManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	job := ctx.Resource.(*types.Job)
	k8sJob, err := getJob(cluster.GetKubeClient(), namespace, job.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found job %s", job.GetID()))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get job %s failed %s", job.GetID(), err.Error()))
	}

	return k8sJobToSCJob(k8sJob), nil
}

func (m *JobManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	job := ctx.Resource.(*types.Job)
	if err := deleteJob(cluster.GetKubeClient(), namespace, job.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found job %s", job.GetID()))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete job %s failed %s", job.GetID(), err.Error()))
	}

	return nil
}

func getJob(cli client.Client, namespace, name string) (*batchv1.Job, error) {
	job := batchv1.Job{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &job)
	return &job, err
}

func getJobs(cli client.Client, namespace string) (*batchv1.JobList, error) {
	jobs := batchv1.JobList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &jobs)
	return &jobs, err
}

func createJob(cli client.Client, namespace string, job *types.Job) error {
	k8sPodSpec, _, err := scPodSpecToK8sPodSpecAndPVCs(nil, job.Containers, nil)
	if err != nil {
		return err
	}

	policy, err := scRestartPolicyToK8sRestartPolicy(job.RestartPolicy)
	if err != nil {
		return err
	}

	k8sPodSpec.RestartPolicy = policy
	k8sJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.Name,
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"job-name": job.Name},
			},
			Template: corev1.PodTemplateSpec{
				Spec: k8sPodSpec,
			},
		},
	}
	return cli.Create(context.TODO(), k8sJob)
}

func deleteJob(cli client.Client, namespace, name string) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), job, client.PropagationPolicy(metav1.DeletePropagationForeground))
}

func k8sJobToSCJob(k8sJob *batchv1.Job) *types.Job {
	containers, _ := k8sPodSpecToScContainersAndVCTemplates(k8sJob.Spec.Template.Spec.Containers,
		k8sJob.Spec.Template.Spec.Volumes)

	var conditions []types.JobCondition
	for _, condition := range k8sJob.Status.Conditions {
		conditions = append(conditions, types.JobCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastProbeTime:      resource.ISOTime(condition.LastProbeTime.Time),
			LastTransitionTime: resource.ISOTime(condition.LastTransitionTime.Time),
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	jobStatus := types.JobStatus{
		StartTime:      k8sMetaV1TimePtrToISOTime(k8sJob.Status.StartTime),
		CompletionTime: k8sMetaV1TimePtrToISOTime(k8sJob.Status.CompletionTime),
		Active:         k8sJob.Status.Active,
		Succeeded:      k8sJob.Status.Succeeded,
		Failed:         k8sJob.Status.Failed,
		JobConditions:  conditions,
	}

	job := &types.Job{
		Name:          k8sJob.Name,
		RestartPolicy: string(k8sJob.Spec.Template.Spec.RestartPolicy),
		Containers:    containers,
		Status:        jobStatus,
	}
	job.SetID(k8sJob.Name)
	job.SetCreationTimestamp(k8sJob.CreationTimestamp.Time)
	if k8sJob.GetDeletionTimestamp() != nil {
		job.SetDeletionTimestamp(k8sJob.DeletionTimestamp.Time)
	}
	return job
}

func k8sMetaV1TimePtrToISOTime(metav1Time *metav1.Time) (isoTime resource.ISOTime) {
	if metav1Time != nil {
		isoTime = resource.ISOTime(metav1Time.Time)
	}

	return isoTime
}
