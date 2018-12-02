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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
)

func newKubernetesConfig() KubernetesConfig {
	kc := KubernetesConfig{
		ConfigPath: "",
		clientset:  fake.NewSimpleClientset(),
	}
	kc.informers = informers.NewSharedInformerFactory(kc.clientset, 0)
	return kc
}

func TestWatchPodsAdd(t *testing.T) {
	// Configure the pod watcher
	kc := newKubernetesConfig()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	podChannel := kc.WatchPods(ctx, cancel)
	assert.NotNil(t, podChannel)

	// Inject an event into the fake client
	p := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "test-pod"}}
	_, err := kc.clientset.Core().Pods("namespace").Create(p)
	assert.Nil(t, err)

	// Ensure that the event is delivered to our pod subscriber's channel
	select {
	case podEvent := <-podChannel:
		assert.Equal(t, podEvent.Type, Add)
		assert.Equal(t, podEvent.Pod.Name, "test-pod")
		assert.Equal(t, podEvent.Pod.Namespace, "namespace")
	case <-time.After(50 * time.Millisecond):
		assert.Fail(t, "Informer did not receive the added pod in 50ms")
	}
}

func TestWatchServicesAdd(t *testing.T) {
	// Configure the services watcher
	kc := newKubernetesConfig()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	serviceChannel := kc.WatchServices(ctx, cancel)
	assert.NotNil(t, serviceChannel)

	// Inject an event into the fake client
	s := &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: "test-service"}}
	_, err := kc.clientset.Core().Services("namespace").Create(s)
	assert.Nil(t, err)

	// Ensure that the event is delivered to our service subscriber's channel
	select {
	case serviceEvent := <-serviceChannel:
		assert.Equal(t, serviceEvent.Type, Add)
		assert.Equal(t, serviceEvent.Service.Name, "test-service")
		assert.Equal(t, serviceEvent.Service.Namespace, "namespace")
	case <-time.After(50 * time.Millisecond):
		assert.Fail(t, "Informer did not receive the added service in 50ms")
	}
}
