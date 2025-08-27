package slice

import (
	"sort"
)

type Item interface {
	Less(other Item) bool
}

type SortedSlice []Item

func (s *SortedSlice) InsertAt(index int, item Item) {
	*s = append(*s, nil)
	if index < len(*s) {
		copy((*s)[index+1:], (*s)[index:])
	}
	(*s)[index] = item
}

// removeAt removes a value at a given index, pulling all subsequent values
// back.
func (s *SortedSlice) RemoveAt(index int) Item {
	item := (*s)[index]
	copy((*s)[index:], (*s)[index+1:])
	(*s)[len(*s)-1] = nil
	*s = (*s)[:len(*s)-1]
	return item
}

// pop removes and returns the last element in the list.
func (s *SortedSlice) Pop() (out Item) {
	index := len(*s) - 1
	out = (*s)[index]
	(*s)[index] = nil
	*s = (*s)[:index]
	return
}

// truncate truncates this instance at index so that it contains only the
// first index SortedSlice. index must be less than or equal to length.
func (s *SortedSlice) Truncate(index int) {
	var toClear SortedSlice
	*s, toClear = (*s)[:index], (*s)[index:]
	for i := 0; i < len(toClear); i++ {
		toClear[i] = nil
	}
}

// find returns the index where the given item should be inserted into this
// list.  'found' is true if the item already exists in the list at the given
// index.
func (s SortedSlice) Find(item Item) (index int, found bool) {
	i := sort.Search(len(s), func(i int) bool {
		return item.Less(s[i])
	})
	if i > 0 && !s[i-1].Less(item) {
		return i - 1, true
	}
	return i, false
}
