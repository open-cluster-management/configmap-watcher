// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2019. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package watcher

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	watcherAnnotation string = "watcher.ibm.com/configmap-resource"
	restartLabel      string = "watcher.ibm.com/restart-time"
	optInLabel        string = "watcher.ibm.com/opt-in=true"
)

var watchedConfigmaps map[types.NamespacedName]*ConfigMapper = make(map[types.NamespacedName]*ConfigMapper)
var listOptions metav1.ListOptions = metav1.ListOptions{LabelSelector: optInLabel}
var allowedNamespaces map[string]struct{}
var storedCounter uint = 0
var clean uint = 0
var restrictNamespaces bool

type ConfigMapper struct {
	stopCh       *chan struct{}
	Deployments  map[types.NamespacedName]uint
	Daemonsets   map[types.NamespacedName]uint
	Statefulsets map[types.NamespacedName]uint
	Mark         uint
}

type WatcherController struct {
	client *kubernetes.Clientset
}

func Init(cl *kubernetes.Clientset, allowed map[string]struct{}, cleanFreq uint, restrict bool) *WatcherController {
	klog.V(4).Info("Initializing watcher controller.")
	allowedNamespaces = allowed
	clean = cleanFreq
	restrictNamespaces = restrict
	return &WatcherController{
		client: cl,
	}
}

// Creates the Informer for the configmap specified in order to watch it.
func (w *WatcherController) createInformer(configmap types.NamespacedName, stopCh *chan struct{}) {
	klog.V(5).Infof("Creating informer for %s", configmap.String())
	informerFactory := informers.NewSharedInformerFactoryWithOptions(w.client, 0, informers.WithNamespace(configmap.Namespace),
		informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
			opts.FieldSelector = fmt.Sprintf("metadata.name=%s", configmap.Name)
		}))

	informer := informerFactory.Core().V1().ConfigMaps().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(old interface{}, new interface{}) {
			klog.V(5).Infof("Update to configmap %s/%s occurred, restarting all pods watching it.", new.(*corev1.ConfigMap).ObjectMeta.Namespace, new.(*corev1.ConfigMap).ObjectMeta.Name)
			RestartAll(w.client, configmap, watchedConfigmaps)
		},
	})
	klog.V(2).Infof("Starting informer for %s", configmap.String())
	go informer.Run(*stopCh)
}

// GatherConfigMaps - periodically gathers configmaps specified by any deployment, daemonset, and/or statefulset
// that opts into this watcher
func (w *WatcherController) GatherConfigMaps(freq uint, stopCh <-chan struct{}) {
	storedCounter++
	klog.V(4).Infof("Gather configmaps counter: %d", storedCounter)

	// Query for deployments, daemonsets, and statefulsets that target this watcher
	deployments, _ := w.client.AppsV1().Deployments("").List(listOptions)
	daemonsets, _ := w.client.AppsV1().DaemonSets("").List(listOptions)
	statefulsets, _ := w.client.AppsV1().StatefulSets("").List(listOptions)

	klog.V(6).Infof("List of deployments found: %v\nList of daemonsets found: %v\nList of statefulsets found: %v", deployments, daemonsets, statefulsets)
NEXT_DEPLOYMENT:
	// Check for the configmap watched by each
	for _, deployment := range deployments.Items {
		// If we're restricting the namespaces allowed and the namespace this deployment is in is not allowed, we ignore it
		if _, ok := allowedNamespaces[deployment.ObjectMeta.Namespace]; restrictNamespaces && !ok {
			klog.V(5).Infof("Ignoring deployment %s/%s since it's not in an allowed namespace.", deployment.ObjectMeta.Namespace, deployment.ObjectMeta.Name)
			continue NEXT_DEPLOYMENT
		}
		klog.Infof("Found deployment opting in: %s", deployment.ObjectMeta.Name)
		if _, ok := deployment.ObjectMeta.Annotations[watcherAnnotation]; ok {
			klog.V(5).Info("Deployment has the watcher annotation")
			// If the deployment has the annotation, get the namespace/name of the configmap
			configmapName := splitNamespacedName(deployment.ObjectMeta.Annotations[watcherAnnotation])
			klog.Infof("The configmap specified by this deployment %s", configmapName.String())
			_, err := w.client.CoreV1().ConfigMaps(configmapName.Namespace).Get(configmapName.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("ERROR: %s: unable to get configmap; invalid name/namespace for configmap [%s] or error with contacting the server", err.Error(), configmapName.String())
				continue NEXT_DEPLOYMENT
			}
			if _, ok := watchedConfigmaps[configmapName]; !ok {
				klog.V(3).Infof("Configmap doesn't exist in list yet, adding it %s and deployment %s", configmapName.String(), deployment.ObjectMeta.Name)
				storedDeployments := make(map[types.NamespacedName]uint)
				storedDeployments[types.NamespacedName{Name: deployment.ObjectMeta.Name, Namespace: deployment.ObjectMeta.Namespace}] = storedCounter
				stopCh := make(chan struct{})
				watchedConfigmaps[configmapName] = &ConfigMapper{stopCh: &stopCh, Deployments: storedDeployments, Mark: storedCounter}
				// Create a watcher informer for it
				w.createInformer(configmapName, &stopCh)
			} else {
				klog.V(3).Info("Configmap already in list to watch, updating associated deployment counter.")
				if watchedConfigmaps[configmapName].Deployments == nil {
					storedDeployments := make(map[types.NamespacedName]uint)
					storedDeployments[types.NamespacedName{Name: deployment.ObjectMeta.Name, Namespace: deployment.ObjectMeta.Namespace}] = storedCounter
					watchedConfigmaps[configmapName].Deployments = storedDeployments
				} else {
					watchedConfigmaps[configmapName].Deployments[types.NamespacedName{Name: deployment.ObjectMeta.Name, Namespace: deployment.ObjectMeta.Namespace}] = storedCounter
				}
			}
			watchedConfigmaps[configmapName].Mark = storedCounter
		}
	}
NEXT_DAEMONSET:
	// Check for the configmap watched by each
	for _, daemonset := range daemonsets.Items {
		// If we're restricting the namespaces allowed and the namespace this deployment is in is not allowed, we ignore it
		if _, ok := allowedNamespaces[daemonset.ObjectMeta.Namespace]; restrictNamespaces && !ok {
			klog.V(5).Infof("Ignoring daemonset %s/%s since it's not in an allowed namespace.", daemonset.ObjectMeta.Namespace, daemonset.ObjectMeta.Name)
			continue NEXT_DAEMONSET
		}
		klog.Infof("Found daemonset opting in: %s", daemonset.ObjectMeta.Name)
		if _, ok := daemonset.ObjectMeta.Annotations[watcherAnnotation]; ok {
			// If the daemonset has the annotation, get the namespace/name of the configmap
			configmapName := splitNamespacedName(daemonset.ObjectMeta.Annotations[watcherAnnotation])
			klog.Infof("The configmap specified by this daemonset %s", configmapName.String())
			_, err := w.client.CoreV1().ConfigMaps(configmapName.Namespace).Get(configmapName.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Unable to get configmap; invalid name/namespace for configmap or error with contacting the server, error: %s, configmap name specified in annotation %s", err.Error(), configmapName)
				continue NEXT_DAEMONSET
			}
			if _, ok := watchedConfigmaps[configmapName]; !ok {
				klog.V(3).Infof("Configmap doesn't exist in list yet, adding it %s and daemonset %s", configmapName.String(), daemonset.ObjectMeta.Name)
				storedDaemonset := make(map[types.NamespacedName]uint)
				storedDaemonset[types.NamespacedName{Name: daemonset.ObjectMeta.Name, Namespace: daemonset.ObjectMeta.Namespace}] = storedCounter
				stopCh := make(chan struct{})
				watchedConfigmaps[configmapName] = &ConfigMapper{stopCh: &stopCh, Daemonsets: storedDaemonset, Mark: storedCounter}
				// Create a watcher informer for it
				w.createInformer(configmapName, &stopCh)
			} else {
				klog.V(3).Info("Configmap already in list to watch, updating daemonset counter")
				if watchedConfigmaps[configmapName].Daemonsets == nil {
					storedDaemonset := make(map[types.NamespacedName]uint)
					storedDaemonset[types.NamespacedName{Name: daemonset.ObjectMeta.Name, Namespace: daemonset.ObjectMeta.Namespace}] = storedCounter
					watchedConfigmaps[configmapName].Daemonsets = storedDaemonset
				} else {
					watchedConfigmaps[configmapName].Daemonsets[types.NamespacedName{Name: daemonset.ObjectMeta.Name, Namespace: daemonset.ObjectMeta.Namespace}] = storedCounter
				}
			}
			watchedConfigmaps[configmapName].Mark = storedCounter
		}
	}
NEXT_STATEFULSET:
	// Check for the configmap watched by each
	for _, statefulset := range statefulsets.Items {
		// If we're restricting the namespaces allowed and the namespace this statefulset is in is not allowed, we ignore it
		if _, ok := allowedNamespaces[statefulset.ObjectMeta.Namespace]; restrictNamespaces && !ok {
			klog.V(5).Infof("Ignoring statefulset %s/%s since it's not in an allowed namespace.", statefulset.ObjectMeta.Namespace, statefulset.ObjectMeta.Name)
			continue NEXT_STATEFULSET
		}
		klog.Infof("Found statefulset opting in: %s", statefulset.ObjectMeta.Name)
		if _, ok := statefulset.ObjectMeta.Annotations[watcherAnnotation]; ok {
			// If the statefulset has the annotation, get the namespace/name of the configmap
			configmapName := splitNamespacedName(statefulset.ObjectMeta.Annotations[watcherAnnotation])
			klog.V(3).Infof("The configmap specified by this statefulset %s", configmapName.String())
			_, err := w.client.CoreV1().ConfigMaps(configmapName.Namespace).Get(configmapName.Name, metav1.GetOptions{})
			if err != nil {
				klog.Errorf("Unable to get configmap; invalid name/namespace for configmap or error with contacting the server, error: %s, configmap name specified: %s", err.Error(), configmapName)
				continue NEXT_STATEFULSET
			}
			if _, ok := watchedConfigmaps[configmapName]; !ok {
				klog.V(3).Infof("Configmap doesn't exist in list yet, adding it %s and statefulset %s", configmapName.String(), statefulset.ObjectMeta.Name)
				storedStatefulset := make(map[types.NamespacedName]uint)
				storedStatefulset[types.NamespacedName{Name: statefulset.ObjectMeta.Name, Namespace: statefulset.ObjectMeta.Namespace}] = storedCounter
				stopCh := make(chan struct{})
				watchedConfigmaps[configmapName] = &ConfigMapper{stopCh: &stopCh, Statefulsets: storedStatefulset, Mark: storedCounter}
				w.createInformer(configmapName, &stopCh)
			} else {
				klog.V(3).Info("Configmap already in list to watch, updating counter on statefulset")
				if watchedConfigmaps[configmapName].Statefulsets == nil {
					storedStatefulset := make(map[types.NamespacedName]uint)
					storedStatefulset[types.NamespacedName{Name: statefulset.ObjectMeta.Name, Namespace: statefulset.ObjectMeta.Namespace}] = storedCounter
					watchedConfigmaps[configmapName].Statefulsets = storedStatefulset
				} else {
					watchedConfigmaps[configmapName].Statefulsets[types.NamespacedName{Name: statefulset.ObjectMeta.Name, Namespace: statefulset.ObjectMeta.Namespace}] = storedCounter
				}
			}
			watchedConfigmaps[configmapName].Mark = storedCounter
		}
	}

	// Garbage collection
	if (storedCounter % clean) == 0 {
		klog.V(2).Info("Stored counter has reach clean count, removing stale resources.")
		removeStale(storedCounter, watchedConfigmaps)

		if storedCounter/clean == 2 { // Only resetting once it reaches double the clean frequency allows resources that were removed on the clean frequency to get removed
			storedCounter = 0
			klog.V(4).Info("Completely reset counter.")
		}
	}
	time.Sleep(time.Duration(freq) * time.Second)
}
