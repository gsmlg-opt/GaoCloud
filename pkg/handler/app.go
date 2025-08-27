package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"gorest"
	"gorest/adaptor"
	restresource "gorest/resource"
	"gorest/resource/schema"
	"config"
	"pkg/alarm"
	"pkg/auditlog"
	"pkg/authentication"
	"pkg/authorization"
	"pkg/types"
	"pkg/zke/zkelog"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "zcloud.cn",
	}
)

type App struct {
	clusterManager *ClusterManager
	conf           *config.SinglecloudConf
}

func NewApp(authenticator *authentication.Authenticator, authorizer *authorization.Authorizer, conf *config.SinglecloudConf) (*App, error) {
	clusterMgr, err := newClusterManager(authenticator, authorizer)
	if err != nil {
		return nil, err
	}
	return &App{
		clusterManager: clusterMgr,
		conf:           conf,
	}, nil
}

func (a *App) RegisterHandler(router gin.IRoutes) error {
	if err := a.registerRestHandler(router); err != nil {
		return err
	}
	a.registerWSHandler(router)
	return nil
}

func (a *App) registerRestHandler(router gin.IRoutes) error {
	schemas := schema.NewSchemaManager()
	schemas.MustImport(&Version, types.Cluster{}, a.clusterManager)
	schemas.MustImport(&Version, types.Alarm{}, alarm.GetAlarmManager())
	schemas.MustImport(&Version, types.Node{}, newNodeManager(a.clusterManager))
	schemas.MustImport(&Version, types.PodNetwork{}, newPodNetworkManager(a.clusterManager))
	schemas.MustImport(&Version, types.NodeNetwork{}, newNodeNetworkManager(a.clusterManager))
	schemas.MustImport(&Version, types.ServiceNetwork{}, newServiceNetworkManager(a.clusterManager))
	schemas.MustImport(&Version, types.BlockDevice{}, newBlockDeviceManager(a.clusterManager))
	namespaceManager, err := newNamespaceManager(a.clusterManager, a.conf.Server.EnableDebug)
	if err != nil {
		return err
	}
	schemas.MustImport(&Version, types.Namespace{}, namespaceManager)
	schemas.MustImport(&Version, types.Chart{}, newChartManager(a.conf.Chart.Path, a.conf.Chart.Repo))
	schemas.MustImport(&Version, types.ConfigMap{}, newConfigMapManager(a.clusterManager))
	schemas.MustImport(&Version, types.CronJob{}, newCronJobManager(a.clusterManager))
	schemas.MustImport(&Version, types.DaemonSet{}, newDaemonSetManager(a.clusterManager))
	schemas.MustImport(&Version, types.Deployment{}, newDeploymentManager(a.clusterManager))
	schemas.MustImport(&Version, types.Ingress{}, newIngressManager(a.clusterManager))
	schemas.MustImport(&Version, types.Job{}, newJobManager(a.clusterManager))
	schemas.MustImport(&Version, types.LimitRange{}, newLimitRangeManager(a.clusterManager))
	schemas.MustImport(&Version, types.PersistentVolumeClaim{}, newPersistentVolumeClaimManager(a.clusterManager))
	schemas.MustImport(&Version, types.PersistentVolume{}, newPersistentVolumeManager(a.clusterManager))
	schemas.MustImport(&Version, types.ResourceQuota{}, newResourceQuotaManager(a.clusterManager))
	schemas.MustImport(&Version, types.Secret{}, newSecretManager(a.clusterManager))
	schemas.MustImport(&Version, types.Service{}, newServiceManager(a.clusterManager))
	schemas.MustImport(&Version, types.StatefulSet{}, newStatefulSetManager(a.clusterManager))
	schemas.MustImport(&Version, types.Pod{}, newPodManager(a.clusterManager))
	schemas.MustImport(&Version, types.UDPIngress{}, newUDPIngressManager(a.clusterManager))
	schemas.MustImport(&Version, types.StorageClass{}, newStorageClassManager(a.clusterManager))
	schemas.MustImport(&Version, types.InnerService{}, newInnerServiceManager(a.clusterManager))
	schemas.MustImport(&Version, types.OuterService{}, newOuterServiceManager(a.clusterManager))
	schemas.MustImport(&Version, types.KubeConfig{}, newKubeConfigManager(a.clusterManager))
	schemas.MustImport(&Version, types.FluentBitConfig{}, newFluentBitConfigManager(a.clusterManager))
	schemas.MustImport(&Version, types.SvcMeshWorkload{}, newSvcMeshWorkloadManager(a.clusterManager))
	schemas.MustImport(&Version, types.SvcMeshPod{}, newSvcMeshPodManager(a.clusterManager))
	schemas.MustImport(&Version, types.Metric{}, newMetricManager(a.clusterManager))
	schemas.MustImport(&Version, types.WorkFlow{}, newWorkFlowManager(a.clusterManager))
	schemas.MustImport(&Version, types.WorkFlowTask{}, newWorkFlowTaskManager(a.clusterManager))

	auditLogger, err := auditlog.New()
	if err != nil {
		return err
	}
	schemas.MustImport(&Version, types.AuditLog{}, newAuditLogManager(auditLogger))

	userQuotaManager, err := newUserQuotaManager(a.clusterManager)
	if err != nil {
		return err
	}
	schemas.MustImport(&Version, types.UserQuota{}, userQuotaManager)
	appManager := newApplicationManager(a.clusterManager, a.conf.Chart.Path)
	schemas.MustImport(&Version, types.Application{}, appManager)
	schemas.MustImport(&Version, types.Monitor{}, newMonitorManager(a.clusterManager, a.conf.Chart.Path))
	schemas.MustImport(&Version, types.EFK{}, newEFKManager(a.clusterManager, a.conf.Chart.Path))

	registryManager, err := newRegistryManager(a.clusterManager, a.conf.Chart.Path, a.conf.Registry)
	if err != nil {
		return err
	}
	schemas.MustImport(&Version, types.Registry{}, registryManager)
	thresholdManager, err := newThresholdManager(a.clusterManager)
	if err != nil {
		return err
	}
	schemas.MustImport(&Version, types.Threshold{}, thresholdManager)

	schemas.MustImport(&Version, types.Storage{}, newStorageManager(a.clusterManager))

	userManager := newUserManager(a.clusterManager.authenticator.JwtAuth, a.clusterManager.authorizer)
	schemas.MustImport(&Version, types.User{}, userManager)
	schemas.MustImport(&Version, types.HorizontalPodAutoscaler{}, newHorizontalPodAutoscalerManager(a.clusterManager))
	server := gorest.NewAPIServer(schemas)
	server.Use(a.clusterManager.authorizationHandler(a.conf.Server.EnableDebug))
	server.Use(auditLogger.AuditHandler())

	adaptor.RegisterHandler(router, server, schemas.GenerateResourceRoute())
	return nil
}

const (
	WSPrefix                  = "/apis/ws.zcloud.cn/v1"
	WSPodLogPathTemp          = WSPrefix + "/clusters/%s/namespaces/%s/pods/%s/containers/%s/log"
	WSTapPathTemp             = WSPrefix + "/clusters/%s/namespaces/%s/tap"
	WSWorkFlowTaskLogPathTemp = WSPrefix + "/clusters/%s/namespaces/%s/workflows/%s/workflowtasks/%s/log"
)

func (a *App) registerWSHandler(router gin.IRoutes) {
	podLogPath := fmt.Sprintf(WSPodLogPathTemp, ":cluster", ":namespace", ":pod", ":container")
	router.GET(podLogPath, func(c *gin.Context) {
		a.clusterManager.OpenPodLog(c.Param("cluster"), c.Param("namespace"), c.Param("pod"), c.Param("container"), c.Request, c.Writer)
	})

	zkeLogPath := fmt.Sprintf(zkelog.WSZKELogPathTemp, ":cluster")
	router.GET(zkeLogPath, func(c *gin.Context) {
		a.clusterManager.zkeManager.OpenLog(c.Param("cluster"), c.Request, c.Writer)
	})

	tapPath := fmt.Sprintf(WSTapPathTemp, ":cluster", ":namespace")
	router.GET(tapPath, func(c *gin.Context) {
		a.clusterManager.Tap(c.Param("cluster"), c.Param("namespace"), c.Query("resource_type"), c.Query("resource_name"), c.Query("to_resource_type"), c.Query("to_resource_name"), c.Query("method"), c.Query("path"), c.Request, c.Writer)
	})

	workFlowTaskLogPath := fmt.Sprintf(WSWorkFlowTaskLogPathTemp, ":cluster", ":namespace", ":workflow", ":workflowtask")
	router.GET(workFlowTaskLogPath, func(c *gin.Context) {
		a.clusterManager.OpenWorkFlowTaskLog(c.Param("cluster"), c.Param("namespace"), c.Param("workflow"), c.Param("workflowtask"), c.Request, c.Writer)
	})
}
