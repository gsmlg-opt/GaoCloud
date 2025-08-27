package viewselector

import (
	"net"
	"testing"

	ut "cement/unittest"
	"vanguard/acl"
	"vanguard/config"
	"vanguard/core"
	"vanguard/httpcmd"
	"vanguard/logger"
)

func TestAddrBaseView(t *testing.T) {
	logger.UseDefaultLogger("error")

	viewSelector := newAddrBasedView()
	var conf = &config.VanguardConf{
		Views: config.ViewConf{
			ViewAcls: []config.ViewAcl{
				config.ViewAcl{
					View: "v1",
					Acls: []string{"a1"},
				},
				config.ViewAcl{
					View: "v2",
					Acls: []string{"a2"},
				},

				config.ViewAcl{
					View: "v3",
					Acls: []string{"a3"},
				},
			},
		},
	}
	viewSelector.ReloadConfig(conf)

	_, err := viewSelector.updateViewPriority([]string{"v3", "v1", "v4"})
	ut.Equal(t, err.Code, httpcmd.ErrUnknownView.Code)

	ut.Equal(t, viewSelector.GetViews(), []string{"v1", "v2", "v3"})
	_, err = viewSelector.updateViewPriority([]string{"v3", "v2", "v1"})
	ut.Equal(t, err, (*httpcmd.Error)(nil))
	ut.Equal(t, viewSelector.GetViews(), []string{"v3", "v2", "v1"})
}

func TestAddrBaseViewSelectView(t *testing.T) {
	logger.UseDefaultLogger("error")
	httpcmd.ClearHandler()
	viewSelector := newAddrBasedView()

	var conf = &config.VanguardConf{
		Views: config.ViewConf{
			ViewAcls: []config.ViewAcl{
				config.ViewAcl{
					View: "v1",
					Acls: []string{acl.AnyAcl},
				},
			},
		},
	}
	viewSelector.ReloadConfig(conf)

	addr, _ := net.ResolveUDPAddr("udp", "1.1.1.1:50000")
	client := core.Client{
		Addr: addr,
	}
	v, _ := viewSelector.ViewForQuery(&client)
	ut.Equal(t, v, "v1")

	conf = &config.VanguardConf{
		Views: config.ViewConf{
			ViewAcls: []config.ViewAcl{
				config.ViewAcl{
					View: "v1",
					Acls: []string{acl.NoneAcl},
				},
				config.ViewAcl{
					View: "v2",
					Acls: []string{acl.AnyAcl},
				},
			},
		},
	}
	viewSelector.ReloadConfig(conf)
	v, found := viewSelector.ViewForQuery(&client)
	ut.Equal(t, found, true)
	ut.Equal(t, v, "v2")
}
