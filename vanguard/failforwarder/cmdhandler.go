package failforwarder

import (
	"fmt"

	"vanguard/httpcmd"
)

type AddFailForwarder struct {
	View      string `json:"view"`
	Forwarder string `json:"forwarder"`
}

func (c *AddFailForwarder) String() string {
	return fmt.Sprintf("add fail forwarder and params:{view:%s, forwarder:%s}", c.View, c.Forwarder)
}

type UpdateFailForwarder struct {
	View      string `json:"view"`
	Forwarder string `json:"forwarder"`
}

func (c *UpdateFailForwarder) String() string {
	return fmt.Sprintf("update fail forwarder params:{view:%s, forwarder:%s}", c.View, c.Forwarder)
}

type DeleteFailForwarder struct {
	View string `json:"view"`
}

func (c *DeleteFailForwarder) String() string {
	return fmt.Sprintf("delete fail forwarder params:{view:%s}", c.View)
}

func (ff *FailForwarder) HandleCmd(cmd httpcmd.Command) (interface{}, *httpcmd.Error) {
	switch c := cmd.(type) {
	case *AddFailForwarder:
		return nil, ff.AddForwarder(c.View, c.Forwarder)
	case *DeleteFailForwarder:
		return nil, ff.DeleteForwarder(c.View)
	case *UpdateFailForwarder:
		return nil, ff.UpdateForwarder(c.View, c.Forwarder)
	default:
		panic("should not be here")
	}
}
