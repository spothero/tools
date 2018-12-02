// Copyright 2018 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesConfig defines necessary configuration for interaction with Kubernetes
type KubernetesConfig struct {
	ConfigPath string
	clientset  kubernetes.Interface
	informers  informers.SharedInformerFactory
	ctx        context.Context
}

// EventType Defines the type of event received from Kubernetes
type EventType int

const (
	// Add means the resource was added
	Add EventType = iota
	// Update means the resource was updated
	Update
	// Delete means the resource was deleted
	Delete
)

// PodEvent defines a Kubernetes Pod Event in the cluster
type PodEvent struct {
	Pod    *v1.Pod
	OldPod *v1.Pod
	Type   EventType
}

// ServiceEvent defines a Kubernetes Service Event in the cluster
type ServiceEvent struct {
	Service    *v1.Service
	OldService *v1.Service
	Type       EventType
}

// Init initializes the Kubernetes cluster connection for use. If running in-cluster, users are
// responsible for ensuring that their deployment offers the correct RBAC permissions for their
// Kubernetes client to access whatever portion of the API they are trying to interact with.
// See: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
// Likewise, your personal kubeconfig must have high-enough permissions for the namespace you are
// interacting with to leverage the outside-of-cluster configuration.
func (kc *KubernetesConfig) Init() error {
	// Use the current context set in the local machine's kubeconfig directory
	if kc.ConfigPath != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kc.ConfigPath)
		if err != nil {
			Logger.Error(
				"Unable to create Kubernetes Config from file",
				zap.Error(err))
			return err
		}
		kc.clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			Logger.Error(
				"Unable to instantiate Kubernetes Client outside of cluster",
				zap.Error(err))
			return err
		}
	} else {
		// Use the in-cluster config -- requires correct RBAC permissions for
		// whatever resources the user intends to access
		config, err := rest.InClusterConfig()
		if err != nil {
			Logger.Error(
				"Unable to create Kubernetes Config in cluster",
				zap.Error(err))
			return err
		}
		kc.clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			Logger.Error(
				"Unable to instantiate Kubernetes Client in cluster",
				zap.Error(err))
			return err
		}
	}

	// Configure informers
	kc.informers = informers.NewSharedInformerFactory(kc.clientset, 0)
	return nil
}

// WatchPods creates a Pod event handler channel and returns the changes to the caller. Callers
// may subscribe to the Pods channel to watch for changes within Kubernetes.
func (kc *KubernetesConfig) WatchPods(ctx context.Context, cancel context.CancelFunc) chan PodEvent {
	pods := make(chan PodEvent, 100)
	podInformer := kc.informers.Core().V1().Pods().Informer()
	podInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			Logger.Debug("Pod Added",
				zap.String("namespace", pod.Namespace),
				zap.String("name", pod.Name))
			pods <- PodEvent{Pod: pod, Type: Add}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			pod := newObj.(*v1.Pod)
			Logger.Debug("Pod Updated",
				zap.String("namespace", pod.Namespace),
				zap.String("name", pod.Name))
			pods <- PodEvent{Pod: pod, OldPod: oldObj.(*v1.Pod), Type: Update}
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*v1.Pod)
			Logger.Debug("Pod Deleted",
				zap.String("namespace", pod.Namespace),
				zap.String("name", pod.Name))
			pods <- PodEvent{Pod: pod, Type: Delete}
		},
	})
	kc.informers.Start(ctx.Done())
	return pods
}

// WatchServices creates a Service event handler channel and returns the changes to the caller. Callers
// may subscribe to the Services channel to watch for changes within Kubernetes.
func (kc *KubernetesConfig) WatchServices(ctx context.Context, cancel context.CancelFunc) chan ServiceEvent {
	services := make(chan ServiceEvent, 100)
	serviceInformer := kc.informers.Core().V1().Services().Informer()
	serviceInformer.AddEventHandler(&cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			Logger.Debug("Service Added",
				zap.String("namespace", service.Namespace),
				zap.String("name", service.Name))
			services <- ServiceEvent{Service: service, Type: Add}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			service := newObj.(*v1.Service)
			Logger.Debug("Service Updated",
				zap.String("namespace", service.Namespace),
				zap.String("name", service.Name))
			services <- ServiceEvent{Service: service, OldService: oldObj.(*v1.Service), Type: Update}
		},
		DeleteFunc: func(obj interface{}) {
			service := obj.(*v1.Service)
			Logger.Debug("Service Deleted",
				zap.String("namespace", service.Namespace),
				zap.String("name", service.Name))
			services <- ServiceEvent{Service: service, Type: Delete}
		},
	})
	kc.informers.Start(ctx.Done())
	return services
}
