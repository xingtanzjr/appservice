package main

import (
	"fmt"
	"time"

	istioClientset "istio.io/client-go/pkg/clientset/versioned"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"metricsadvisor.ai/appservice/components"
	asClient "metricsadvisor.ai/appservice/generated/multitenancy/clientset/versioned"
	asInformerFactory "metricsadvisor.ai/appservice/generated/multitenancy/informers/externalversions"
)

var DEFAULT_RESYNC_INTERVAL = time.Second * 30

var clusterMap = map[string]string{
	"aks-0": "/Users/zhangjinrui/.kube/config-aks-0",
	"aks-1": "/Users/zhangjinrui/.kube/config-aks-1",
	"aks-2": "/Users/zhangjinrui/.kube/config-aks-2",
}

var apiServiceInformerFactoryList []asInformerFactory.SharedInformerFactory
var kubeInformerFactoryList []kubeinformers.SharedInformerFactory

func buildClusterTool() (map[string]*components.ClusterTool, error) {

	clusterToolMap := make(map[string]*components.ClusterTool)

	for clusterId := range clusterMap {
		cfg, err := getConfigByClusterId(clusterId)
		if err != nil {
			return nil, fmt.Errorf("Failed to fetch kubernetes config for cluster %s. Reason: %s", clusterId, err.Error())
		}

		apiServiceClient, err := asClient.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("Failed to create apiServiceClient for cluster %s. Reason: %s", clusterId, err.Error())
		}

		apiServiceInformerFactory := asInformerFactory.NewSharedInformerFactory(apiServiceClient, DEFAULT_RESYNC_INTERVAL)
		// Add the factory to list so that we can Start() it after initialization
		apiServiceInformerFactoryList = append(apiServiceInformerFactoryList, apiServiceInformerFactory)

		kubeClient, err := kubernetes.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("Failed to create kube client for cluster %s, Reason: %s", clusterId, err.Error())
		}

		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, DEFAULT_RESYNC_INTERVAL)
		// Add the kube informer factory to list so that we can Start() it after initialization
		kubeInformerFactoryList = append(kubeInformerFactoryList, kubeInformerFactory)

		istioClient, err := istioClientset.NewForConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("Failed to create istio client for cluster %s, Reason: %s", clusterId, err.Error())
		}

		clusterToolMap[clusterId] = &components.ClusterTool{
			ClusterId: clusterId,

			ApiServiceLister:   apiServiceInformerFactory.Multitenancy().V1().AppServices().Lister(),
			ApiServiceClient:   apiServiceClient,
			ApiServiceInformer: apiServiceInformerFactory.Multitenancy().V1().AppServices(),

			DeploymentLister:         kubeInformerFactory.Apps().V1().Deployments().Lister(),
			ServiceLister:            kubeInformerFactory.Core().V1().Services().Lister(),
			ServiceAccountLister:     kubeInformerFactory.Core().V1().ServiceAccounts().Lister(),
			RoleLister:               kubeInformerFactory.Rbac().V1().Roles().Lister(),
			RoleBindingLister:        kubeInformerFactory.Rbac().V1().RoleBindings().Lister(),
			ClusterRoleLister:        kubeInformerFactory.Rbac().V1().ClusterRoles().Lister(),
			ClusterRoleBindingLister: kubeInformerFactory.Rbac().V1().ClusterRoleBindings().Lister(),
			KubeClient:               kubeClient,

			IstioClient: istioClient,
		}
	}
	return clusterToolMap, nil
}

func getConfigByClusterId(clusterId string) (*restclient.Config, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", clusterMap[clusterId])
	return cfg, err
}

func startAndWaitInforms(stopCh <-chan struct{}) error {
	// Start all informers by its factory
	klog.Info("Starting all informers")
	for _, informer := range apiServiceInformerFactoryList {
		informer.Start(stopCh)
	}
	for _, informer := range kubeInformerFactoryList {
		informer.Start(stopCh)
	}

	// Wait until all the informer's first sync is done
	klog.Info("Waiting for informers' caches to sync")
	for _, informer := range apiServiceInformerFactoryList {
		if ok := cache.WaitForCacheSync(stopCh, informer.Multitenancy().V1().AppServices().Informer().HasSynced); !ok {
			return fmt.Errorf("Failed to wait for api service caches to sync")
		}
	}

	// Wait until all the kube basic informers' cache to be sync.
	// IMPORTANT: If some other Lister is used, remember to wait its cache sync here
	for _, informer := range kubeInformerFactoryList {
		if ok := cache.WaitForCacheSync(stopCh,
			informer.Apps().V1().Deployments().Informer().HasSynced,
			informer.Core().V1().Services().Informer().HasSynced,
			informer.Core().V1().ServiceAccounts().Informer().HasSynced,
			informer.Rbac().V1().Roles().Informer().HasSynced,
			informer.Rbac().V1().RoleBindings().Informer().HasSynced,
			informer.Rbac().V1().ClusterRoles().Informer().HasSynced,
			informer.Rbac().V1().ClusterRoleBindings().Informer().HasSynced); !ok {
			return fmt.Errorf("Failed to wait for kube basic informers' cache to sycn")
		}
	}
	return nil
}

func main() {
	stopCh := make(<-chan struct{})

	clusterToolMap, err := buildClusterTool()
	if err != nil {
		klog.Errorf("App existed because of: %s", err.Error())
		return
	}

	err = startAndWaitInforms(stopCh)
	if err != nil {
		klog.Errorf("App existed because of: %s", err.Error())
		return
	}

	apiServiceController := components.NewApiServiceController(clusterToolMap)
	apiServiceController.Run(1, stopCh)
	// close(stopCh)
}
