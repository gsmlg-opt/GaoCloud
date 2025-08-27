package validator

import (
	"reflect"
	"strings"
	"testing"

	ut "cement/unittest"
)

func TestBuildValidator(t *testing.T) {
	type MyOption string
	type testStruct struct {
		DomainName            string     `json:"domainName" rest:"isDomain=true"`
		StringWithOption      MyOption   `json:"stringWithOption,omitempty" rest:"required=true,options=lvm|ceph"`
		StringWithLenLimit    string     `json:"stringWithLenLimit" rest:"minLen=2,maxLen=10"`
		IntWithRange          uint32     `json:"intWithRange" rest:"min=1,max=1000"`
		StringSliceWithDomain []string   `json:"stringSliceWithDomain,omitempty" rest:"required=true,isDomain=true"`
		StringSliceWithOption []MyOption `json:"stringSliceWithOption,omitempty" rest:"required=true,options=lvm|ceph"`
	}

	st := reflect.TypeOf(testStruct{})
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		tags := strings.Split(f.Tag.Get("rest"), ",")
		ut.Assert(t, len(tags) > 0, "")
		validator, err := Build(f.Type, tags)
		ut.Assert(t, err == nil && validator != nil, "")
	}

	type testStruct2 struct {
		IntWithOption      int  `rest:"required=true,options=lvm|ceph"`
		IntWithLenLimit    int  `rest:"minLen=10,maxLen=11"`
		IntWithDomainCheck int  `rest:"isDomain=true"`
		IntRangeInvalid    int8 `rest:"min=2,max=1"`
		IntWithEmptyRange  int8 `rest:"min="`

		StringWithInvalidLenLimit string `rest:"minLen=12,maxLen=12"`
		StringWithEmptyInterval   string `rest:"minLen=,maxLen="`
	}

	st = reflect.TypeOf(testStruct2{})
	for i := 0; i < st.NumField(); i++ {
		f := st.Field(i)
		tags := strings.Split(f.Tag.Get("rest"), ",")
		validators, _ := Build(f.Type, tags)
		ut.Assert(t, len(validators) == 0, "tag should has error %v", tags)
	}
}

type testCase struct {
	value    interface{}
	isValide bool
}

func TestIntegerRangeValidator(t *testing.T) {
	cases := []testCase{
		{1, true},
		{10, false},
		{11, false},
	}
	testValidator(t, int(10), []string{"min=1", "max=10"}, cases)
	testValidator(t, []uint{10}, []string{"min=1", "max=10"}, cases)

	cases = []testCase{
		{4, true},
		{2, false},
		{1000, true},
	}
	testValidator(t, int(10), []string{"min=4"}, cases)
	testValidator(t, []uint{10}, []string{"min=4"}, cases)

	cases = []testCase{
		{4, true},
		{99, true},
		{1000, false},
	}
	testValidator(t, int(10), []string{"max=1000"}, cases)
	testValidator(t, []uint{10}, []string{"max=1000"}, cases)
}

func TestStringLenValidator(t *testing.T) {
	cases := []testCase{
		{"a", true},
		{"abc", false},
		{"", false},
	}
	testValidator(t, "xxx", []string{"minLen=1", "maxLen=3"}, cases)
	testValidator(t, []string{"xxx"}, []string{"minLen=1", "maxLen=3"}, cases)

	cases = []testCase{
		{"a", false},
		{"ab", true},
		{"abcc", true},
		{"", false},
	}
	testValidator(t, "xxx", []string{"minLen=2"}, cases)
	testValidator(t, []string{"xxx"}, []string{"minLen=2"}, cases)

	cases = []testCase{
		{"", true},
		{"a", true},
		{"abb", false},
		{"abcc", false},
	}
	testValidator(t, "xxx", []string{"maxLen=3"}, cases)
	testValidator(t, []string{"xxx"}, []string{"maxLen=3"}, cases)
}

func TestOptionValidator(t *testing.T) {
	cases := []testCase{
		{"aa", true},
		{"bb", true},
		{"Aa", false},
		{"Aaa", false},
	}
	testValidator(t, "xxx", []string{"options=aa|bb"}, cases)
	testValidator(t, []string{"xxx"}, []string{"options=aa|bb"}, cases)
}

func TestDomainValidator(t *testing.T) {
	cases := []testCase{
		{"aa", true},
		{"11-bb", true},
		{"11_bb", false},
		{"Aaa", false},
		{"-aa", false},
		{"11?aa", false},
	}

	testValidator(t, "xxxx", []string{"isDomain=true"}, cases)
	testValidator(t, []string{"xxxx"}, []string{"isDomain=true"}, cases)
}

func testValidator(t *testing.T, fieldValue interface{}, tags []string, cases []testCase) {
	validators, err := Build(reflect.TypeOf(fieldValue), tags)
	ut.Assert(t, err == nil && len(validators) == 1, "")
	validator := validators[0]
	for i := 0; i < len(cases); i++ {
		err := validator.Validate(cases[i].value)
		if cases[i].isValide {
			ut.Assert(t, err == nil, "")
		} else {
			ut.Assert(t, err != nil, "")
		}
	}
}
