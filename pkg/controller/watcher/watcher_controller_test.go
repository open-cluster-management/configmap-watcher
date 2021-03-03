// Copyright Contributors to the Open Cluster Management project

// Copyright 2019 The Kubernetes Authors.
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
package watcher

import (
	"testing"
	"time"

	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var err error

func TestReconcile(t *testing.T) {

	var simpleClient kubernetes.Interface = testclient.NewSimpleClientset()

	// Create the opt in label
	newlbl := make(map[string]string)
	newlbl["watcher.ibm.com/opt-in"] = "true"
	deployment.Labels = newlbl
	daemonset.Labels = newlbl
	statefulset.Labels = newlbl

	// create the config map to watch annotation
	newannot := make(map[string]string)
	newannot[watcherAnnotation] = "default/configmap"
	deployment.Annotations = newannot
	daemonset.Annotations = newannot
	statefulset.Annotations = newannot

	simpleClient.CoreV1().ConfigMaps("default").Create(&configmap)
	simpleClient.AppsV1().Deployments("default").Create(&deployment)
	simpleClient.AppsV1().DaemonSets("default").Create(&daemonset)
	simpleClient.AppsV1().StatefulSets("default").Create(&statefulset)
	watcher := WatcherController{
		client: simpleClient,
	}
	Init(simpleClient, nil, 1, true)
	watcher.GatherConfigMaps(1)

	// sleep for a bit
	time.Sleep(time.Second * 2)
	configmap.Labels = newlbl
	simpleClient.CoreV1().ConfigMaps("default").Update(&configmap)

	time.Sleep(time.Second * 2)
	watcher.GatherConfigMaps(1)
}
