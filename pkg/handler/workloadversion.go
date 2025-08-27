package handler

import (
	"context"
	"encoding/json"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"gok8s/client"
)

func getControllerRevisions(cli client.Client, namespace string, selector *metav1.LabelSelector, uid k8stypes.UID) ([]appsv1.ControllerRevision, error) {
	controllerRevisionList := appsv1.ControllerRevisionList{}
	opts := &client.ListOptions{Namespace: namespace}
	labels, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, err
	}

	opts.LabelSelector = labels
	if err := cli.List(context.TODO(), opts, &controllerRevisionList); err != nil {
		return nil, err
	}

	var controllerRevisions []appsv1.ControllerRevision
	for _, item := range controllerRevisionList.Items {
		if isControllerBy(item.OwnerReferences, uid) {
			controllerRevisions = append(controllerRevisions, item)
		}
	}
	return controllerRevisions, nil
}

func marshalPatch(template corev1.PodTemplateSpec, annotations map[string]string) ([]byte, error) {
	return json.Marshal([]interface{}{
		map[string]interface{}{
			"op":    "replace",
			"path":  "/spec/template",
			"value": template,
		},
		map[string]interface{}{
			"op":    "replace",
			"path":  "/metadata/annotations",
			"value": annotations,
		},
	})
}

func isControllerBy(ownerRefs []metav1.OwnerReference, uid k8stypes.UID) bool {
	for _, ref := range ownerRefs {
		if ref.Controller != nil && *ref.Controller && ref.UID == uid {
			return true
		}
	}
	return false
}
