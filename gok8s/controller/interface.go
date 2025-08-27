package controller

import (
	"k8s.io/apimachinery/pkg/runtime"

	"gok8s/handler"
	"gok8s/predicate"
)

type Controller interface {
	Watch(obj runtime.Object) error
	Start(stop <-chan struct{}, handler handler.EventHandler, predicates ...predicate.Predicate)
}
