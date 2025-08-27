package localdata

import (
	"g53"

	"vanguard/core"
)

type LocalPolicy interface {
	GetName() *g53.Name
	GetMessage(*core.Client, *g53.Name) *g53.Message
	IsZoneMatch() bool
}

type basePolicy struct {
	name        *g53.Name
	isZoneMatch bool
}

func (b *basePolicy) GetName() *g53.Name {
	return b.name
}

func (b *basePolicy) IsZoneMatch() bool {
	return b.isZoneMatch
}

type NXDomain struct {
	*basePolicy
}

func (p *NXDomain) GetMessage(cli *core.Client, qname *g53.Name) *g53.Message {
	msg := cli.Request.MakeResponse()
	msg.Header.Rcode = g53.R_NXDOMAIN
	return msg
}

type NXRRset struct {
	*basePolicy
}

func (p *NXRRset) GetMessage(cli *core.Client, qname *g53.Name) *g53.Message {
	msg := cli.Request.MakeResponse()
	msg.Header.Rcode = g53.R_NOERROR
	return msg
}

type ExceptionDomain struct {
	*basePolicy
}

func (p *ExceptionDomain) GetMessage(cli *core.Client, qname *g53.Name) *g53.Message {
	return nil
}

type LocalRRset struct {
	*basePolicy
	rrsets []*g53.RRset
}

func newLocalRRset(base *basePolicy, typ string, ttl int, rdataStr string) (*LocalRRset, error) {
	rrType, err := g53.TypeFromString(typ)
	if err != nil {
		return nil, err
	}

	rdata, err := g53.RdataFromString(rrType, rdataStr)
	if err != nil {
		return nil, err
	}

	newRRset := &g53.RRset{
		Name:   base.name,
		Type:   rrType,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(ttl),
		Rdatas: []g53.Rdata{rdata},
	}

	return &LocalRRset{
		basePolicy: base,
		rrsets:     []*g53.RRset{newRRset},
	}, nil
}

func (p *LocalRRset) GetMessage(cli *core.Client, qname *g53.Name) *g53.Message {
	var msg *g53.Message
	if cli.Response != nil {
		msg = cli.Response
	} else {
		msg = cli.Request.MakeResponse()
	}

	answers := msg.Sections[g53.AnswerSection]
	for _, rrset := range p.rrsets {
		if rrset.Type == cli.Request.Question.Type {
			msg.Header.ANCount = uint16(rrset.RRCount())
			answers = append(answers, &g53.RRset{
				Name:   qname,
				Type:   rrset.Type,
				Class:  rrset.Class,
				Ttl:    rrset.Ttl,
				Rdatas: rrset.Rdatas,
			})
			msg.Sections[g53.AnswerSection] = answers
			msg.Header.Rcode = g53.R_NOERROR
			return msg
		}
	}
	return nil
}

func (p *LocalRRset) addRRset(new *g53.RRset) error {
	hasRRset := false
	for _, old := range p.rrsets {
		if old.Type == new.Type {
			hasRRset = true
			for _, rdata := range new.Rdatas {
				if err := old.AddRdata(rdata); err != nil {
					return err
				}
			}
			old.Ttl = new.Ttl
			break
		}
	}

	if hasRRset == false {
		p.rrsets = append(p.rrsets, new)
	}
	return nil
}

func (p *LocalRRset) deleteRRset(new *g53.RRset) {
	for i, old := range p.rrsets {
		if old.Type == new.Type {
			for _, rdata := range new.Rdatas {
				old.RemoveRdata(rdata)
			}

			if old.RRCount() == 0 {
				p.rrsets = append(p.rrsets[:i], p.rrsets[i+1:]...)
			}
		}
	}
}

func (p *LocalRRset) isEmpty() bool {
	return len(p.rrsets) == 0
}
