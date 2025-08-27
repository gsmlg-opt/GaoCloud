package handler

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"gok8s/client"
	resterror "gorest/error"
	"gorest/resource"
	"pkg/types"
)

var (
	ErrDuplicateKeyInSecret = errors.New("duplicate key in secret")
	ErrUpdateDeletingSecret = errors.New("secret is deleting")
)

type SecretManager struct {
	clusters *ClusterManager
}

func newSecretManager(clusters *ClusterManager) *SecretManager {
	return &SecretManager{clusters: clusters}
}

func (m *SecretManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	secret := ctx.Resource.(*types.Secret)
	if err := createSecret(cluster.GetKubeClient(), namespace, secret); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate secret name %s", secret.Name))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create secret failed %s", err.Error()))
	}

	secret.SetID(secret.Name)
	return secret, nil
}

func (m *SecretManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	secret := ctx.Resource.(*types.Secret)
	if err := updateSecret(cluster.GetKubeClient(), namespace, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found secret %s", secret.GetID()))
		}
		return nil, resterror.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("update secret %s failed %s", secret.GetID(), err.Error()))
	} else {
		return secret, nil
	}
}

func (m *SecretManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sSecrets, err := getSecrets(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found secrets")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list secrets failed %s", err.Error()))
	}

	var secrets []*types.Secret
	for _, secret := range k8sSecrets.Items {
		secrets = append(secrets, k8sSecretToSCSecret(&secret))
	}
	return secrets, nil
}

func (m SecretManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	secret := ctx.Resource.(*types.Secret)
	k8sSecret, err := getSecret(cluster.GetKubeClient(), namespace, secret.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found secret %s", secret.GetID()))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get secret %s failed %s", secret.GetID(), err.Error()))
	}

	return k8sSecretToSCSecret(k8sSecret), nil
}

func (m SecretManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn`t exists")
	}

	namespace := ctx.Resource.GetParent().GetID()
	secret := ctx.Resource.(*types.Secret)
	err := deleteSecret(cluster.GetKubeClient(), namespace, secret.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found secret %s", secret.GetID()))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete secret failed %s", err.Error()))
	}
	return nil
}

func getSecret(cli client.Client, namespace, name string) (*corev1.Secret, error) {
	secret := corev1.Secret{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &secret)
	return &secret, err
}

func getSecrets(cli client.Client, namespace string) (*corev1.SecretList, error) {
	secrets := corev1.SecretList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &secrets)
	return &secrets, err
}

func createSecret(cli client.Client, namespace string, secret *types.Secret) error {
	k8sSecret, err := scSecretToK8sSecret(secret, namespace)
	if err != nil {
		return err
	} else {
		return cli.Create(context.TODO(), k8sSecret)
	}
}

func updateSecret(cli client.Client, namespace string, secret *types.Secret) error {
	target, err := getSecret(cli, namespace, secret.GetID())
	if err != nil {
		return err
	}

	if target.GetDeletionTimestamp() != nil {
		return ErrUpdateDeletingSecret
	}

	k8sSecret, err := scSecretToK8sSecret(secret, namespace)
	if err != nil {
		return err
	} else {
		target.Data = k8sSecret.Data
		target.Type = k8sSecret.Type
		return cli.Update(context.TODO(), target)
	}
}

func deleteSecret(cli client.Client, namespace, name string) error {
	k8sSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), k8sSecret)
}

func scSecretToK8sSecret(secret *types.Secret, namespace string) (*corev1.Secret, error) {
	data := make(map[string][]byte)
	for _, s := range secret.Data {
		if _, ok := data[s.Key]; ok {
			return nil, ErrDuplicateKeyInSecret
		}
		data[s.Key] = []byte(s.Value)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secret.Name, Namespace: namespace},
		Data:       data,
		Type:       corev1.SecretTypeOpaque,
	}, nil
}

func k8sSecretToSCSecret(k8sSecret *corev1.Secret) *types.Secret {
	var data []types.SecretData
	for k, v := range k8sSecret.Data {
		data = append(data, types.SecretData{
			Key:   k,
			Value: string(v),
		})
	}

	secret := &types.Secret{
		Name: k8sSecret.Name,
		Data: data,
	}
	secret.SetID(k8sSecret.Name)
	secret.SetCreationTimestamp(k8sSecret.CreationTimestamp.Time)
	if k8sSecret.GetDeletionTimestamp() != nil {
		secret.SetDeletionTimestamp(k8sSecret.DeletionTimestamp.Time)
	}
	return secret
}
