package utils

import (
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetKubeClient(inCluster bool, kubeconfig, kubeContext string) (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	if inCluster {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}

		logrus.Info("Using in cluster kube config")
		return kubernetes.NewForConfig(config)
	}

	if kubeContext == "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		logrus.Debugf("Using given kube config: %+v", config)
		return kubernetes.NewForConfig(config)
	}

	config, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{
			CurrentContext: kubeContext,
		}).ClientConfig()
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Using given kube config: %+v", config)
	return kubernetes.NewForConfig(config)
}
