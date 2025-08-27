package viewselector

import (
	"vanguard/config"
	"vanguard/core"
)

type ViewSelector interface {
	ReloadConfig(*config.VanguardConf)
	ViewForQuery(*core.Client) (string, bool)
	GetViews() []string
}
