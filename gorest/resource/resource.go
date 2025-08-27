package resource

import (
	"reflect"
	"strings"
	"time"

	"gorest/util"
)

type ResourceLinkType string
type ResourceLink string //url schema + host + path

const (
	SelfLink       ResourceLinkType = "self"
	UpdateLink     ResourceLinkType = "update"
	RemoveLink     ResourceLinkType = "remove"
	CollectionLink ResourceLinkType = "collection"
)

type Resource interface {
	GetParent() Resource
	SetParent(Resource)

	GetID() string
	SetID(string)

	GetLinks() map[ResourceLinkType]ResourceLink
	SetLinks(map[ResourceLinkType]ResourceLink)

	GetCreationTimestamp() time.Time
	SetCreationTimestamp(time.Time)

	GetDeletionTimestamp() time.Time
	SetDeletionTimestamp(time.Time)

	GetSchema() Schema
	SetSchema(Schema)

	//return kind name like "pod"
	SetType(string)
	GetType() string

	GetAction() *Action
	SetAction(*Action)
}

//struct implement ResourceKind
//struct pointer should implement Resource
type ResourceKind interface {
	GetParents() []ResourceKind
	//return the default resource if the related field
	//isn't speicified in json data
	//NOTE: default field shouldn't include map
	//json unmarshal will merge map, in this case
	//when real data is provided, it will merge with
	//default value
	CreateDefaultResource() Resource
	GetActions() []Action
	SupportAsyncDelete() bool
}

//lowercase singluar
//eg: type Node struct -> node
func DefaultKindName(t interface{}) string {
	typ := reflect.TypeOf(t)
	kind := typ.Kind()
	if kind == reflect.Ptr {
		typ = typ.Elem()
		kind = typ.Kind()
	}
	if kind != reflect.Struct {
		panic("invalid param, it's not a struct or a struct pointer")
	}

	return strings.ToLower(typ.Name())
}

//resource name is lowercase, plural word
//eg: type Node struct -> nodes
func DefaultResourceName(t interface{}) string {
	return util.GuessPluralName(DefaultKindName(t))
}
