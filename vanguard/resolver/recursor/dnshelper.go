package recursor

import (
	"errors"

	"g53"
)

var errNameServerIsOutOfQuery = errors.New("name server isn't parent of query name")
var errAuthSectionIsNotValid = errors.New("auth section should has one ns rrset")

func getAuthAndGlues(zone *g53.Name, msg *g53.Message) (*g53.RRset, []*g53.RRset, error) {
	var auth g53.Section
	if msg.Question.Type == g53.RR_NS && len(msg.Sections[g53.AnswerSection]) == 1 {
		auth = msg.Sections[g53.AnswerSection]
	} else {
		auth = msg.Sections[g53.AuthSection]
		if len(auth) != 1 {
			return nil, nil, errAuthSectionIsNotValid
		}
	}

	nsRRset := auth[0]
	if nsRRset.Type != g53.RR_NS {
		return nil, nil, errAuthSectionIsNotValid
	}

	glues := msg.Sections[g53.AdditionalSection]
	validGlues := []*g53.RRset{}
	for _, rdata := range nsRRset.Rdatas {
		nameServerName := rdata.(*g53.NS).Name
		validGlue := &g53.RRset{
			Name:  nameServerName,
			Type:  g53.RR_A,
			Class: g53.CLASS_IN,
		}
		for _, glue := range glues {
			if glue.Type == g53.RR_A && glue.Name.Equals(nameServerName) {
				validGlue.Rdatas = append(validGlue.Rdatas, glue.Rdatas...)
				validGlue.Ttl = glue.Ttl
			}
		}
		if len(validGlue.Rdatas) > 0 {
			validGlues = append(validGlues, validGlue)
		}
	}

	return nsRRset, validGlues, nil
}

func isValidResponse(msg *g53.Message) bool {
	return msg.Header.Rcode == g53.R_NOERROR || msg.Header.Rcode == g53.R_NXDOMAIN
}

//ns glue shouldn't have cname
func getARRsetFromAnswer(msg *g53.Message) *g53.RRset {
	answer := msg.Sections[g53.AnswerSection]
	if len(answer) == 1 && answer[0].Type == g53.RR_A && answer[0].Name.Equals(msg.Question.Name) {
		return answer[0]
	} else {
		return nil
	}
}
