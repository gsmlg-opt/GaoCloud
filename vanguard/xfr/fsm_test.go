package xfr

import (
	ut "cement/unittest"
	"testing"

	"g53"
	"vanguard/logger"
	"vanguard/resolver/auth/zone"
)

type dumbTx struct {
	commited bool
}

func (tx *dumbTx) RollBack() error {
	tx.commited = false
	return nil
}

func (tx *dumbTx) Commit() error {
	tx.commited = true
	return nil
}

type dumpUpdator struct {
	serialIncreaseCount int
	addedRRsetCount     int
}

func (u *dumpUpdator) Begin() (zone.Transaction, error) {
	return &dumbTx{}, nil
}

func (u *dumpUpdator) Add(zone.Transaction, *g53.RRset) error {
	u.addedRRsetCount += 1
	return nil
}
func (u *dumpUpdator) DeleteRRset(zone.Transaction, *g53.RRset) error {
	return nil
}
func (u *dumpUpdator) DeleteDomain(zone.Transaction, *g53.Name) error {
	return nil
}
func (u *dumpUpdator) DeleteRr(zone.Transaction, *g53.RRset) error {
	return nil
}
func (u *dumpUpdator) IncreaseSerialNumber(zone.Transaction) {
	u.serialIncreaseCount += 1
}
func (u *dumpUpdator) Clean() error {
	return nil
}

func rrsetFromString(str string) *g53.RRset {
	rrset, err := g53.RRsetFromString(str)
	if err != nil {
		panic("invalid rrset:" + str + " err:" + err.Error())
	}
	return rrset
}

func buildXFRAnswer(strs []string) g53.Section {
	xfrAnswers := make([]*g53.RRset, len(strs))
	for i, s := range strs {
		xfrAnswers[i] = rrsetFromString(s)
	}
	return xfrAnswers
}

func TestXFRStateMachine(t *testing.T) {
	logger.UseDefaultLogger("debug")

	updator := &dumpUpdator{}
	tx, _ := updator.Begin()
	sm := newFSMGenerator(IXFR, uint32(2002022437), uint32(2002022439), updator, tx).GenStateMachine()

	xfrAnswers := buildXFRAnswer([]string{
		"example.com.	86400	IN	SOA	ns1.example.com. hostmaster.example.com. 2002022439 10800 15 604800 10800",
		"example.com.	86400	IN	SOA	ns1.example.com. hostmaster.example.com. 2002022437 10800 15 604800 10800",
		"ben.example.com.	3600	IN	A	2.2.2.2",
		"ben.example.com.	3600	IN	A	3.3.3.3",
		"example.com.	86400	IN	SOA	ns1.example.com. hostmaster.example.com. 2002022438 10800 15 604800 10800",
		"example.com.	86400	IN	SOA	ns1.example.com. hostmaster.example.com. 2002022438 10800 15 604800 10800",
		"example.com.	86400	IN	SOA	ns1.example.com. hostmaster.example.com. 2002022439 10800 15 604800 10800",
		"ben.example.com.	3600	IN	A	3.3.3.3",
		"ben.example.com.	3600	IN	A	2.2.2.2",
		"example.com.	86400	IN	SOA	ns1.example.com. hostmaster.example.com. 2002022439 10800 15 604800 10800",
	})

	err := sm.Run(xfrAnswers)
	ut.Assert(t, err == nil, "fsm should ok but get %v\n", err)
	ut.Equal(t, updator.serialIncreaseCount, 2)
}

func TestXFRStateMachineHandleAXFR(t *testing.T) {
	logger.UseDefaultLogger("debug")

	updator := &dumpUpdator{}
	tx, _ := updator.Begin()
	sm := newFSMGenerator(AXFR, 0, uint32(2002022401), updator, tx).GenStateMachine()

	axfrAnswers := buildXFRAnswer([]string{
		"example.com.		86400	IN	SOA	ns1.example.com. hostmaster.example.com. 2002022401 10800 15 604800 10800",
		"example.com.		86400	IN	NS	ns1.example.com.",
		"example.com.		86400	IN	NS	ns2.smokeyjoe.com.",
		"example.com.		86400	IN	MX	10 mail.another.com.",
		"bill.example.com.	86400	IN	A	192.168.0.3",
		"fred.example.com.	86400	IN	A	192.168.0.4",
		"ftp.example.com.	86400	IN	CNAME	www.example.com.",
		"ns1.example.com.	86400	IN	A	192.168.0.1",
		"www.example.com.	86400	IN	A	192.168.0.2",
		"example.com.		86400	IN	SOA	ns1.example.com. hostmaster.example.com. 2002022401 10800 15 604800 10800",
	})

	err := sm.Run(axfrAnswers)
	ut.Assert(t, err == nil, "fsm should ok but get %v\n", err)
	ut.Equal(t, updator.serialIncreaseCount, 0)
	ut.Equal(t, updator.addedRRsetCount, 9)
}
