package kvzoo

import (
	"testing"

	ut "cement/unittest"
)

func TestTableName(t *testing.T) {
	invalidNames := []string{
		"xxx",
		"/",
		"/xxx//",
		"/xxxx//xx/",
		"/xxxx/xx/",
		"/1/2/3/4/5/6/7/8/9/10/11/12",
	}
	for _, n := range invalidNames {
		_, err := NewTableName(n)
		ut.Assert(t, err != nil, "")
	}

	validNames := []string{
		"/xxx",
		"/xxx/dd",
		"/xxxx/dd/xx",
	}
	for _, n := range validNames {
		_, err := NewTableName(n)
		ut.Assert(t, err == nil, "")
	}

	tn, err := TableNameFromSegments("a", "b")
	ut.Assert(t, err == nil, "")
	ut.Equal(t, string(tn), "/a/b")

	invalidSegments := [][]string{
		[]string{"", "a"},
		[]string{"c", "", "a"},
		[]string{"1", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1", "1"},
		[]string{"a", "b/c"},
		[]string{"/a", "bc"},
	}
	for _, segs := range invalidSegments {
		_, err := TableNameFromSegments(segs...)
		ut.Assert(t, err != nil, "")
	}

	nameAndSegs := map[string][]string{
		"/a/b/c": []string{"a", "b", "c"},
		"/a/b":   []string{"a", "b"},
		"/a":     []string{"a"},
	}

	for k, v := range nameAndSegs {
		tn, err = NewTableName(k)
		ut.Assert(t, err == nil, "")
		ut.Equal(t, v, tn.Segments())
	}

	grandParent, _ := NewTableName("/a")
	parent, _ := NewTableName("/a/b")
	child, _ := NewTableName("/a/b/c")
	ut.Assert(t, parent.IsParent(child), "")
	ut.Assert(t, grandParent.IsParent(parent), "")
	ut.Assert(t, grandParent.IsParent(child), "")
	p, _ := child.Parent()
	ut.Equal(t, p, parent)
	p, _ = parent.Parent()
	ut.Equal(t, p, grandParent)
	_, err = grandParent.Parent()
	ut.Assert(t, err != nil, "")
}
