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

	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

var err error

func TestReconcile(t *testing.T) {

	var simpleClient kubernetes.Interface = testclient.NewSimpleClientset()
	newlbl := make(map[string]string)
	newlbl[watcherAnnotation] = "true"
	deployment.Labels = newlbl
	daemonset.Labels = newlbl
	statefulset.Labels = newlbl
	simpleClient.AppsV1().Deployments("default").Create(&deployment)
	simpleClient.AppsV1().DaemonSets("default").Create(&daemonset)
	simpleClient.AppsV1().StatefulSets("default").Create(&statefulset)
	watcher := WatcherController{
		client: simpleClient,
	}
	Init(simpleClient, nil, 1, true)
	watcher.GatherConfigMaps(1)

}
