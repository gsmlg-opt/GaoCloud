package ratelimit

import (
	"container/list"
	"encoding/binary"
	"net"
	"time"
)

type V6Key [16]byte

func ipToV6Key(ip net.IP) V6Key {
	var arr V6Key
	copy(arr[:], ip.To16())
	return arr
}

type IPAccessRecord struct {
	ip    net.IP
	token uint32
	start time.Time
}

func newIPAccessRecord(ip net.IP) *IPAccessRecord {
	r := &IPAccessRecord{ip: ip}
	r.Init()
	return r
}

func (r *IPAccessRecord) Init() {
	r.token = 1
	r.start = time.Now()
}

func (r *IPAccessRecord) IsAccessAllowed(limit uint32) bool {
	if r.Expired() {
		r.Init()
		return true
	}

	if r.token+1 > limit {
		return false
	}

	r.token += 1
	return true
}

func (r *IPAccessRecord) Expired() bool {
	return time.Now().Sub(r.start) > time.Second
}

func ipToUint32(ip net.IP) uint32 {
	return binary.LittleEndian.Uint32(ip.To4())
}

type listEntry struct {
	record *IPAccessRecord
}

type IPAccessRecordStore struct {
	MaxCount int

	ll        *list.List
	v4Records map[uint32]*list.Element
	v6Records map[V6Key]*list.Element
}

func newIPAccessRecordStore(maxCount int) *IPAccessRecordStore {
	return &IPAccessRecordStore{
		MaxCount:  maxCount,
		ll:        list.New(),
		v4Records: make(map[uint32]*list.Element),
		v6Records: make(map[V6Key]*list.Element),
	}
}

func (c *IPAccessRecordStore) Add(r *IPAccessRecord) {
	entry := &listEntry{r}
	if elem, hit := c.getElem(r.ip); hit {
		c.ll.MoveToFront(elem)
		elem.Value = entry
	} else {
		elem := c.ll.PushFront(entry)
		if r.ip.To4() != nil {
			c.v4Records[ipToUint32(r.ip)] = elem
		} else {
			c.v6Records[ipToV6Key(r.ip)] = elem
		}

		if c.MaxCount != 0 && c.ll.Len() > c.MaxCount {
			c.removeOldest()
		}
	}
}

func (c *IPAccessRecordStore) Get(ip net.IP) (*IPAccessRecord, bool) {
	if elem, hit := c.getElem(ip); hit {
		c.ll.MoveToFront(elem)
		return elem.Value.(*listEntry).record, true
	}

	return nil, false
}

func (c *IPAccessRecordStore) getElem(ip net.IP) (elem *list.Element, hit bool) {
	if ip.To4() != nil {
		elem, hit = c.v4Records[ipToUint32(ip)]
	} else {
		elem, hit = c.v6Records[ipToV6Key(ip)]
	}

	return
}

func (c *IPAccessRecordStore) Remove(ip net.IP) {
	if elem, hit := c.getElem(ip); hit {
		c.removeElement(elem)
	}
}

func (c *IPAccessRecordStore) removeOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.removeElement(ele)
	}
}

func (c *IPAccessRecordStore) removeElement(e *list.Element) {
	c.ll.Remove(e)
	r := e.Value.(*listEntry).record
	if r.ip.To4() != nil {
		delete(c.v4Records, ipToUint32(r.ip))
	} else {
		delete(c.v6Records, ipToV6Key(r.ip))
	}
}

func (c *IPAccessRecordStore) Len() int {
	return c.ll.Len()
}
