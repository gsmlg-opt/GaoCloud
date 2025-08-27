package services

import (
	"context"

	"zke/pkg/docker"
	"zke/pkg/hosts"
	"zke/types"
)

func runKubeproxy(ctx context.Context, host *hosts.Host, prsMap map[string]types.PrivateRegistry, kubeProxyProcess types.Process, alpineImage string) error {
	imageCfg, hostCfg, healthCheckURL := GetProcessConfig(kubeProxyProcess)
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, KubeproxyContainerName, host.Address, WorkerRole, prsMap); err != nil {
		return err
	}
	if err := runHealthcheck(ctx, host, KubeproxyContainerName, healthCheckURL, nil); err != nil {
		return err
	}
	return createLogLink(ctx, host, KubeproxyContainerName, WorkerRole, alpineImage, prsMap)
}

func removeKubeproxy(ctx context.Context, host *hosts.Host) error {
	return docker.DoRemoveContainer(ctx, host.DClient, KubeproxyContainerName, host.Address)
}

func RestartKubeproxy(ctx context.Context, host *hosts.Host) error {
	return docker.DoRestartContainer(ctx, host.DClient, KubeproxyContainerName, host.Address)
}
