package main

import (
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"log"

	"gok8s/cache"
	"gok8s/client/config"
	"gok8s/controller"
	"gok8s/event"
	"gok8s/handler"
	"gok8s/predicate"
)

type dumbEventHandler struct {
	podCreateEvent      int
	podUpdateEventCount int
	podDeleteEventCount int
}

func (d *dumbEventHandler) OnCreate(e event.CreateEvent) (handler.Result, error) {
	log.Printf("create kind [%v] with name [%s]\n", e.Object.GetObjectKind(), e.Meta.GetName())
	d.podCreateEvent += 1
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnUpdate(e event.UpdateEvent) (handler.Result, error) {
	log.Printf("update kind [%v] with name [%s]\n", e.ObjectOld.GetObjectKind(), e.MetaOld.GetName())
	d.podUpdateEventCount += 1
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnDelete(e event.DeleteEvent) (handler.Result, error) {
	log.Printf("delete kind [%v] with name [%s]\n", e.Object.GetObjectKind(), e.Meta.GetName())
	d.podDeleteEventCount += 1
	return handler.Result{}, nil
}

func (d *dumbEventHandler) OnGeneric(e event.GenericEvent) (handler.Result, error) {
	return handler.Result{}, nil
}

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Panic(fmt.Sprintf("get config failed:%v\n", err))
	}

	stop := make(chan struct{})
	defer close(stop)

	c, err := cache.New(cfg, cache.Options{})
	if err != nil {
		log.Panic(fmt.Sprintf("create cache failed %v\n", err))
	}
	go c.Start(stop)

	c.WaitForCacheSync(stop)

	ctrl := controller.New("dumbController", c, scheme.Scheme)
	ctrl.Watch(&corev1.Pod{})
	handler := &dumbEventHandler{}
	ctrl.Start(stop, handler, predicate.NewIgnoreUnchangedUpdate())
}
