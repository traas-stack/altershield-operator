package utils

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// App 用于直接调用k8s api
var App AppClient

type AppClient struct {
	client.Client
	cache.Cache
	K8sClient *kubernetes.Clientset
	K8sConfig *rest.Config
}

func NewApp(client client.Client, cache cache.Cache, config *rest.Config) AppClient {
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	App = AppClient{client, cache, k8sClient, config}
	return App
}
