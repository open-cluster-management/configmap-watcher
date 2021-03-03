// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package watcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
)

func TestSplitNamespacedName(t *testing.T) {
	result := splitNamespacedName("default")
	assert.NotNil(t, result)
	assert.True(t, "default" == result.Name)
	assert.True(t, "" == result.Namespace)

	result = splitNamespacedName("default/name")
	assert.NotNil(t, result)
	assert.True(t, "name" == result.Name)
	assert.True(t, "default" == result.Namespace)

	result = splitNamespacedName("")
	assert.True(t, "" == result.Name)
	assert.True(t, "" == result.Namespace)
}

func TestPrint(t *testing.T) {
	var watchedConfigmaps map[types.NamespacedName]*ConfigMapper = make(map[types.NamespacedName]*ConfigMapper)
	var nn types.NamespacedName
	nn.Name = "test"
	nn.Namespace = "default"

	var cm ConfigMapper
	cm.stopCh = nil
	cm.Daemonsets = make(map[types.NamespacedName]uint)
	cm.Statefulsets = make(map[types.NamespacedName]uint)
	cm.Deployments = make(map[types.NamespacedName]uint)
	cm.Daemonsets[nn] = 1
	cm.Deployments[nn] = 1
	cm.Statefulsets[nn] = 1
	watchedConfigmaps[nn] = &cm

	print(watchedConfigmaps)
}

func TestRemoveStale(t *testing.T) {
	var watchedConfigmaps map[types.NamespacedName]*ConfigMapper = make(map[types.NamespacedName]*ConfigMapper)
	var nn types.NamespacedName
	nn.Name = "test"
	nn.Namespace = "default"

	var cm ConfigMapper
	cm.stopCh = nil
	cm.Daemonsets = make(map[types.NamespacedName]uint)
	cm.Statefulsets = make(map[types.NamespacedName]uint)
	cm.Deployments = make(map[types.NamespacedName]uint)
	cm.Daemonsets[nn] = 1
	cm.Deployments[nn] = 1
	cm.Statefulsets[nn] = 1
	watchedConfigmaps[nn] = &cm

	removeStale(0, watchedConfigmaps)
}
