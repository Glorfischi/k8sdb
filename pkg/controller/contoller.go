package controller

import (
	clientset "github.com/glorfischi/k8sdb/pkg/client/clientset/versioned"
	lister "github.com/glorfischi/k8sdb/pkg/client/listers/k8sdb/v1alpha1"
	"k8s.io/client-go/util/workqueue"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	v1aplha1 "github.com/glorfischi/k8sdb/pkg/client/informers/externalversions/k8sdb/v1alpha1"
	"k8s.io/client-go/tools/cache"
	"fmt"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
	"k8s.io/api/core/v1"
	"github.com/glorfischi/k8sdb/pkg/db"
	"k8s.io/client-go/kubernetes"
	"github.com/golang/glog"
	"k8s.io/client-go/rest"
)

const (
	dbFinalizer = "fischi.me/db"
	retry       = 15
)

type eventType int

const (
	add    eventType = iota
	update
	delete
)

type event struct {
	eventType eventType
	oldKey    string
	newKey    string
}

type Controller struct {
	dbClientset   clientset.Interface
	kubeClientset *kubernetes.Clientset
	dbLister      lister.DatabaseLister
	workqueue     workqueue.RateLimitingInterface
	informer      cache.SharedIndexInformer
	dbServers     map[string]db.DatabaseServer
}

func NewController(cfg *rest.Config, dbServers map[string]db.DatabaseServer) (*Controller, error) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	dbClientset, err := clientset.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	kubeClientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	informer := v1aplha1.NewDatabaseInformer(dbClientset, v1.NamespaceAll, 0, cache.Indexers{})
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err != nil {
				return
			}
			ev := event{eventType: add, newKey: key}
			queue.Add(ev)

		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newkey, err := cache.MetaNamespaceKeyFunc(newObj)
			if err != nil {
				return
			}
			oldkey, err := cache.DeletionHandlingMetaNamespaceKeyFunc(oldObj)
			if err != nil {
				return
			}
			ev := event{eventType: update, oldKey: oldkey, newKey: newkey}
			queue.Add(ev)
		},
	})

	return &Controller{dbClientset: dbClientset, kubeClientset: kubeClientset,
		workqueue: queue, informer: informer, dbServers: dbServers}, nil
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		return fmt.Errorf("timed out waiting for caches to sync")
	}

	wait.Until(c.runWorker, time.Second, stopCh)
	return nil
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	obj, quit := c.workqueue.Get()

	if quit {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)

		// We expect events to come off the workqueue.
		ev, ok := obj.(event)
		if !ok {
			return nil
		}
		if err := c.handleEvent(ev); err != nil {
			return err
		}
		return nil
	}(obj)

	if err == nil {
		c.workqueue.Forget(obj)
		return true
	}

	// This controller retries n times if something goes wrong. After that, it stops trying.
	if c.workqueue.NumRequeues(obj) < retry {
		glog.Errorf("error handling db. Retrying %v %v", obj, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.workqueue.AddRateLimited(obj)
		return true
	}

	glog.Errorf("error handling db. Dropping out of queue %v", obj)
	return true
}
