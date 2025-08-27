package querysource

import (
	"fmt"

	"vanguard/httpcmd"
)

const DefaultViewForQuery = "*"

type AddQuerySource struct {
	View        string `json:"view"`
	QuerySource string `json:"query_source"`
}

func (c *AddQuerySource) String() string {
	return fmt.Sprintf("name: add query source and params:{view:%s, query_source:%s}", c.View, c.QuerySource)
}

type DeleteQuerySource struct {
	View string `json:"view"`
}

func (c *DeleteQuerySource) String() string {
	return fmt.Sprintf("name: delete query source and params:{view:%s}", c.View)
}

type UpdateQuerySource struct {
	View        string `json:"view"`
	QuerySource string `json:"query_source"`
}

func (c *UpdateQuerySource) String() string {
	return fmt.Sprintf("name: update query source and params:{view:%s, query_source:%s}", c.View, c.QuerySource)
}

func (q *QuerySourceManager) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddQuerySource:
		return nil, q.addQuerySource(c.View, c.QuerySource)
	case *DeleteQuerySource:
		return nil, q.deleteQuerySource(c.View)
	case *UpdateQuerySource:
		return nil, q.updateQuerySource(c.View, c.QuerySource)
	default:
		panic("should not be here")
	}
}

func (q *QuerySourceManager) addQuerySource(view, addr string) *httpcmd.Error {
	q.lock.Lock()
	defer q.lock.Unlock()
	if _, ok := q.querySources[view]; ok {
		return ErrDuplicateQuerySource
	} else {
		q.querySources[view] = addr
		return nil
	}
}

func (q *QuerySourceManager) deleteQuerySource(view string) *httpcmd.Error {
	q.lock.Lock()
	defer q.lock.Unlock()
	if _, ok := q.querySources[view]; ok == false {
		return ErrNotExistQuerySource
	} else {
		delete(q.querySources, view)
		return nil
	}
}

func (q *QuerySourceManager) updateQuerySource(view, addr string) *httpcmd.Error {
	q.lock.Lock()
	defer q.lock.Unlock()
	if _, ok := q.querySources[view]; ok == false {
		return ErrNotExistQuerySource
	} else {
		q.querySources[view] = addr
		return nil
	}
}
