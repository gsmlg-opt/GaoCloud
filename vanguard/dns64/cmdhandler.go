package dns64

import (
	"strings"

	"vanguard/httpcmd"
)

type PutDns64 struct {
	View            string   `json:"view"`
	PreAndPostfixes []string `json:"pre_and_post_fixes"`
}

func (c *PutDns64) String() string {
	return "name: update view dns64 and params: {view:" + c.View +
		", dns64s:[" + strings.Join(c.PreAndPostfixes, ",") + "]}"
}

func (d *DNS64) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *PutDns64:
		return nil, d.updateDns64(c.View, c.PreAndPostfixes)
	default:
		panic("should not be here")
	}
}

func (d *DNS64) updateDns64(view string, preAndPostfixes []string) *httpcmd.Error {
	var converters []*Dns64Converter
	for _, preAndPostfixe := range preAndPostfixes {
		converter, err := converterFromString(preAndPostfixe)
		if err != nil {
			return err.(*httpcmd.Error)
		}
		converters = append(converters, converter)
	}

	d.lock.Lock()
	if len(converters) > 0 {
		d.converters[view] = converters
	} else {
		delete(d.converters, view)
	}
	d.lock.Unlock()
	return nil
}
