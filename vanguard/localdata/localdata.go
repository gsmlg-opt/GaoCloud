package localdata

import (
	"sync"

	"cement/domaintree"
	"g53"
	"vanguard/core"
	"vanguard/httpcmd"
)

type LocalData struct {
	policies map[string]*domaintree.DomainTree
	lock     sync.RWMutex
}

func NewLocalData() *LocalData {
	return &LocalData{
		policies: make(map[string]*domaintree.DomainTree),
	}
}

func (l *LocalData) ResponseWithLocalData(client *core.Client) bool {
	ps, ok := l.policies[client.View]
	if ok == false {
		return false
	}

	l.lock.RLock()
	defer l.lock.RUnlock()

	qname := realQueryName(client)
	parents, match := ps.SearchParents(qname)
	var resp *g53.Message
	if match == domaintree.ExactMatch {
		policy := parents.Top().Data().(LocalPolicy)
		resp = policy.GetMessage(client, qname)
	} else if match == domaintree.ClosestEncloser {
		for parents.IsEmpty() == false {
			parentPolicy := parents.Top().Data().(LocalPolicy)
			if parentPolicy.IsZoneMatch() {
				if msg := parentPolicy.GetMessage(client, qname); msg != nil {
					resp = msg
					break
				}
			}
			parents.Pop()
		}
	}

	if resp != nil {
		client.Response = resp
	}

	return resp != nil
}

func (l *LocalData) AddPolicies(view string, typ LocalPolicyType, datas []string) *httpcmd.Error {
	l.lock.Lock()
	defer l.lock.Unlock()

	for _, data := range datas {
		if p, err := BuildLocalPolicy(typ, data); err != nil {
			return ErrInvalidPolicy.AddDetail(err.Error())
		} else if err := l.addPolicy(view, p); err != nil {
			return err
		}
	}
	return nil
}

func (l *LocalData) addPolicy(view string, p LocalPolicy) *httpcmd.Error {
	ps, ok := l.policies[view]
	if ok == false {
		ps = domaintree.NewDomainTree()
		l.policies[view] = ps
	}

	parents, match := ps.SearchParents(p.GetName())
	if match != domaintree.ExactMatch {
		ps.Insert(p.GetName(), p)
		return nil
	}

	oldp := parents.Top().Data().(LocalPolicy)
	redirect, ok := oldp.(*LocalRRset)
	if ok == false {
		return ErrDuplicatePolicy
	}
	newRedirect, ok := p.(*LocalRRset)
	if ok == false {
		return ErrDuplicatePolicy
	}

	for _, rrset := range newRedirect.rrsets {
		if err := redirect.addRRset(rrset); err != nil {
			return ErrDuplicatePolicy.AddDetail(err.Error())
		}
	}
	return nil
}

func (l *LocalData) RemovePolicies(view string, typ LocalPolicyType, datas []string) *httpcmd.Error {
	l.lock.Lock()
	defer l.lock.Unlock()

	for _, data := range datas {
		if p, err := BuildLocalPolicy(typ, data); err != nil {
			return ErrInvalidPolicy.AddDetail(err.Error())
		} else if err := l.removePolicy(view, p); err != nil {
			return err
		}
	}
	return nil
}

func (l *LocalData) removePolicy(view string, p LocalPolicy) *httpcmd.Error {
	ps, ok := l.policies[view]
	if ok == false {
		return httpcmd.ErrUnknownView
	}

	parents, match := ps.SearchParents(p.GetName())
	if match != domaintree.ExactMatch {
		return nil
	}

	oldp := parents.Top().Data().(LocalPolicy)
	target, ok := oldp.(*LocalRRset)
	if ok == false {
		ps.Delete(p.GetName())
		return nil
	}

	redirectToRemove, ok := p.(*LocalRRset)
	if ok == false {
		return nil
	}

	for _, rrset := range redirectToRemove.rrsets {
		target.deleteRRset(rrset)
	}

	if target.isEmpty() {
		ps.Delete(p.GetName())
	}

	return nil
}
