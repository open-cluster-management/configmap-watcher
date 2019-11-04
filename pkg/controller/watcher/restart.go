package watcher

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
)

// RestartAll calls the restart functions for every deployment/daemonset/statefulset that is watching
// the configmap that was updated.
func RestartAll(client *kubernetes.Clientset, configmap types.NamespacedName, watchedConfigmaps map[types.NamespacedName]*ConfigMapper) {
	klog.V(3).Infof("Configmap update %v", configmap)
	// Get the configmapper
	configmapper := watchedConfigmaps[configmap]

	// Restart deployments
	for kind := range configmapper.Deployments {
		if err := restartDeployment(client, kind); err != nil {
			klog.Errorf("Unable to restart pods associated with deployment %s, error message: %s", kind.Name, err.Error())
		}
	}
	// Restart daemonsets
	for kind := range configmapper.Daemonsets {
		if err := restartDaemonset(client, kind); err != nil {
			klog.Errorf("Unable to restart pods associated with daemonset %s, error message: %s", kind.Name, err.Error())
		}
	}
	// Restart statefulset
	for kind := range configmapper.Statefulsets {
		if err := restartStatefulset(client, kind); err != nil {
			klog.Errorf("Unable to restart pods associated with statefulset %s, error message: %s", kind.Name, err.Error())
		}
	}
}

func restartDeployment(client *kubernetes.Clientset, deploymentName types.NamespacedName) error {
	update := time.Now().Format("2006-1-2.1504")
	klog.Infof("Restarting deployment %s at %s", deploymentName.String(), update)
	deploymentsInterface := client.AppsV1().Deployments(deploymentName.Namespace)
	deployment, err := deploymentsInterface.Get(deploymentName.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("error occurred getting deployment %v", deployment)
		return err
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

func restartDaemonset(client *kubernetes.Clientset, daemonsetName types.NamespacedName) error {
	update := time.Now().Format("2006-1-2.1504")
	klog.Infof("Restarting daemonset %s at %s", daemonsetName.String(), update)
	daemonsetInterface := client.AppsV1().DaemonSets(daemonsetName.Namespace)
	daemonset, err := daemonsetInterface.Get(daemonsetName.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error getting daemonset %v", daemonsetName)
		return err
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

func restartStatefulset(client *kubernetes.Clientset, statefulsetName types.NamespacedName) error {
	update := time.Now().Format("2006-1-2.1504")
	klog.Infof("Restarting statefulset %s at %s", statefulsetName.String(), update)
	statefulsetInterface := client.AppsV1().StatefulSets(statefulsetName.Namespace)
	statefulset, err := statefulsetInterface.Get(statefulsetName.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Error getting statefulset %v", statefulsetName)
		return err
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
