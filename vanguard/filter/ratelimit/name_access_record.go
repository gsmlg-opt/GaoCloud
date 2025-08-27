package ratelimit

import (
	"time"
)

type NameMatchType int

const (
	exactMatch NameMatchType = 0
	zoneMatch                = 1
)

type NameAccessRecord struct {
	token      uint32
	limit      uint32
	start      time.Time
	match_type NameMatchType
}

func newNameAccessRecord(match_type NameMatchType, limit uint32) *NameAccessRecord {
	r := &NameAccessRecord{
		limit:      limit,
		match_type: match_type,
	}
	return r
}

func (r *NameAccessRecord) Init() {
	r.token = 1
	r.start = time.Now()
}

func (r *NameAccessRecord) Expired() bool {
	return time.Now().Sub(r.start) > time.Second
}

func (r *NameAccessRecord) IsAccessAllowed() bool {
	if r.start.IsZero() {
		r.start = time.Now()
	}

	if r.Expired() {
		r.Init()
		return true
	}

	if r.token+1 > r.limit {
		return false
	} else {
		r.token += 1
		return true
	}
}
