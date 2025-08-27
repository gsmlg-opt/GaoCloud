package types

import (
	"k8s.io/apimachinery/pkg/util/intstr"

	"gorest/resource"
)

const (
	PortProtocolUDP = "udp"
	PortProtocolTCP = "tcp"
)

type ServicePort struct {
	Name       string             `json:"name" rest:"required=true,isDomain=true"`
	Port       int                `json:"port" rest:"required=true"`
	TargetPort intstr.IntOrString `json:"targetPort" rest:"required=true"`
	Protocol   string             `json:"protocol" rest:"required=true,options=tcp|udp"`
	NodePort   int                `json:"nodePort,omitempty"`
}

type Service struct {
	resource.ResourceBase `json:",inline"`
	Name                  string        `json:"name" rest:"required=true,isDomain=true"`
	ServiceType           string        `json:"serviceType" rest:"required=true,options=clusterip|nodeport|loadbalancer"`
	Headless              bool          `json:"headless"`
	ClusterIP             string        `json:"clusterIP,omitempty"`
	ExposedPorts          []ServicePort `json:"exposedPorts" rest:"required=true"`
	LoadBalanceVIP        string        `json:"loadBalanceVip"`
	LoadBalanceMethod     string        `json:"loadBalanceMethod" rest:"options=rr|lc|hash"`
}

func (s Service) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

func (s Service) SupportAsyncDelete() bool {
	return true
}
