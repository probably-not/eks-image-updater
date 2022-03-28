package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/probably-not/eks-image-updater/utils"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	inCluster               bool
	kubeconfig, kubeContext string
	logLevel                string
	region, targetTag       string
)

type servicesFlag []service
type service struct {
	namespace string
	name      string
}

func parseService(v string) (service, error) {
	spl := strings.Split(v, "/")
	if len(spl) != 2 {
		return service{}, errors.New("invalid service")
	}
	return service{
		namespace: spl[0],
		name:      spl[1],
	}, nil
}

func (sf *servicesFlag) Set(value string) error {
	split := strings.Split(value, ",")
	for _, s := range split {
		svc, err := parseService(s)
		if err != nil {
			return err
		}
		*sf = append(*sf, svc)
	}

	return nil
}

func (sf *servicesFlag) String() string {
	return fmt.Sprint(*sf)
}

func init() {
	defaultKubeconfigPath, defaultKubeconfigHelp := "", "absolute path to the kubeconfig file"
	if home := os.Getenv("HOME"); home != "" {
		defaultKubeconfigPath = filepath.Join(home, ".kube", "config")
		defaultKubeconfigHelp = "(optional) absolute path to the kubeconfig file"
	}

	// K8S configuration
	flag.StringVar(&kubeconfig, "kubeconfig", defaultKubeconfigPath, defaultKubeconfigHelp)
	flag.BoolVar(&inCluster, "in-cluster", true, "Set to false if run outside of a k8s cluster.")
	flag.StringVar(&kubeContext, "kube-context", "", "The name of the kubernetes context to search")

	flag.StringVar(&logLevel, "log-level", os.Getenv("LOG_LEVEL"), "The level of logging")
	flag.StringVar(&region, "region", "us-east-1", "the region to use in AWS")
	flag.StringVar(&targetTag, "tag", "latest", "the tag to look for that signifies the image you want to push to your deployment")
}

func main() {
	var services servicesFlag
	flag.Var(&services, "services", "a comma separated list of the services to watch and update; may be used multiple times")
	flag.Parse()

	// Set up the STDOUT logger to enable logging to STDOUT when we want to use it
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	if region == "" {
		flag.Usage()
		logrus.Fatal("region must not be empty")
	}

	if len(services) == 0 {
		flag.Usage()
		logrus.Fatal("services must not be empty")
	}

	kubeClient, err := utils.GetKubeClient(inCluster, kubeconfig, kubeContext)
	if err != nil {
		logrus.WithError(err).Fatalf("failed to get Kubernetes API Client")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		logrus.WithError(err).Fatal("unable to load AWS SDK config")
	}

	svc := ecr.NewFromConfig(cfg)

	for _, service := range services {
		images, err := fetchImages(svc, service.name)
		if err != nil {
			logrus.WithError(err).WithField("service", service.name).Fatal("failed to get images for service")
		}

		var (
			targetImage types.ImageDetail
			found       bool
		)
		for _, image := range images {
			if utils.StrSliceContains(image.ImageTags, targetTag) {
				targetImage = image
				found = true
				break
			}
		}

		if !found {
			logrus.WithError(err).WithField("service", service.name).Fatal("failed find latest image tag")
		}

		err = reconcileService(kubeClient, service, targetImage)
		if err != nil {
			logrus.WithError(err).WithField("service", service.name).Fatal("failed to reconcile service")
		}
	}
}

func fetchImages(svc *ecr.Client, serviceName string) ([]types.ImageDetail, error) {
	images := make([]types.ImageDetail, 0)
	repositoryName := aws.String(serviceName)
	var nextToken *string
	for {
		resp, err := svc.DescribeImages(context.TODO(), &ecr.DescribeImagesInput{RepositoryName: repositoryName, NextToken: nextToken})
		if err != nil {
			return nil, err
		}

		images = append(images, resp.ImageDetails...)
		if resp.NextToken == nil {
			break
		}

		nextToken = resp.NextToken
	}

	return images, nil
}

func reconcileService(kubeClient *kubernetes.Clientset, svc service, imageDetails types.ImageDetail) error {
	deployment, err := kubeClient.AppsV1().Deployments(svc.namespace).Get(context.TODO(), svc.name, v1.GetOptions{})
	if err != nil {
		return err
	}

	for cIdx, c := range deployment.Spec.Template.Spec.Containers {
		if c.Name != svc.name {
			continue
		}

		tagIDX := strings.LastIndex(c.Image, ":")
		if tagIDX < 0 {
			return fmt.Errorf("image %s does not contain a tag", c.Image)
		}

		currentTag := c.Image[tagIDX+1:]
		if utils.StrSliceContains(imageDetails.ImageTags, currentTag) {
			logrus.WithField("service", svc).Info("service tag has not changed")
			continue
		}

		logrus.WithFields(logrus.Fields{
			"service":     svc,
			"current_tag": currentTag,
			"latest_tags": imageDetails.ImageTags,
		}).Info("service tag has changed")

		newTag := utils.GetValidImageTag(imageDetails.ImageTags)
		newImage := strings.ReplaceAll(c.Image, currentTag, newTag)
		deployment.Spec.Template.Spec.Containers[cIdx].Image = newImage
		_, err := kubeClient.AppsV1().Deployments(svc.namespace).Update(context.TODO(), deployment, v1.UpdateOptions{})
		if err != nil {
			return err
		}

		logrus.WithFields(logrus.Fields{
			"service":      svc,
			"previous_tag": currentTag,
			"new_tag":      newTag,
		}).Info("updated service tag")
	}

	return nil
}
