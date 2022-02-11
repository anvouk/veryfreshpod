package app

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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
	namedLogger := logger.Named("k8s")
	if !isInsideCluster() {
		return nil, errors.New("usage outside of a k8s cluster is not supported at the moment")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster config: %v", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed creating new k8s client: %v", err)
	}

	ver, err := k8sClient.ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed detecting k8s version: %v", err)
	}
	namedLogger.Infof("connected to k8s: %v.%v", ver.Major, ver.Minor)

	client := &K8s{
		logger: namedLogger,
		client: k8sClient,
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

func (k8s *K8s) WatchPodsForChanges(config *Config) cache.Controller {
	restClient := k8s.client.CoreV1().RESTClient()
	lw := cache.NewListWatchFromClient(restClient, "pods", v1.NamespaceAll, fields.Everything())
	_, controller := cache.NewInformer(lw,
		&v1.Pod{},
		config.RefreshInterval,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				pod, ok := obj.(*v1.Pod)
				if !ok {
					k8s.logger.Errorw("failed watch on add pod", "error", obj)
					return
				}
				k8s.logger.Infow("pod added", "pod", pod.Name, "namespace", pod.Namespace)
			},
			DeleteFunc: func(obj interface{}) {
				pod, ok := obj.(*v1.Pod)
				if !ok {
					k8s.logger.Errorw("failed watch on remove pod", "error", obj)
					return
				}
				k8s.logger.Infow("pod removed", "pod", pod.Name, "namespace", pod.Namespace)
			},
		},
	)
	return controller
}
