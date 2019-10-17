// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2019. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package main

import (
	"flag"
	"os"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	watcherController "github.ibm.com/IBMPrivateCloud/configmap-watcher/pkg/controller/watcher"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	var gatherFreq, checkConfigmapFreq, cleanFreq uint
	flag.UintVar(&gatherFreq, "gather-frequency", 20, "How frequently (in seconds) to gather configmaps from kubernetes deployments/daemonsets/statefulsets")
	flag.UintVar(&checkConfigmapFreq, "check-configmap-frequency", 60, "How frequently (in seconds) to check the configmaps for changes.")
	flag.UintVar(&cleanFreq, "clean-frequency", 100, "How frequently (in count) we want to clean up stale resources.")
	flag.Set("logtostderr", "true")

	flag.Parse()

	klog.V(11).Info("In main. Starting now")
	stopCh := ctrl.SetupSignalHandler()
	klog.V(11).Info("Getting the kubeconfig...")
	// Get the kube config
	cfg, err := config.GetConfig()
	if err != nil {
		klog.Error(err, "Unable to get kube config")
		os.Exit(1)
	}

	klog.V(11).Info("Got kube config, getting client")
	// Get kubernetes client based on config
	kubeClient := kubernetes.NewForConfigOrDie(cfg)
	watcher := watcherController.Init(kubeClient, cleanFreq)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		klog.V(11).Info("starting gather configmap")
		for {
			watcher.GatherConfigMaps(gatherFreq, stopCh)
			klog.V(11).Info("Back in for loop before another call gather configs")
		}
		klog.V(11).Info("Exited worker configmap watcher")
	}()
	go func() {
		defer wg.Done()
		klog.V(11).Info("starting check for configmap changes")
		for {
			watcher.CheckConfigMapsForChanges(checkConfigmapFreq, stopCh)
			klog.V(11).Info("Back in for loop before another call")
		}
		klog.V(11).Info("Exited configmap checker")
	}()
	klog.V(11).Info("Outside the go functions")
	wg.Wait()
	klog.V(11).Info("After the wait function")
}
