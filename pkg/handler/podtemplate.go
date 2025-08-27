package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"cement/log"
	"gok8s/client"
	resttypes "gorest/resource"
	"pkg/types"
)

var FilesystemVolumeMode = corev1.PersistentVolumeFilesystem

const (
	AnnotationForCreateWorkload          = "init version"
	VolumeNamePrefix                     = "vol"
	AnnkeyForWordloadAdvancedoption      = "zcloud_workload_advanded_options"
	AnnkeyForPromethusScrape             = "prometheus.io/scrape"
	AnnkeyForPromethusPort               = "prometheus.io/port"
	AnnkeyForPromethusPath               = "prometheus.io/path"
	AnnKeyForReloadWhenConfigChange      = "zcloud.cn/update-on-config-change"
	AnnKeyForConfigHashAnnotation        = "zcloud.cn/config-hash"
	AnnkeyForDeletePVsWhenDeleteWorkload = "zcloud_delete_pvs_when_delete_workload"
	AnnKeyForInjectServiceMesh           = "linkerd.io/inject"

	DefaultRequestCPU    = "10m"
	DefaultRequestMemory = "20Mi"
)

func createPodTempateSpec(namespace string, podOwner interface{}, cli client.Client) (*corev1.PodTemplateSpec, []corev1.PersistentVolumeClaim, error) {
	structVal := reflect.ValueOf(podOwner).Elem()
	advancedOpts := structVal.FieldByName("AdvancedOptions").Interface().(types.AdvancedOptions)
	containers := structVal.FieldByName("Containers").Interface().([]types.Container)
	pvs := structVal.FieldByName("PersistentVolumes").Interface().([]types.PersistentVolumeTemplate)

	k8sPodSpec, k8sPVCs, err := scPodSpecToK8sPodSpecAndPVCs(cli, containers, pvs)
	if err != nil {
		return nil, nil, err
	}

	name := structVal.FieldByName("Name").String()
	meta, err := createPodTempateObjectMeta(name, namespace, cli, advancedOpts, containers)
	if err != nil {
		return nil, nil, err
	}

	if _, ok := podOwner.(*types.StatefulSet); ok == false {
		if err := createPVCs(cli, namespace, k8sPVCs); err != nil {
			return nil, nil, err
		}
	}

	return &corev1.PodTemplateSpec{
		ObjectMeta: meta,
		Spec:       k8sPodSpec,
	}, k8sPVCs, nil
}

func generatePodOwnerObjectMeta(namespace string, podOwner interface{}) metav1.ObjectMeta {
	structVal := reflect.ValueOf(podOwner).Elem()
	advancedOpts := structVal.FieldByName("AdvancedOptions").Interface().(types.AdvancedOptions)
	opts, _ := json.Marshal(advancedOpts)
	memo := structVal.FieldByName("Memo").String()
	if memo == "" {
		memo = AnnotationForCreateWorkload
	}
	annotations := map[string]string{
		AnnkeyForWordloadAdvancedoption: string(opts),
		ChangeCauseAnnotation:           memo,
	}
	if advancedOpts.ReloadWhenConfigChange {
		annotations[AnnKeyForReloadWhenConfigChange] = "true"
	}
	if advancedOpts.DeletePVsWhenDeleteWorkload {
		annotations[AnnkeyForDeletePVsWhenDeleteWorkload] = "true"
	}
	return metav1.ObjectMeta{
		Name:        structVal.FieldByName("Name").String(),
		Namespace:   namespace,
		Annotations: annotations,
	}
}

func createPVCs(cli client.Client, namespace string, k8sPVCs []corev1.PersistentVolumeClaim) error {
	var err error
	for _, pvc := range k8sPVCs {
		pvc.Namespace = namespace
		if e := cli.Create(context.TODO(), &pvc); e != nil {
			err = fmt.Errorf("create pvc %s with namespace %s failed: %s", pvc.Name, namespace, e.Error())
			break
		}
	}

	if err != nil {
		deletePVCs(cli, namespace, k8sPVCs)
	}

	return err
}

func getPVCs(cli client.Client, namespace string, templates []types.PersistentVolumeTemplate) ([]types.PersistentVolumeTemplate, error) {
	var pvTemplates []types.PersistentVolumeTemplate
	for _, template := range templates {
		if template.StorageClassName != types.StorageClassNameTemp {
			k8sPVC, err := getPersistentVolumeClaim(cli, namespace, template.Name)
			if err != nil {
				return nil, err
			}
			pvc := k8sPVCToSCPVC(k8sPVC)
			pvTemplates = append(pvTemplates, types.PersistentVolumeTemplate{
				Name:             pvc.Name,
				Size:             pvc.RequestStorageSize,
				StorageClassName: pvc.StorageClassName,
			})
		}
	}

	return pvTemplates, nil
}

func deletePVCs(cli client.Client, namespace string, k8sPVCs []corev1.PersistentVolumeClaim) {
	for _, pvc := range k8sPVCs {
		deletePVC(cli, namespace, pvc.Name)
	}
}

func deletePVC(cli client.Client, namespace, pvcName string) {
	k8sPVC, err := getPersistentVolumeClaim(cli, namespace, pvcName)
	if err != nil {
		log.Warnf("get persistentvolumeclaim %s failed:%s", pvcName, err.Error())
		return
	}

	if err := deletePersistentVolumeClaim(cli, namespace, pvcName); err != nil {
		log.Warnf("delete persistentvolumeclaim %s failed:%s", pvcName, err.Error())
	}

	if volumeName := k8sPVC.Spec.VolumeName; volumeName != "" {
		if _, err := getPersistentVolume(cli, volumeName); err != nil {
			if apierrors.IsNotFound(err) == false {
				log.Warnf("get persistentvolume %s failed:%s", volumeName, err.Error())
			}
		} else {
			if err := deletePersistentVolume(cli, volumeName); err != nil {
				log.Warnf("delete persistentvolume %s failed:%s", volumeName, err.Error())
			}
		}
	}
}

func deleteWorkLoadPVCs(cli client.Client, namespace string, k8sVolumes []corev1.Volume) {
	for _, volume := range k8sVolumes {
		if volume.PersistentVolumeClaim != nil {
			deletePVC(cli, namespace, volume.PersistentVolumeClaim.ClaimName)
		}
	}
}

func scPodSpecToK8sPodSpecAndPVCs(cli client.Client, containers []types.Container, pvs []types.PersistentVolumeTemplate) (corev1.PodSpec, []corev1.PersistentVolumeClaim, error) {
	var k8sPodSpec corev1.PodSpec
	k8sEmptyDirs, k8sPVCs, err := scPVCsToK8sVolumesAndPVCs(cli, pvs)
	if err != nil {
		return k8sPodSpec, nil, err
	}

	k8sPodSpec, err = scContainersAndPVToK8sPodSpec(containers, k8sEmptyDirs, k8sPVCs)
	return k8sPodSpec, k8sPVCs, err
}

func scPVCsToK8sVolumesAndPVCs(cli client.Client, pvs []types.PersistentVolumeTemplate) ([]corev1.Volume, []corev1.PersistentVolumeClaim, error) {
	if len(pvs) == 0 {
		return nil, nil, nil
	}

	var k8sEmptydirVolumes []corev1.Volume
	var k8sPVCs []corev1.PersistentVolumeClaim
	for _, pv := range pvs {
		storageClassName := pv.StorageClassName
		if storageClassName == "" {
			return nil, nil, fmt.Errorf("persistent volume storageclass name should not be empty")
		}

		var k8sQuantity *resource.Quantity
		if pv.Size != "" {
			quantity, err := resource.ParseQuantity(pv.Size)
			if err != nil {
				return nil, nil, fmt.Errorf("parse storage size %s failed: %s", pv.Size, err.Error())
			}
			k8sQuantity = &quantity
		}

		var accessModes []corev1.PersistentVolumeAccessMode
		switch storageClassName {
		case types.StorageClassNameTemp:
			k8sEmptydirVolumes = append(k8sEmptydirVolumes, corev1.Volume{
				Name: pv.Name,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{
						SizeLimit: k8sQuantity,
					},
				},
			})
			continue
		default:
			storageClass, err := getStorageClass(cli, storageClassName)
			if err != nil {
				return nil, nil, fmt.Errorf("get storageClass %s failed: %s", storageClassName, err.Error())
			}
			if accessMode, ok := storageClass.Parameters["accessMode"]; ok {
				accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(accessMode))
			} else {
				return nil, nil, fmt.Errorf("parse storage %s accessMode failed: %s", storageClassName, err.Error())
			}
		}
		if k8sQuantity == nil {
			return nil, nil, fmt.Errorf("persistentClaimVolumes storage size must not be zero")
		}

		k8sPVCs = append(k8sPVCs, corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: pv.Name,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: accessModes,
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceStorage: *k8sQuantity,
					},
				},
				StorageClassName: &storageClassName,
				VolumeMode:       &FilesystemVolumeMode,
			},
		})
	}

	return k8sEmptydirVolumes, k8sPVCs, nil
}

func scContainersAndPVToK8sPodSpec(containers []types.Container, k8sEmptyDirs []corev1.Volume, k8sPVCs []corev1.PersistentVolumeClaim) (corev1.PodSpec, error) {
	var k8sContainers []corev1.Container
	var k8sVolumes []corev1.Volume
	for _, c := range containers {
		var mounts []corev1.VolumeMount
		var ports []corev1.ContainerPort
		var env []corev1.EnvVar
		for i, volume := range c.Volumes {
			readOnly := true
			exists := false
			volumeName := c.Name + "-" + VolumeNamePrefix + strconv.Itoa(i)
			var volumeSource corev1.VolumeSource
			switch volume.Type {
			case types.VolumeTypeConfigMap:
				volumeSource = corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: volume.Name,
						},
					},
				}
			case types.VolumeTypeSecret:
				volumeSource = corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: volume.Name,
					},
				}
			case types.VolumeTypePersistentVolume:
				readOnly = false
				found := false
				for _, emptydir := range k8sEmptyDirs {
					if emptydir.Name == volume.Name {
						volumeName = emptydir.Name
						volumeSource = emptydir.VolumeSource
						found = true
						break
					}
				}

				if found == false {
					for _, pvc := range k8sPVCs {
						if pvc.Name == volume.Name {
							volumeName = pvc.Name
							volumeSource = corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: volume.Name,
								},
							}
							found = true
							break
						}
					}
				}

				if found == false {
					return corev1.PodSpec{}, fmt.Errorf("no found volume %s in persistent volume", volume.Name)
				} else {
					for _, k8sVolume := range k8sVolumes {
						if k8sVolume.Name == volumeName {
							exists = true
							break
						}
					}
				}
			default:
				return corev1.PodSpec{}, fmt.Errorf("volume type %s is unsupported", volume.Type)
			}

			if exists == false {
				k8sVolumes = append(k8sVolumes, corev1.Volume{
					Name:         volumeName,
					VolumeSource: volumeSource,
				})
			}

			mounts = append(mounts, corev1.VolumeMount{
				Name:      volumeName,
				MountPath: volume.MountPath,
				ReadOnly:  readOnly,
			})
		}

		var portNames []string
		for _, spec := range c.ExposedPorts {
			protocol, err := scPortProtocolToK8SProtocol(spec.Protocol)
			if err != nil {
				return corev1.PodSpec{}, fmt.Errorf("invalid protocol %s for container port", spec.Protocol)
			}

			if err := validatePortName(spec.Name); err != nil {
				return corev1.PodSpec{}, fmt.Errorf("exposed port name %s invalid: %s", spec.Name, err.Error())
			}

			for _, pn := range portNames {
				if pn == spec.Name {
					return corev1.PodSpec{}, fmt.Errorf("duplicate container port name %s", pn)
				}
			}
			portNames = append(portNames, spec.Name)

			ports = append(ports, corev1.ContainerPort{
				Name:          spec.Name,
				ContainerPort: int32(spec.Port),
				Protocol:      protocol,
			})
		}

		for _, e := range c.Env {
			env = append(env, corev1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}

		k8sContainers = append(k8sContainers, corev1.Container{
			Name:         c.Name,
			Image:        c.Image,
			Command:      c.Command,
			Args:         c.Args,
			VolumeMounts: mounts,
			Ports:        ports,
			Env:          env,
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceCPU:    apiresource.MustParse(DefaultRequestCPU),
					corev1.ResourceMemory: apiresource.MustParse(DefaultRequestMemory),
				},
			},
		})
	}

	return corev1.PodSpec{
		Containers: k8sContainers,
		Volumes:    k8sVolumes,
	}, nil
}

var (
	portNameCharsetRegex    = regexp.MustCompile("^[-a-z0-9]+$")
	portNameOneLetterRegexp = regexp.MustCompile("[a-z]")
)

const maxPortNameLen = 15

func validatePortName(name string) error {
	if len(name) > maxPortNameLen {
		return fmt.Errorf("must be no more than 15 characters")
	}
	if !portNameCharsetRegex.MatchString(name) {
		return fmt.Errorf("must contain only alpha-numeric characters (a-z, 0-9), and hyphens (-)")
	}
	if !portNameOneLetterRegexp.MatchString(name) {
		return fmt.Errorf("must contain at least one letter (a-z)")
	}
	if strings.Contains(name, "--") {
		return fmt.Errorf("must not contain consecutive hyphens(--)")
	}
	if len(name) > 0 && (name[0] == '-' || name[len(name)-1] == '-') {
		return fmt.Errorf("must not begin or end with a hyphen (-)")
	}
	return nil
}

func k8sPodSpecToScContainersAndVCTemplates(k8sContainers []corev1.Container, k8sVolumes []corev1.Volume) ([]types.Container, []types.PersistentVolumeTemplate) {
	var containers []types.Container
	var templates []types.PersistentVolumeTemplate
	for _, c := range k8sContainers {
		var volumes []types.Volume
		for _, vm := range c.VolumeMounts {
			for _, v := range k8sVolumes {
				if v.Name == vm.Name {
					var template types.PersistentVolumeTemplate
					if v.ConfigMap != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypeConfigMap,
							Name:      v.ConfigMap.Name,
							MountPath: vm.MountPath,
						})
					} else if v.Secret != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypeSecret,
							Name:      v.Secret.SecretName,
							MountPath: vm.MountPath,
						})
					} else if v.PersistentVolumeClaim != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypePersistentVolume,
							Name:      v.PersistentVolumeClaim.ClaimName,
							MountPath: vm.MountPath,
						})
						template.Name = v.PersistentVolumeClaim.ClaimName
					} else if v.EmptyDir != nil {
						volumes = append(volumes, types.Volume{
							Type:      types.VolumeTypePersistentVolume,
							Name:      v.Name,
							MountPath: vm.MountPath,
						})
						template.Name = v.Name
						template.StorageClassName = types.StorageClassNameTemp
						if v.EmptyDir.SizeLimit != nil {
							template.Size = v.EmptyDir.SizeLimit.String()
						}
					}
					if template.Name != "" {
						exists := false
						for _, t := range templates {
							if t.Name == template.Name {
								exists = true
								break
							}
						}
						if exists == false {
							templates = append(templates, template)
						}
					}
					break
				}
			}
		}

		var exposedPorts []types.ContainerPort
		for _, p := range c.Ports {
			exposedPorts = append(exposedPorts, types.ContainerPort{
				Name:     p.Name,
				Port:     int(p.ContainerPort),
				Protocol: strings.ToLower(string(p.Protocol)),
			})
		}

		var env []types.EnvVar
		for _, e := range c.Env {
			env = append(env, types.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}

		containers = append(containers, types.Container{
			Name:         c.Name,
			Image:        c.Image,
			Command:      c.Command,
			Args:         c.Args,
			ExposedPorts: exposedPorts,
			Env:          env,
			Volumes:      volumes,
		})
	}

	return containers, templates
}

func createPodTempateObjectMeta(name, namespace string, cli client.Client, advancedOpts types.AdvancedOptions, containers []types.Container) (metav1.ObjectMeta, error) {
	meta := metav1.ObjectMeta{
		Labels:      map[string]string{"app": name},
		Annotations: make(map[string]string)}

	exposedMetric := advancedOpts.ExposedMetric
	if exposedMetric.Port != 0 && exposedMetric.Path != "" {
		meta.Annotations[AnnkeyForPromethusScrape] = "true"
		meta.Annotations[AnnkeyForPromethusPort] = strconv.Itoa(exposedMetric.Port)
		meta.Annotations[AnnkeyForPromethusPath] = exposedMetric.Path
	}

	if advancedOpts.ReloadWhenConfigChange {
		configs, err := getConfigmapAndSecretContainersUse(namespace, cli, containers)
		if err != nil {
			return meta, err
		}

		if len(configs) > 0 {
			hash, err := calculateConfigHash(configs)
			if err != nil {
				return meta, err
			}
			meta.Annotations[AnnKeyForConfigHashAnnotation] = hash
		}
	}

	if advancedOpts.InjectServiceMesh {
		meta.Annotations[AnnKeyForInjectServiceMesh] = "enabled"
	}

	return meta, nil
}

func k8sAnnotationsToScExposedMetric(annotations map[string]string) types.ExposedMetric {
	if doScrape, ok := annotations[AnnkeyForPromethusScrape]; ok && doScrape == "true" {
		port, _ := strconv.Atoi(annotations[AnnkeyForPromethusPort])
		return types.ExposedMetric{
			Port: port,
			Path: annotations[AnnkeyForPromethusPath],
		}
	}
	return types.ExposedMetric{}
}

func k8sWorkloadConditionsToScWorkloadConditions(k8sConditions interface{}, isDeploy bool) []types.WorkloadCondition {
	k8sConditionsData := reflect.ValueOf(k8sConditions)
	var conditions []types.WorkloadCondition
	if k8sConditionsData.Kind() == reflect.Slice {
		for i := 0; i < k8sConditionsData.Len(); i++ {
			val := k8sConditionsData.Index(i)
			condition := types.WorkloadCondition{
				Type:               val.FieldByName("Type").String(),
				Status:             val.FieldByName("Status").String(),
				LastTransitionTime: resttypes.ISOTime(val.FieldByName("LastTransitionTime").Interface().(metav1.Time).Time),
				Reason:             val.FieldByName("Reason").String(),
				Message:            val.FieldByName("Message").String(),
			}
			if isDeploy {
				condition.LastUpdateTime = resttypes.ISOTime(val.FieldByName("LastUpdateTime").Interface().(metav1.Time).Time)
			}
			conditions = append(conditions, condition)
		}
	}

	return conditions
}

func addWorkloadUpdateMemoToAnnotations(annotations map[string]string, memo string) map[string]string {
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[ChangeCauseAnnotation] = memo
	return annotations
}
