// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

// Licensed Materials - Property of IBM
// (c) Copyright IBM Corporation 2019. All Rights Reserved.
// Note to U.S. Government Users Restricted Rights:
// Use, duplication or disclosure restricted by GSA ADP Schedule
// Contract with IBM Corp.

package main

import (
	"flag"
	"os"
	"strings"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	watcherController "github.com/open-cluster-management/configmap-watcher/pkg/controller/watcher"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func main() {
	klog.InitFlags(nil)
	defer klog.Flush()

	var allowed map[string]struct{}
	var allowedNamespaces string
	var gatherFreq, cleanFreq uint
	var restrictNamespaces bool
	flag.StringVar(&allowedNamespaces, "allowed-namespaces", "", "Space-separated namespaces. Only the deployments/daemonsets/statefulsets in these namespaces are allowed to use this controller to watch configmaps and restart themselves when those configmaps change.")
	flag.UintVar(&gatherFreq, "gather-frequency", 20, "How frequently (in seconds) to gather configmaps from kubernetes deployments/daemonsets/statefulsets")
	flag.UintVar(&cleanFreq, "clean-frequency", 100, "How frequently (in count) we want to clean up stale resources.")
	flag.BoolVar(&restrictNamespaces, "restrict-namespaces", false, "If true, restricts which deployable is allowed to use this controller based on the allowed-namespaces flag.")
	flag.Set("logtostderr", "true") /* #nosec G104 */

	flag.Parse()

	// Adding every allowed namespace into the map
	allowed = make(map[string]struct{})
	namespaces := strings.Fields(allowedNamespaces)
	for _, namespace := range namespaces {
		allowed[namespace] = struct{}{}
	}
	klog.V(5).Infof("Allowed namespaces %v", allowed)

	klog.Info("In main. Starting now")

	klog.V(11).Info("Getting the kubeconfig...")
	// Get the kube config
	cfg, err := config.GetConfig()
	if err != nil {
		klog.Error(err, "Unable to get kube config")
		os.Exit(1)
	}

	klog.V(11).Info("Got kube config, getting client")
	// Get kubernetes client based on config
	var kubeClient kubernetes.Interface = kubernetes.NewForConfigOrDie(cfg)
	watcher := watcherController.Init(kubeClient, allowed, cleanFreq, restrictNamespaces)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		klog.V(11).Info("Starting gather configmap")
		for {
			watcher.GatherConfigMaps(gatherFreq)
			klog.V(11).Info("Back in for loop before another call gather configs")
		}
		klog.V(11).Info("Exited worker configmap watcher")
	}()
	klog.V(11).Info("Outside the go functions")
	wg.Wait()
	klog.V(11).Info("After the wait function")
}
