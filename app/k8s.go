package app

import (
	"context"
	"errors"
	"fmt"
	"go.uber.org/zap"
	appsV1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"os"
	"strconv"
)

const minSupportedVersionMinor int = 24

type K8s struct {
	logger *zap.SugaredLogger
	client *kubernetes.Clientset

	coreInformers informers.SharedInformerFactory
}

type K8sWatcher struct {
	OnAddDeployment    func(deployment *appsV1.Deployment)
	OnRemoveDeployment func(deployment *appsV1.Deployment)

	OnAddStatefulSet    func(statefulSet *appsV1.StatefulSet)
	OnRemoveStatefulSet func(statefulSet *appsV1.StatefulSet)
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
		logger:        namedLogger,
		client:        k8sClient,
		coreInformers: nil,
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

// RegisterWatchForChanges watches k8s resources for added or removed pods
func (k8s *K8s) RegisterWatchForChanges(config *Config, watcher *K8sWatcher) error {
	if watcher == nil {
		return errors.New("invalid arg: watcher is nil")
	}

	var err error
	coreInformers := informers.NewSharedInformerFactory(k8s.client, config.RefreshInterval)

	// Deployments
	_, err = coreInformers.Apps().V1().Deployments().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			deployment, ok := obj.(*appsV1.Deployment)
			if !ok {
				k8s.logger.Errorf("failed cast on k8s resource: %T", deployment)
				return
			}
			if watcher.OnAddDeployment != nil {
				watcher.OnAddDeployment(deployment)
			}
		},
		DeleteFunc: func(obj interface{}) {
			deployment, ok := obj.(*appsV1.Deployment)
			if !ok {
				k8s.logger.Errorf("failed cast on k8s resource: %T", deployment)
				return
			}
			if watcher.OnRemoveDeployment != nil {
				watcher.OnRemoveDeployment(deployment)
			}
		},
	})
	if err != nil {
		return err
	}

	// StatefulSets
	_, err = coreInformers.Apps().V1().StatefulSets().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			statefulSet, ok := obj.(*appsV1.StatefulSet)
			if !ok {
				k8s.logger.Errorf("failed cast on k8s resource: %T", statefulSet)
				return
			}
			if watcher.OnAddStatefulSet != nil {
				watcher.OnAddStatefulSet(statefulSet)
			}
		},
		DeleteFunc: func(obj interface{}) {
			statefulSet, ok := obj.(*appsV1.StatefulSet)
			if !ok {
				k8s.logger.Errorf("failed cast on k8s resource: %T", statefulSet)
				return
			}
			if watcher.OnRemoveStatefulSet != nil {
				watcher.OnRemoveStatefulSet(statefulSet)
			}
		},
	})
	if err != nil {
		return err
	}

	k8s.coreInformers = coreInformers
	return nil
}

func (k8s *K8s) Run(ctx context.Context) error {
	if k8s.coreInformers == nil {
		return errors.New("no watchers registered. Call RegisterWatchForChanges before running")
	}

	stopCh := ctx.Done()
	k8s.coreInformers.WaitForCacheSync(stopCh)
	k8s.coreInformers.Start(stopCh)

	return nil
}
