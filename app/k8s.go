package app

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	"strconv"
)

const minSupportedVersionMinor int = 20

type K8s struct {
	logger *zap.SugaredLogger
	client *kubernetes.Clientset
}

func isInsideCluster() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

func NewK8sClient(logger *zap.SugaredLogger) (*K8s, error) {
	if !isInsideCluster() {
		return nil, errors.New("usage outside of a k8s cluster is not supported at the moment")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed creating new k8s client: %v", err)
	}

	client := &K8s{
		client: clientset,
		logger: logger.Named("k8s"),
	}

	return client, nil
}

func (k8s *K8s) IsClusterVersionSupported() error {
	ver, err := k8s.client.ServerVersion()
	if err != nil {
		return fmt.Errorf("failed connecting to k8s cluster: %v", err)
	}

	intMin, err := strconv.ParseInt(ver.Minor, 10, 32)
	if err != nil {
		return fmt.Errorf("failed parsing k8s version: %v", err)
	}

	if int(intMin) < minSupportedVersionMinor {
		return fmt.Errorf("detected unsupported k8s version '%v'", intMin)
	}
	return nil
}
