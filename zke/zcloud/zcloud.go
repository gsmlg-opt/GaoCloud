package zcloud

import (
	"context"
	"fmt"

	"zke/core"
	"zke/core/pki"
	"zke/pkg/k8s"
	"zke/pkg/log"
	"zke/pkg/util"
	appoperator "zke/zcloud/application-operator"
	"zke/zcloud/cicd"
	clusteragent "zke/zcloud/cluster-agent"
	"zke/zcloud/lbcontroller"
	nodeagent "zke/zcloud/node-agent"
	"zke/zcloud/proxy"
	zcloudsa "zke/zcloud/sa"
	"zke/zcloud/servicemesh"
	"zke/zcloud/storage"
	zcloudshell "zke/zcloud/zcloud-shell"

	"gok8s/client"
)

const (
	RBACConfig               = "RBACConfig"
	Image                    = "Image"
	NodeAgentPort            = "9001"
	ClusterAgentResourceName = "cluster-agent"
	SAResourceName           = "sa"
	ClusterAgentJobName      = "zcloud-cluster-agent"
	SAJobName                = "zcloud-sa"

	StorageNFSProvisionerImage = "StorageNFSProvisionerImage"
)

type deployFunc func(ctx context.Context, c *core.Cluster, client client.Client) error

var deploys []deployFunc = []deployFunc{
	deployServiceAccount,
	deployClusterAgent,
	deployNodeAgent,
	deployStorageOperator,
	deployZcloudShell,
	deployServiceMesh,
	deployCICD,
	deployApplicationOperator,
}

func DeployZcloudComponents(ctx context.Context, c *core.Cluster) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		k8sClient, err := k8s.GetK8sClientFromYaml(c.Certificates[pki.KubeAdminCertName].Config)
		if err != nil {
			return err
		}
		for _, f := range deploys {
			if err := f(ctx, c, k8sClient); err != nil {
				return err
			}
		}
		return nil
	}
}

func DeployZcloudProxy(ctx context.Context, c *core.Cluster) error {
	if c.SingleCloudAddress == "" {
		log.Infof(ctx, "[zcloud] gaocloud address is empty, skip deploy ZcloudProxy")
		return nil
	}
	log.Infof(ctx, "[zcloud] deploy ZcloudProxy")
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		k8sClient, err := k8s.GetK8sClientFromYaml(c.Certificates[pki.KubeAdminCertName].Config)
		if err != nil {
			return err
		}
		return proxy.CreateOrUpdate(k8sClient, c)
	}
}

func DeployZcloudLBController(ctx context.Context, c *core.Cluster) error {
	select {
	case <-ctx.Done():
		return util.CancelErr
	default:
		k8sClient, err := k8s.GetK8sClientFromYaml(c.Certificates[pki.KubeAdminCertName].Config)
		if err != nil {
			return err
		}

		if !c.LoadBalance.Enable {
			log.Infof(ctx, "[zcloud] LoadBalance disabled, will delete it if exist")
			return lbcontroller.DeleteIfExist(k8sClient, c)
		}

		log.Infof(ctx, "[zcloud] deploy ZcloudLBController")
		return lbcontroller.CreateOrUpdate(k8sClient, c)
	}
}

func deployServiceAccount(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[zcloud] Setting up ZcloudSADeploy : %s", SAResourceName)
	saconfig := map[string]interface{}{
		RBACConfig: c.Authorization.Mode,
	}
	return k8s.DoCreateFromTemplate(cli, zcloudsa.SATemplate, saconfig)
}

func deployClusterAgent(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[zcloud] Setting up ClusterAgentDeploy : %s", ClusterAgentResourceName)
	clusteragentConfig := map[string]interface{}{
		Image: c.Image.ClusterAgent,
	}
	return k8s.DoCreateFromTemplate(cli, clusteragent.ClusterAgentTemplate, clusteragentConfig)
}

func deployNodeAgent(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[zcloud] Setting up NodeAgent")
	cfg := map[string]interface{}{
		Image:           c.Image.NodeAgent,
		"NodeAgentPort": NodeAgentPort,
	}
	return k8s.DoCreateFromTemplate(cli, nodeagent.NodeAgentTemplate, cfg)
}
func deployStorageOperator(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[zcloud] Setting up storage CRD and operator")
	cfg := map[string]interface{}{
		RBACConfig:             c.Authorization.Mode,
		"StorageOperatorImage": c.Image.StorageOperator,
	}
	return k8s.DoCreateFromTemplate(cli, storage.OperatorTemplate, cfg)
}

func deployZcloudShell(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[zcloud] deploy zcloud-shell")
	cfg := map[string]interface{}{
		"ZcloudShellImage": c.Image.ZcloudShell,
	}
	return k8s.DoCreateFromTemplate(cli, zcloudshell.ZcloudShellTemplate, cfg)
}

func deployServiceMesh(ctx context.Context, c *core.Cluster, cli client.Client) error {
	if c.DisableLinkerd {
		log.Infof(ctx, "[zcloud] linkerd disabled, skip it")
		return nil
	}
	log.Infof(ctx, "[zcloud] deploy servicemesh")
	cfg, err := servicemesh.GetDeployConfig(c.ZKEConfig.Option.ClusterDomain, c.Image.ServiceMesh)
	if err != nil {
		return fmt.Errorf("get servicemesh deploy config failed: %s", err.Error())
	}

	return k8s.DoCreateFromTemplate(cli, servicemesh.Template, cfg)
}

func deployCICD(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[zcloud] deploy cicd")
	if err := k8s.DoCreateFromTemplate(cli, cicd.TektonTemplate, cicd.GetDeployConfig(c)); err != nil {
		return err
	}
	return k8s.DoCreateFromTemplate(cli, cicd.TektonDashBoardTemplate, cicd.GetDeployConfig(c))
}

func deployApplicationOperator(ctx context.Context, c *core.Cluster, cli client.Client) error {
	log.Infof(ctx, "[zcloud] deploy application operator")
	appOperatorConfig := map[string]interface{}{
		Image: c.Image.ApplicationOperator,
	}
	return k8s.DoCreateFromTemplate(cli, appoperator.ApplicationOperatorTemplate, appOperatorConfig)
}
