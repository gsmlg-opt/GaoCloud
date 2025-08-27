package acl

import (
	"net"
	"os"
	"testing"

	ut "cement/unittest"
	"vanguard/config"
	"vanguard/httpcmd"
	"vanguard/logger"
)

var logFile = "acl.log"

var generalLog = config.GeneralLogConf{
	Level:    "debug",
	Path:     logFile,
	FileSize: 50000000000,
	Versions: 5,
}
var logconf = config.LoggerConf{
	GeneralLog: generalLog,
}
var conf = &config.VanguardConf{
	Logger: logconf,
}

func TestAclCmd(t *testing.T) {
	logger.UseDefaultLogger("debug")
	defer os.Remove(logFile)

	NewAclManager(conf)

	ut.Equal(t, GetAclManager().Find("a1", net.IP{2, 2, 2, 2}), false)
	result, err := GetAclManager().addAcl("a1", []string{"1.1.1.1", "2.2.2.0/24"})
	ut.Equal(t, err, (*httpcmd.Error)(nil))

	ut.Equal(t, GetAclManager().Find("a1", net.IP{2, 2, 2, 2}), true)

	_, err = GetAclManager().addAcl("any", []string{"1.1.1.1", "2.2.2.0/24"})
	ut.Equal(t, err.Info, "any or none or all acl is read only")

	result, err = GetAclManager().updateAcl("a1", []string{"1.1.1.0/24", "2.2.2.2"})
	ut.Equal(t, err, (*httpcmd.Error)(nil))
	ut.Equal(t, result, nil)

	ut.Equal(t, GetAclManager().Find("a1", net.IP{1, 1, 1, 2}), true)

	result, err = GetAclManager().deleteAcl("a1")
	ut.Equal(t, err, (*httpcmd.Error)(nil))
	ut.Equal(t, result, nil)
	ut.Assert(t, GetAclManager().hasAcl("a1") == false, "")
}
