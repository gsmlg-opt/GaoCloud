package types

import (
	"gorest/resource"
)

type StorageClass struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
}

func (s StorageClass) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Cluster{}}
}
