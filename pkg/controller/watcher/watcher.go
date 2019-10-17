// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2019. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package watcher

import (
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

var listOptions metav1.ListOptions = metav1.ListOptions{LabelSelector: "watcher.ibm.com/opt-in=true"}
var watcherAnnotation string = "watcher.ibm.com/configmap-resource"
var watchedConfigmaps map[types.NamespacedName]*ConfigMapper = make(map[types.NamespacedName]*ConfigMapper)
var storedCounter uint = 0
var clean uint = 0

type ConfigMapper struct {
	Configmap    corev1.ConfigMap
	Deployments  map[types.NamespacedName]*AssociatedObject
	Daemonsets   map[types.NamespacedName]*AssociatedObject
	Statefulsets map[types.NamespacedName]*AssociatedObject
	Mark         uint
}

type AssociatedObject struct {
	Restarter restartFunc
	Mark      uint
}

// Restart func is a function that takes an interface and the object it needs to restart
type restartFunc func(types.NamespacedName) error
type WatcherController struct {
	client *kubernetes.Clientset
}

func Init(cl *kubernetes.Clientset, cleanFreq uint) *WatcherController {
	clean = cleanFreq
	return &WatcherController{
		client: cl,
	}
}

// GatherConfigMaps - periodically gathers configmaps specified by any deployment, daemonset, and/or statefulset
// that opts into this watcher
func (w *WatcherController) GatherConfigMaps(freq uint, stopCh <-chan struct{}) {
	time.Sleep(time.Duration(freq) * time.Second)
	storedCounter++
	klog.V(4).Infof("Gather configmaps counter: %d", storedCounter)
	// Query for deployments, daemonsets, and statefulsets that target this watcher
	deployments, _ := w.client.AppsV1().Deployments("").List(listOptions)
	daemonsets, _ := w.client.AppsV1().DaemonSets("").List(listOptions)
	statefulsets, _ := w.client.AppsV1().StatefulSets("").List(listOptions)
	klog.V(10).Infof("List of deployments found: %v\nList of daemonsets found: %v\nList of statefulsets found: %v", deployments, daemonsets, statefulsets)
NEXT_DEPLOYMENT:
	// Check for the configmap watched by each by querying their annotations
	for _, deployment := range deployments.Items {
		klog.V(4).Infof("Checking deployment: %s", deployment.ObjectMeta.Name)
		if _, ok := deployment.ObjectMeta.Annotations[watcherAnnotation]; ok {
			klog.V(5).Info("Deployment has the watcher annotation")
			// If the deployment has the annotation, get the namespace/name of the configmap
			configmapName := splitNamespacedName(deployment.ObjectMeta.Annotations[watcherAnnotation])
			klog.V(5).Infof("The configmap name %s", configmapName.String())
			configmap, err := w.client.CoreV1().ConfigMaps(configmapName.Namespace).Get(configmapName.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("ERROR: %s: unable to get configmap; invalid name/namespace for configmap [%s] or error with contacting the server", err.Error(), configmapName.String())
				continue NEXT_DEPLOYMENT
			}
			if _, ok := watchedConfigmaps[configmapName]; !ok {
				klog.V(5).Infof("Configmap doesn't exist in list yet, adding it %s and deployment %s", configmapName.String(), deployment.ObjectMeta.Name)
				funcs := make(map[types.NamespacedName]*AssociatedObject)
				funcs[types.NamespacedName{Name: deployment.ObjectMeta.Name, Namespace: deployment.ObjectMeta.Namespace}] = &AssociatedObject{Restarter: restartDeployment(w.client), Mark: storedCounter}
				watchedConfigmaps[configmapName] = &ConfigMapper{Configmap: *configmap, Deployments: funcs, Mark: storedCounter}
			} else {
				klog.V(5).Info("Configmap already in list to watch")
				// Add this to the map if it doesn't exist
				deploymentObj, exists := watchedConfigmaps[configmapName].Deployments[types.NamespacedName{Name: deployment.ObjectMeta.Name, Namespace: deployment.ObjectMeta.Namespace}]
				if !exists {
					klog.V(5).Infof("This deployment does not exist in list to restart, adding %s to list", deployment.ObjectMeta.Name)
					watchedConfigmaps[configmapName].Deployments[types.NamespacedName{Name: deployment.ObjectMeta.Name, Namespace: deployment.ObjectMeta.Namespace}] = &AssociatedObject{Restarter: restartDeployment(w.client), Mark: storedCounter}
				} else {
					deploymentObj.Mark = storedCounter
				}
			}
			watchedConfigmaps[configmapName].Mark = storedCounter
		}
	}
NEXT_DAEMONSET:
	// Check for the configmap watched by each by querying their annotations
	for _, daemonset := range daemonsets.Items {
		if _, ok := daemonset.ObjectMeta.Annotations[watcherAnnotation]; ok {
			// If the daemonset has the annotation, get the namespace/name of the configmap
			configmapName := splitNamespacedName(daemonset.ObjectMeta.Annotations[watcherAnnotation])
			configmap, err := w.client.CoreV1().ConfigMaps(configmapName.Namespace).Get(configmapName.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Unable to get configmap; invalid name/namespace for configmap or error with contacting the server, error: %s, configmap name specified in annotation %s", err.Error(), configmapName)
				continue NEXT_DAEMONSET
			}
			if _, ok := watchedConfigmaps[configmapName]; !ok {
				klog.V(5).Infof("Configmap doesn't exist in list yet, adding it %s and daemonset %s", configmapName.String(), daemonset.ObjectMeta.Name)
				funcs := make(map[types.NamespacedName]*AssociatedObject)
				funcs[types.NamespacedName{Name: daemonset.ObjectMeta.Name, Namespace: daemonset.ObjectMeta.Namespace}] = &AssociatedObject{Restarter: restartDaemonset(w.client), Mark: storedCounter}
				watchedConfigmaps[configmapName] = &ConfigMapper{Configmap: *configmap, Daemonsets: funcs, Mark: storedCounter}
			} else {
				klog.V(5).Info("Configmap already in list to watch")
				// Add this to the map if it doesn't exist
				daemonsetObj, exists := watchedConfigmaps[configmapName].Daemonsets[types.NamespacedName{Name: daemonset.ObjectMeta.Name, Namespace: daemonset.ObjectMeta.Namespace}]
				if !exists {
					klog.V(5).Infof("This daemonset does not exist in list to restart, adding %s to list", daemonset.ObjectMeta.Name)
					watchedConfigmaps[configmapName].Daemonsets[types.NamespacedName{Name: daemonset.ObjectMeta.Name, Namespace: daemonset.ObjectMeta.Namespace}] = &AssociatedObject{Restarter: restartDaemonset(w.client), Mark: storedCounter}
				} else {
					klog.V(5).Info("Updating the daemonset counter")
					daemonsetObj.Mark = storedCounter
				}
			}
			watchedConfigmaps[configmapName].Mark = storedCounter
		}
	}
NEXT_STATEFULSET:
	// Check for the configmap watched by each by querying their annotations
	for _, statefulset := range statefulsets.Items {
		if _, ok := statefulset.ObjectMeta.Annotations[watcherAnnotation]; ok {
			// If the statefulset has the annotation, get the namespace/name of the configmap
			configmapName := splitNamespacedName(statefulset.ObjectMeta.Annotations[watcherAnnotation])
			configmap, err := w.client.CoreV1().ConfigMaps(configmapName.Namespace).Get(configmapName.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Unable to get configmap; invalid name/namespace for configmap or error with contacting the server, error: %s, configmap name specified: %s", err.Error(), configmapName)
				continue NEXT_STATEFULSET
			}
			if _, ok := watchedConfigmaps[configmapName]; !ok {
				klog.V(5).Infof("Configmap doesn't exist in list yet, adding it %s and statefulset %s", configmapName.String(), statefulset.ObjectMeta.Name)
				funcs := make(map[types.NamespacedName]*AssociatedObject)
				funcs[types.NamespacedName{Name: statefulset.ObjectMeta.Name, Namespace: statefulset.ObjectMeta.Namespace}] = &AssociatedObject{Restarter: restartStatefulset(w.client), Mark: storedCounter}
				watchedConfigmaps[configmapName] = &ConfigMapper{Configmap: *configmap, Daemonsets: funcs}
			} else {
				klog.V(5).Info("Configmap already in list to watch")
				// Add this to the map if it doesn't exist
				statefulsetObj, exists := watchedConfigmaps[configmapName].Statefulsets[types.NamespacedName{Name: statefulset.ObjectMeta.Name, Namespace: statefulset.ObjectMeta.Namespace}]
				if !exists {
					klog.V(5).Infof("This daemonset does not exist in list to restart, adding %s to list", statefulset.ObjectMeta.Name)
					watchedConfigmaps[configmapName].Statefulsets[types.NamespacedName{Name: statefulset.ObjectMeta.Name, Namespace: statefulset.ObjectMeta.Namespace}] = &AssociatedObject{Restarter: restartStatefulset(w.client), Mark: storedCounter}
				} else {
					statefulsetObj.Mark = storedCounter
				}
			}
			watchedConfigmaps[configmapName].Mark = storedCounter
		}
	}

	if storedCounter == clean {
		klog.V(5).Info("Stored counter has reach clean count, removing stale resources.")
		removeStale(storedCounter)
		storedCounter = 0
	}
}

func removeStale(count uint) {
	// Loop through the config maps
	for name, mapper := range watchedConfigmaps {
		if mapper.Mark != count { // Hasn't been updated - delete it
			klog.V(5).Infof("Removing configmap %s since its count is %d but the current stored counter is %d", name, mapper.Mark, count)
			if _, ok := watchedConfigmaps[name]; ok {
				delete(watchedConfigmaps, name)
			}
		} else { // Loop through all of its associated objects
			// Deployments
			for deploymentName, deployment := range watchedConfigmaps[name].Deployments {
				if deployment.Mark != count { // Hasn't been found recently - remove it
					klog.V(4).Infof("Removing deployment %s since its count is %d but the current stored counter is %d", deploymentName, deployment.Mark, count)
					if _, ok := watchedConfigmaps[name].Deployments[deploymentName]; ok {
						delete(watchedConfigmaps[name].Deployments, deploymentName)
					}
				}
			}
			// Daemonsets
			for daemonsetName, daemonset := range watchedConfigmaps[name].Daemonsets {
				if daemonset.Mark != count { // Hasn't been found recently - remove it
					klog.V(4).Infof("Removing daemonset %s since its count is %d but the current stored counter is %d", daemonsetName, daemonset.Mark, count)
					if _, ok := watchedConfigmaps[name].Daemonsets[daemonsetName]; ok {
						delete(watchedConfigmaps[name].Daemonsets, daemonsetName)
					}
				}
			}
			// Deployments
			for statefulsetName, statefulset := range watchedConfigmaps[name].Statefulsets {
				if statefulset.Mark != count { // Hasn't been found recently - remove it
					klog.V(4).Infof("Removing statefulset %s since its count is %d but the current stored counter is %d", statefulsetName, statefulset.Mark, count)
					if _, ok := watchedConfigmaps[name].Statefulsets[statefulsetName]; ok {
						delete(watchedConfigmaps[name].Statefulsets, statefulsetName)
					}
				}
			}
		}
	}
	klog.V(5).Info("Finished removing stale resources")
	print()
}

// Periodically run this to check for changes
func (w *WatcherController) CheckConfigMapsForChanges(freq uint, stopCh <-chan struct{}) {
	time.Sleep(time.Duration(freq) * time.Second)
NEXT_CONFIGMAP:
	for configmapName, configMapper := range watchedConfigmaps {
		// Get the configmap as it exists out in the wild
		externalConfigmap, err := w.client.CoreV1().ConfigMaps(configmapName.Namespace).Get(configmapName.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Unable to get configmap; invalid name or error with contacting the server. Error message: %s, configmap name specified: %s", err.Error(), configmapName)
			continue NEXT_CONFIGMAP
		}
		// Compare - if changed, restart pod associated with it
		if !reflect.DeepEqual(configMapper.Configmap, *externalConfigmap) {
			klog.V(4).Infof("Configmap %s has changed, restarting all pods watching it", configMapper.Configmap.Name)
			klog.V(10).Infof("Original configmap: %v, New configmap: %v", configMapper.Configmap, *externalConfigmap)
			// Restart deployments
			for kind, associatedObj := range configMapper.Deployments {
				if err := associatedObj.Restarter(kind); err != nil {
					klog.Errorf("Unable to restart pods associated with deployment %s, error message: %s", kind.Name, err.Error())
				}
			}
			// Restart daemonsets
			for kind, associatedObj := range configMapper.Daemonsets {
				if err := associatedObj.Restarter(kind); err != nil {
					klog.Errorf("Unable to restart pods associated with daemonset %s, error message: %s", kind.Name, err.Error())
				}
			}
			// Restart statefulset
			for kind, associatedObj := range configMapper.Statefulsets {
				if err := associatedObj.Restarter(kind); err != nil {
					klog.Errorf("Unable to restart pods associated with statefulset %s, error message: %s", kind.Name, err.Error())
				}
			}
			// Store the new configmap
			watchedConfigmaps[configmapName].Configmap = *externalConfigmap
		}
	}
}

var restartLabel string = "watcher.ibm.com/restart-time"

// Restarts a deployment
func restartDeployment(client *kubernetes.Clientset) restartFunc {
	return func(deploymentName types.NamespacedName) error {
		update := time.Now().Format("2006-1-2.1504")
		klog.V(4).Infof("Restarting deployment %s at %s", deploymentName.String(), update)
		deploymentsInterface := client.AppsV1().Deployments(deploymentName.Namespace)
		deployment, err := deploymentsInterface.Get(deploymentName.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("error occurred getting deployment %v", deployment)
		}
		deployment.ObjectMeta.Labels[restartLabel] = update
		deployment.Spec.Template.ObjectMeta.Labels[restartLabel] = update
		_, err = deploymentsInterface.Update(deployment)
		if err != nil {
			klog.Errorf("Error updating deployment: %v", err)
			return err
		}
		return nil
	}
}

// Restarts a daemonset
func restartDaemonset(client *kubernetes.Clientset) restartFunc {
	return func(daemonsetName types.NamespacedName) error {
		update := time.Now().Format("2006-1-2.1504")
		klog.V(4).Infof("Restarting daemonset %s at %s", daemonsetName.String(), update)
		daemonsetInterface := client.AppsV1().DaemonSets(daemonsetName.Namespace)
		daemonset, err := daemonsetInterface.Get(daemonsetName.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error getting daemonset %v", daemonsetName)
		}
		daemonset.ObjectMeta.Labels[restartLabel] = update
		daemonset.Spec.Template.ObjectMeta.Labels[restartLabel] = update
		_, err = daemonsetInterface.Update(daemonset)
		if err != nil {
			klog.Errorf("Error updating daemonset: %v", err)
			return err
		}
		return nil
	}
}

// Restarts a statefulset
func restartStatefulset(client *kubernetes.Clientset) restartFunc {
	return func(statefulsetName types.NamespacedName) error {
		update := time.Now().Format("2006-1-2.1504")
		klog.V(4).Infof("Restarting statefulset %s at %s", statefulsetName.String(), update)
		statefulsetInterface := client.AppsV1().DaemonSets(statefulsetName.Namespace)
		statefulset, err := statefulsetInterface.Get(statefulsetName.Name, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("Error getting daemonset %v", statefulsetName)
		}
		statefulset.ObjectMeta.Labels[restartLabel] = update
		statefulset.Spec.Template.ObjectMeta.Labels[restartLabel] = update
		_, err = statefulsetInterface.Update(statefulset)
		if err != nil {
			klog.Errorf("Error updating statefulset: %v", err)
			return err
		}
		return nil
	}
}

func print() {
	klog.V(6).Infof("Watched configmaps: %v", watchedConfigmaps)
	for name, mapper := range watchedConfigmaps {
		klog.V(6).Infof("configmap %s count: %d", name.String(), mapper.Mark)
		klog.V(6).Info("Deployments: ")
		for dName, deployment := range mapper.Deployments {
			klog.V(6).Infof("[%s] count: %d", dName, deployment.Mark)
		}
		klog.V(6).Info("Daemonsets: ")
		for dName, daemonset := range mapper.Daemonsets {
			klog.V(6).Infof("[%s] count: %d", dName, daemonset.Mark)
		}
		klog.V(6).Info("Statefulsets: ")
		for sName, statefulset := range mapper.Statefulsets {
			klog.V(6).Infof("[%s] count: %d", sName, statefulset.Mark)
		}
	}
	klog.V(6).Info("")
}
