package types

import (
	"gorest/resource"
)

type KubeConfig struct {
	resource.ResourceBase `json:",inline"`
	User                  string `json:"user" rest:"required=true"`
	KubeConfig            string `json:"kubeConfig"`
}

func (k KubeConfig) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
