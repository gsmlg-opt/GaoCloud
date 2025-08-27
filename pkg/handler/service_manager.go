package handler

import (
	"context"
	"fmt"
	"net"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/gsmlg-opt/gaocloud/pkg/eventbus"
	"github.com/gsmlg-opt/gaocloud/pkg/types"
)

const (
	NoneClusterIP = "None"

	ZcloudLBVIPAnnotationKey    = "lb.zcloud.cn/vip"
	ZcloudLBMethodAnnotationKey = "lb.zcloud.cn/method"
	ZcloudDefaultLBMethod       = "rr"
)

type ServiceManager struct {
	clusters *ClusterManager
}

func newServiceManager(clusters *ClusterManager) *ServiceManager {
	return &ServiceManager{clusters: clusters}
}

func (m *ServiceManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	service := ctx.Resource.(*types.Service)

	if err := validateIfLoadBalancerService(service); err != nil {
		return nil, resterror.NewAPIError(resterror.PermissionDenied, err.Error())
	}

	err := createService(cluster.GetKubeClient(), namespace, service)
	if err == nil {
		service.SetID(service.Name)
		return service, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resterror.NewAPIError(resterror.DuplicateResource, fmt.Sprintf("duplicate service name %s", service.Name))
	} else {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create service failed %s", err.Error()))
	}
}

func validateIfLoadBalancerService(s *types.Service) error {
	if s.ServiceType != "loadbalancer" {
		return nil
	}

	ip := net.ParseIP(s.LoadBalanceVIP)
	if ip == nil || ip.To4() == nil {
		return fmt.Errorf("svc LoadBalanceVIP must be an ipv4 address")
	}
	return nil
}

func (m *ServiceManager) List(ctx *resource.Context) (interface{}, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	k8sServices, err := getServices(cluster.GetKubeClient(), namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, "no found services")
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list services failed %s", err.Error()))
	}

	var services []*types.Service
	for _, sv := range k8sServices.Items {
		services = append(services, k8sServiceToSCService(&sv))
	}
	return services, nil
}

func (m *ServiceManager) Get(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	service := ctx.Resource.(*types.Service)
	k8sService, err := getService(cluster.GetKubeClient(), namespace, service.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found service %s", service.GetID()))
		}
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get service %s failed %s", service.GetID(), err.Error()))
	}

	return k8sServiceToSCService(k8sService), nil
}

func (m *ServiceManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetID()
	service := ctx.Resource.(*types.Service)
	err := deleteService(cluster.GetKubeClient(), namespace, service.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterror.NewAPIError(resterror.NotFound, fmt.Sprintf("no found service %s", service.GetID()))
		}
		return resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("delete service %s failed %s", service.GetID(), err.Error()))
	} else {
		eventbus.PublishResourceDeleteEvent(service)
	}
	return nil
}

func getService(cli client.Client, namespace, name string) (*corev1.Service, error) {
	service := corev1.Service{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &service)
	return &service, err
}

func getServices(cli client.Client, namespace string) (*corev1.ServiceList, error) {
	services := corev1.ServiceList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &services)
	return &services, err
}

func createService(cli client.Client, namespace string, service *types.Service) error {
	typ, err := scServiceTypeToK8sServiceType(service.ServiceType)
	if err != nil {
		return err
	}

	var ports []corev1.ServicePort
	for _, p := range service.ExposedPorts {
		protocol, err := scPortProtocolToK8SProtocol(p.Protocol)
		if err != nil {
			return err
		}

		ports = append(ports, corev1.ServicePort{
			Name:       p.Name,
			Protocol:   protocol,
			Port:       int32(p.Port),
			TargetPort: p.TargetPort,
		})
	}

	k8sService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: service.Name, Namespace: namespace},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": service.Name},
			Ports:    ports,
			Type:     typ,
		},
	}

	if typ == corev1.ServiceTypeLoadBalancer {
		k8sService.ObjectMeta.Annotations = scServiceToLBK8sServiceAnnotation(service)
	}

	if typ == corev1.ServiceTypeClusterIP && service.Headless {
		k8sService.Spec.ClusterIP = NoneClusterIP
	}
	return cli.Create(context.TODO(), k8sService)
}

func scServiceToLBK8sServiceAnnotation(s *types.Service) map[string]string {
	result := map[string]string{}
	result[ZcloudLBVIPAnnotationKey] = s.LoadBalanceVIP
	result[ZcloudLBMethodAnnotationKey] = ZcloudDefaultLBMethod
	if s.LoadBalanceMethod != "" {
		result[ZcloudLBMethodAnnotationKey] = s.LoadBalanceMethod
	}
	return result
}

func deleteService(cli client.Client, namespace, name string) error {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), service)
}

func k8sServiceToSCService(k8sService *corev1.Service) *types.Service {
	var ports []types.ServicePort
	for _, p := range k8sService.Spec.Ports {
		ports = append(ports, types.ServicePort{
			Name:       p.Name,
			Protocol:   strings.ToLower(string(p.Protocol)),
			Port:       int(p.Port),
			TargetPort: p.TargetPort,
			NodePort:   int(p.NodePort),
		})
	}
	service := &types.Service{
		Name:              k8sService.Name,
		ServiceType:       strings.ToLower(string(k8sService.Spec.Type)),
		ClusterIP:         k8sService.Spec.ClusterIP,
		ExposedPorts:      ports,
		LoadBalanceVIP:    k8sService.GetAnnotations()[ZcloudLBVIPAnnotationKey],
		LoadBalanceMethod: k8sService.GetAnnotations()[ZcloudLBMethodAnnotationKey],
	}
	service.SetID(k8sService.Name)
	service.SetCreationTimestamp(k8sService.CreationTimestamp.Time)
	if k8sService.GetDeletionTimestamp() != nil {
		service.SetDeletionTimestamp(k8sService.DeletionTimestamp.Time)
	}
	return service
}
