package types

import (
	"github.com/zdnscloud/gorest/resource"
	"github.com/gsmlg-opt/gaocloud/pkg/charts"
)

type Chart struct {
	resource.ResourceBase `json:",inline"`
	Name                  string         `json:"name"`
	Description           string         `json:"description"`
	Icon                  string         `json:"icon"`
	Dir                   string         `json:"-"`
	Versions              []ChartVersion `json:"versions"`
}

type ChartVersion struct {
	Version string              `json:"version"`
	Config  charts.ChartConfigs `json:"config,omitempty"`
}

type Charts []*Chart

func (c Charts) Len() int {
	return len(c)
}

func (c Charts) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c Charts) Less(i, j int) bool {
	return c[i].Name < c[j].Name
}

func (c Chart) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}
