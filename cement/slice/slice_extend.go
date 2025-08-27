package slice

import (
	"math/rand"
)

func SliceDifference(s1, s2 []string) []string {
	exists := make(map[string]bool)
	for _, v := range s1 {
		exists[v] = true
	}

	for _, v := range s2 {
		exists[v] = false
	}

	result := []string{}
	for v, exist := range exists {
		if exist {
			result = append(result, v)
		}
	}

	return result
}

func SliceIntersection(s1, s2 []string) []string {
	exists := make(map[string]int)
	for _, v := range s1 {
		exists[v] = 1
	}

	for _, v := range s2 {
		if exists[v] == 1 {
			exists[v] = 2
		}
	}

	result := []string{}
	for v, s := range exists {
		if s == 2 {
			result = append(result, v)
		}
	}

	return result
}

func SliceIndex(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
}

func SliceRemoveAt(slice *[]string, i int) {
	if i >= len(*slice) {
		panic("remove index exceed range")
	}

	copy((*slice)[i:], (*slice)[i+1:])
	(*slice)[len(*slice)-1] = ""
	*slice = (*slice)[:len(*slice)-1]
}

func SliceRemove(slice *[]string, s string) bool {
	for i, v := range *slice {
		if v == s {
			SliceRemoveAt(slice, i)
			return true
		}
	}
	return false
}

func RandElem(slice []string) string {
	return slice[rand.Intn(len(slice))]
}

func Shuffle(slice []string) {
	for i := range slice {
		j := rand.Intn(i + 1)
		slice[i], slice[j] = slice[j], slice[i]
	}
}

func Clone(src []string) []string {
	c := len(src)
	if c == 0 {
		return nil
	}

	dst := make([]string, c)
	copy(dst, src)
	return dst
}
