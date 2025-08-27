package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gsmlg-opt/GaoCloud/cement/log"
	"github.com/gsmlg-opt/GaoCloud/cement/x509"
	"gopkg.in/yaml.v2"

	"github.com/gsmlg-opt/GaoCloud/config"
	"github.com/gsmlg-opt/GaoCloud/pkg/alarm"
	"github.com/gsmlg-opt/GaoCloud/pkg/authentication"
	"github.com/gsmlg-opt/GaoCloud/pkg/authorization"
	"github.com/gsmlg-opt/GaoCloud/pkg/clusteragent"
	"github.com/gsmlg-opt/GaoCloud/pkg/db"
	"github.com/gsmlg-opt/GaoCloud/pkg/globaldns"
	"github.com/gsmlg-opt/GaoCloud/pkg/handler"
	"github.com/gsmlg-opt/GaoCloud/pkg/k8seventwatcher"
	"github.com/gsmlg-opt/GaoCloud/pkg/k8sshell"
	"github.com/gsmlg-opt/GaoCloud/server"
)

const (
	defaultTlsCertFile = "/tmp_tls_cert.crt"
	defaultTlsKeyFile  = "/tmp_tls_key.key"
)

var (
	configFile  string
	version     string
	showVersion bool
	genConfFile bool
	build       string
)

func main() {
	flag.StringVar(&configFile, "c", "gaocloud.conf", "configure file path")
	flag.BoolVar(&genConfFile, "gen", false, "generate initial configure file to current directory")
	flag.BoolVar(&showVersion, "version", false, "show version")
	flag.Parse()

	log.InitLogger(log.Debug)

	if showVersion {
		fmt.Printf("gaocloud %s (build at %s)\n", version, build)
		return
	}

	if genConfFile {
		if err := genInitConfig(); err != nil {
			log.Fatalf("generate initial configure file failed:%s", err.Error())
		}
		return
	}

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load configure file failed:%s", err.Error())
	}

	if conf.DB.Role == config.Master {
		runAsMaster(conf)
	} else {
		runAsSlave(conf)
	}
}

func runAsMaster(conf *config.GaoCloudConf) {
	stopCh := make(chan struct{})
	err := db.RunAsMaster(conf, stopCh)
	if err != nil {
		log.Fatalf("create database failed: %s", err.Error())
	}
	defer close(stopCh)

	if err := globaldns.New(conf.Server.DNSAddr); err != nil {
		log.Fatalf("create globaldns failed: %v", err.Error())
	}

	authenticator, err := authentication.New(conf.Server.CasAddr)
	if err != nil {
		log.Fatalf("create authenticator failed:%s", err.Error())
	}

	authorizer, err := authorization.New()
	if err != nil {
		log.Fatalf("create authorizer failed:%s", err.Error())
	}

	server, err := server.NewServer(authenticator.MiddlewareFunc())
	if err != nil {
		log.Fatalf("create server failed:%s", err.Error())
	}

	watcher := k8seventwatcher.New()
	if err := server.RegisterHandler(watcher); err != nil {
		log.Fatalf("register k8s event watcher failed:%s", err.Error())
	}

	if err := alarm.NewAlarmManager(); err != nil {
		log.Fatalf("create alarm failed:%s", err.Error())
	}

	if err := server.RegisterHandler(alarm.GetAlarmManager()); err != nil {
		log.Fatalf("register alarm failed:%s", err.Error())
	}

	shellExecutor := k8sshell.New()
	if err := server.RegisterHandler(shellExecutor); err != nil {
		log.Fatalf("register shell executor failed:%s", err.Error())
	}

	if err := server.RegisterHandler(clusteragent.GetAgent()); err != nil {
		log.Fatalf("register agent failed:%s", err.Error())
	}

	app, err := handler.NewApp(authenticator, authorizer, conf)
	if err != nil {
		log.Fatalf("create app failed %s", err.Error())
	}

	if err := server.RegisterHandler(authenticator); err != nil {
		log.Fatalf("register redirect handler failed:%s", err.Error())
	}

	if err := server.RegisterHandler(app); err != nil {
		log.Fatalf("register resource handler failed:%s", err.Error())
	}

	if conf.Server.TlsCertFile == "" && conf.Server.TlsKeyFile == "" {
		if err := createSelfSignedTlsCert(); err != nil {
			log.Fatalf("create selfsigned tls cert failed %s", err.Error())
		}
		if err := server.Run(conf.Server.Addr, defaultTlsCertFile, defaultTlsKeyFile); err != nil {
			log.Fatalf("server run failed:%s", err.Error())
		}
	} else {
		if err := server.Run(conf.Server.Addr, conf.Server.TlsCertFile, conf.Server.TlsKeyFile); err != nil {
			log.Fatalf("server run failed:%s", err.Error())
		}
	}
}

func createSelfSignedTlsCert() error {
	_, err := os.Stat(defaultTlsCertFile)
	if err != nil && os.IsExist(err) {
		return nil
	}

	cert, err := x509.GenerateSelfSignedCertificate("zcloud", nil, nil, 7300)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(defaultTlsCertFile, []byte(cert.Cert), 0644); err != nil {
		return err
	}
	return ioutil.WriteFile(defaultTlsKeyFile, []byte(cert.Key), 0644)
}

func runAsSlave(conf *config.GaoCloudConf) {
	db.RunAsSlave(conf)
}

func genInitConfig() error {
	yamlConfig, err := yaml.Marshal(config.CreateDefaultConfig())
	if err != nil {
		return err
	}
	configFile := "./gaocloud.conf"
	log.Debugf("Deploying cluster configuration file: %s", configFile)
	return ioutil.WriteFile(configFile, yamlConfig, 0640)
}
