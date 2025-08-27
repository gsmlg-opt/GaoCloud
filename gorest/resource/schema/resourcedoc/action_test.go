package resourcedoc

import (
	"gorest/resource"
	"testing"

	ut "cement/unittest"
)

const (
	ActionLogin    = "actionLogin"
	ActionOnlyName = "onlyName"
)

type Action struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
	ID                    int    `json:"id"`
}

type ActionErr struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name"`
	ID                    int    `json:"id"`
}

type UserPassword struct {
	SliceStructPtr  []*Struct         `json:"sliceStructPtr"`
	MapStringInt8   map[string]int8   `json:"mapStringInt8"`
	MapStringStruct map[string]Struct `json:"mapStringStruct"`
	StructPtr       *Struct           `json:"structPtr"`
}

/*
type Struct struct {
	Name string
	Id   int
	Str  Struct1
}
type Struct1 struct {
	Name string
	Str  Struct2
}
type Struct2 struct {
	Id int
}*/

type LoginInfo struct {
	Uint32          uint32            `json:"uint32"`
	MapStringString map[string]string `json:"mapStringString"`
	MapStringInt    map[string]int    `json:"mapStringInt"`
	BoolPtr         *bool             `json:"boolPtr"`
}

func (a Action) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name:   ActionLogin,
			Input:  &UserPassword{},
			Output: &LoginInfo{},
		},
		resource.Action{
			Name: ActionOnlyName,
		},
	}
}

func (a ActionErr) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name:  "test-string",
			Input: "string",
		},
	}
}

func TestIngressRuleValidate(t *testing.T) {
	cases := []struct {
		kind            resource.ResourceKind
		isValid         bool
		resourceActions []ResourceAction
		actionsNum      int
	}{
		{
			Action{},
			true,
			[]ResourceAction{
				ResourceAction{
					Name: "actionLogin",
					Input: ResourceFields{
						"mapStringInt8":   ResourceField{Type: "map", KeyType: "string", ValueType: "int"},
						"mapStringStruct": ResourceField{Type: "map", KeyType: "string", ValueType: "struct"},
						"structPtr":       ResourceField{Type: "struct"},
						"sliceStructPtr":  ResourceField{Type: "array", ElemType: "struct"},
					},
					Output: ResourceFields{
						"uint32":          ResourceField{Type: "uint"},
						"mapStringString": ResourceField{Type: "map", KeyType: "string", ValueType: "string"},
						"mapStringInt":    ResourceField{Type: "map", KeyType: "string", ValueType: "int"},
						"boolPtr":         ResourceField{Type: "bool"},
					},
					SubResources: map[string]ResourceFields{
						"struct2": map[string]ResourceField{
							"Id": ResourceField{Type: "int"},
						},
						"struct1": map[string]ResourceField{
							"Name": ResourceField{Type: "string"},
							"Str":  ResourceField{Type: "struct2"},
						},
						"struct": map[string]ResourceField{
							"Name": ResourceField{Type: "string"},
							"Id":   ResourceField{Type: "int"},
							"Str":  ResourceField{Type: "struct1"},
						},
					},
				},
				ResourceAction{
					Name:         "onlyName",
					SubResources: map[string]ResourceFields{},
				},
			},
			2,
		},
		{
			ActionErr{},
			false,
			[]ResourceAction{},
			0,
		},
	}
	for _, k := range cases {
		as, err := genActions(k.kind)
		if !k.isValid {
			ut.Assert(t, err != nil, "should err for case %s, but get nothing", k.kind)
		} else {
			ut.Assert(t, err == nil, "should ok but get %v", err)
			ut.Equal(t, len(as), k.actionsNum)
			for i, a := range as {
				ut.Equal(t, a, k.resourceActions[i])
			}
		}
	}
}
