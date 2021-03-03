// Copyright Contributors to the Open Cluster Management project

package watcher

import (
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
)

// splitNamespacedName turns the string form of a namespaced name
// (<namespace>/<name>) back into a types.NamespacedName.
func splitNamespacedName(nameStr string) types.NamespacedName {
	splitPoint := strings.IndexRune(nameStr, types.Separator)
	if splitPoint == -1 {
		return types.NamespacedName{Name: nameStr}
	}
	return types.NamespacedName{Namespace: nameStr[:splitPoint], Name: nameStr[splitPoint+1:]}
}

// print is for debugging purposes, it prints out the current list of configmaps being watched
// as well as the deployments, daemonsets, and statefulsets that specify them.
func print(watchedConfigmaps map[types.NamespacedName]*ConfigMapper) {
	if klog.V(5) {
		klog.Infof("Watched configmaps: %v", watchedConfigmaps)
		for name, mapper := range watchedConfigmaps {
			klog.Infof("configmap %s count: %d", name.String(), mapper.Mark)
			klog.Info("Deployments: ")
			for dName, deployment := range mapper.Deployments {
				klog.Infof("[%s] count: %d", dName, deployment)
			}
			klog.Info("Daemonsets: ")
			for dName, daemonset := range mapper.Daemonsets {
				klog.Infof("[%s] count: %d", dName, daemonset)
			}
			klog.Info("Statefulsets: ")
			for sName, statefulset := range mapper.Statefulsets {
				klog.Infof("[%s] count: %d", sName, statefulset)
			}
		}
	}
}

// RemoveStale is a garbage collector, it'll loop through all configmaps being watched and remove the ones that don't have
// the same value for their Mark as the counter passed in. Same for deployments/daemonsets/statefulsets.
func removeStale(count uint, watchedConfigmaps map[types.NamespacedName]*ConfigMapper) {
	// Loop through the config maps
	for name, mapper := range watchedConfigmaps {
		if mapper.Mark != count { // Hasn't been updated - delete it
			klog.V(2).Infof("Removing configmap %s since its count is %d but the current stored counter is %d", name, mapper.Mark, count)
			if _, ok := watchedConfigmaps[name]; ok {
				delete(watchedConfigmaps, name)
				close(*mapper.stopCh)
			}
		} else { // Loop through all of its associated objects
			// Deployments
			for deploymentName, deploymentCount := range watchedConfigmaps[name].Deployments {
				if deploymentCount != count { // Hasn't been found recently - remove it
					klog.V(2).Infof("Removing deployment %s since its count is %d but the current stored counter is %d", deploymentName, deploymentCount, count)
					if _, ok := watchedConfigmaps[name].Deployments[deploymentName]; ok {
						delete(watchedConfigmaps[name].Deployments, deploymentName)
					}
				}
			}
			// Daemonsets
			for daemonsetName, daemonsetCount := range watchedConfigmaps[name].Daemonsets {
				if daemonsetCount != count { // Hasn't been found recently - remove it
					klog.V(2).Infof("Removing daemonset %s since its count is %d but the current stored counter is %d", daemonsetName, daemonsetCount, count)
					if _, ok := watchedConfigmaps[name].Daemonsets[daemonsetName]; ok {
						delete(watchedConfigmaps[name].Daemonsets, daemonsetName)
					}
				}
			}
			// Statefulsets
			for statefulsetName, statefulsetCount := range watchedConfigmaps[name].Statefulsets {
				if statefulsetCount != count { // Hasn't been found recently - remove it
					klog.V(2).Infof("Removing statefulset %s since its count is %d but the current stored counter is %d", statefulsetName, statefulsetCount, count)
					if _, ok := watchedConfigmaps[name].Statefulsets[statefulsetName]; ok {
						delete(watchedConfigmaps[name].Statefulsets, statefulsetName)
					}
				}
			}
		}
	}
	klog.V(5).Info("Finished removing stale resources")
	print(watchedConfigmaps)
}
