package app

import (
	"context"
	"fmt"
	"github.com/emirpasic/gods/v2/lists/arraylist"
	"github.com/emirpasic/gods/v2/maps/hashmap"
	"go.uber.org/zap"
	v1 "k8s.io/api/apps/v1"
	"os"
	"strings"
)

type Store struct {
	logger *zap.SugaredLogger
	docker *Docker
	k8s    *K8s
	// images map for each statefulset.name|container.name
	imagesForStatefulset *hashmap.Map[string, *arraylist.List[string]]
	imagesForDeployment  *hashmap.Map[string, *arraylist.List[string]]
}

func NewStore(logger *zap.SugaredLogger, docker *Docker, k8s *K8s) *Store {
	return &Store{
		logger:               logger.Named("store"),
		docker:               docker,
		k8s:                  k8s,
		imagesForStatefulset: hashmap.New[string, *arraylist.List[string]](),
		imagesForDeployment:  hashmap.New[string, *arraylist.List[string]](),
	}
}

func (s *Store) NewDockerImage(ctx context.Context, imageName string) {
	s.logger.Infow("docker image added", "imageName", imageName)

	// all images which need to be replaced among statefulsets and deployments
	var replaceImagesList []string

	// check statefulsets
	allImagesForStatefulsets := s.imagesForStatefulset.Keys()
	for _, image := range allImagesForStatefulsets {
		// we replace only images which name is the same but tag differs
		if strings.Split(image, ":")[0] == strings.Split(imageName, ":")[0] {
			replaceImagesList = append(replaceImagesList, image)
		}
	}
	// check deployments
	allImagesForDeployments := s.imagesForDeployment.Keys()
	for _, image := range allImagesForDeployments {
		// we replace only images which name is the same but tag differs
		if strings.Split(image, ":")[0] == strings.Split(imageName, ":")[0] {
			replaceImagesList = append(replaceImagesList, image)
		}
	}

	for _, imageToReplace := range replaceImagesList {
		s.logger.Infow("search and replace image", "oldImage", imageToReplace, "newImage", imageName)
		// check and update statefulsets
		if imagesForStatefulset, found := s.imagesForStatefulset.Get(imageToReplace); found {
			imagesForStatefulset.Each(func(_ int, value string) {
				splittedVal := strings.Split(value, "|")
				err := s.k8s.ReplaceImageForStatefulset(ctx, splittedVal[0], splittedVal[1], splittedVal[2], imageName)
				if err != nil {
					s.logger.Errorw("failed changing statefulset image", "error", err)
					// TODO: better way to handle desync?
					os.Exit(1)
				}
			})
			// update our image index map
			s.imagesForStatefulset.Put(imageName, imagesForStatefulset)
			s.imagesForStatefulset.Remove(imageToReplace)
		}
		// check and update deployments
		if imagesForDeployment, found := s.imagesForDeployment.Get(imageToReplace); found {
			imagesForDeployment.Each(func(_ int, value string) {
				splittedVal := strings.Split(value, "|")
				err := s.k8s.ReplaceImageForDeployment(ctx, splittedVal[0], splittedVal[1], splittedVal[2], imageName)
				if err != nil {
					s.logger.Errorw("failed changing deployment image", "error", err)
					// TODO: better way to handle desync?
					os.Exit(1)
				}
			})
			// update our image index map
			s.imagesForDeployment.Put(imageName, imagesForDeployment)
			s.imagesForDeployment.Remove(imageToReplace)
		}
	}
}

func (s *Store) RemoveDockerImage(imageName string) {
	s.logger.Infow("docker image removed", "imageName", imageName)
}

func (s *Store) NewK8sDeployment(deployment *v1.Deployment) {
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if foundList, found := s.imagesForDeployment.Get(container.Image); found {
			newEntry := fmt.Sprintf("%s|%s|%s", deployment.Namespace, deployment.Name, container.Name)
			foundList.Add(newEntry)
			s.logger.Infow("new deployment entry for existing list", "entry", newEntry, "listPrevSize", foundList.Size()-1)
		} else {
			pairList := arraylist.New[string]()
			newEntry := fmt.Sprintf("%s|%s|%s", deployment.Namespace, deployment.Name, container.Name)
			pairList.Add(newEntry)
			s.imagesForDeployment.Put(container.Image, pairList)
			s.logger.Infow("new deployment entry", "entry", newEntry)
		}
	}
}

func (s *Store) RemoveK8sDeployment(deployment *v1.Deployment) {
	allKeys := s.imagesForDeployment.Keys()
	for _, key := range allKeys {
		containersInfo, _ := s.imagesForDeployment.Get(key)
		var removeIndexesList []int
		containersInfo.Each(func(index int, value string) {
			if strings.HasPrefix(value, fmt.Sprintf("%s|%s", deployment.Namespace, deployment.Name)) {
				removeIndexesList = append(removeIndexesList, index)
			}
		})
		for _, indexToRemove := range removeIndexesList {
			entryName, _ := containersInfo.Get(indexToRemove)
			s.logger.Infow("removed deployment entry", "entry", entryName)
			containersInfo.Remove(indexToRemove)
		}
	}
}

func (s *Store) NewK8sStatefulSet(statefulSet *v1.StatefulSet) {
	for _, container := range statefulSet.Spec.Template.Spec.Containers {
		if foundList, found := s.imagesForStatefulset.Get(container.Image); found {
			newEntry := fmt.Sprintf("%s|%s|%s", statefulSet.Namespace, statefulSet.Name, container.Name)
			foundList.Add(newEntry)
			s.logger.Infow("new statefulset entry for existing list", "entry", newEntry, "listPrevSize", foundList.Size()-1)
		} else {
			pairList := arraylist.New[string]()
			newEntry := fmt.Sprintf("%s|%s|%s", statefulSet.Namespace, statefulSet.Name, container.Name)
			pairList.Add(newEntry)
			s.imagesForStatefulset.Put(container.Image, pairList)
			s.logger.Infow("new statefulset entry", "entry", newEntry)
		}
	}
}

func (s *Store) RemoveK8sStatefulSet(statefulSet *v1.StatefulSet) {
	allKeys := s.imagesForStatefulset.Keys()
	for _, key := range allKeys {
		containersInfo, _ := s.imagesForStatefulset.Get(key)
		var removeIndexesList []int
		containersInfo.Each(func(index int, value string) {
			if strings.HasPrefix(value, fmt.Sprintf("%s|%s", statefulSet.Namespace, statefulSet.Name)) {
				removeIndexesList = append(removeIndexesList, index)
			}
		})
		for _, indexToRemove := range removeIndexesList {
			entryName, _ := containersInfo.Get(indexToRemove)
			s.logger.Infow("removed statefulset entry", "entry", entryName)
			containersInfo.Remove(indexToRemove)
		}
	}
}
