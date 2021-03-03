// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package watcher

import (
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	testclient "k8s.io/client-go/kubernetes/fake"
)

func TestRestartAll(t *testing.T) {

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

	configmap.Labels = newlbl
	simpleClient.CoreV1().ConfigMaps("default").Update(&configmap)

	var watchedConfigmaps map[types.NamespacedName]*ConfigMapper = make(map[types.NamespacedName]*ConfigMapper)
	var cnn types.NamespacedName = splitNamespacedName("default/configmap")

	var cm ConfigMapper
	cm.stopCh = nil
	cm.Daemonsets = make(map[types.NamespacedName]uint)
	cm.Statefulsets = make(map[types.NamespacedName]uint)
	cm.Deployments = make(map[types.NamespacedName]uint)
	var dnn types.NamespacedName = splitNamespacedName("default/daemonset")
	cm.Daemonsets[dnn] = 1
	dnn = splitNamespacedName("default/deployment")
	cm.Deployments[dnn] = 1
	dnn = splitNamespacedName("default/statefulset")
	cm.Statefulsets[dnn] = 1
	watchedConfigmaps[cnn] = &cm

	RestartAll(simpleClient, cnn, watchedConfigmaps)
}
