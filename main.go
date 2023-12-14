/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	promapi "github.com/prometheus/client_golang/api"
	altershield "github.com/traas-stack/altershield-operator/pkg/altershield/client"
	"github.com/traas-stack/altershield-operator/pkg/controllers"
	metricprovider "github.com/traas-stack/altershield-operator/pkg/metric/provider"
	"github.com/traas-stack/altershield-operator/pkg/metric/provider/metricsapi"
	"github.com/traas-stack/altershield-operator/pkg/metric/provider/prometheus"
	"github.com/traas-stack/altershield-operator/pkg/webhook/mutating"
	"github.com/traas-stack/altershield-operator/pkg/webhook/validating"
	"github.com/traas-stack/altershield-operator/routers"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	resourceclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	appv1alpha1 "github.com/traas-stack/altershield-operator/apis/app.ops.cloud.alipay.com/v1alpha1"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(appv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var altershieldAddr string
	var metricProviderType string
	var promConfig         promConfig

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8089", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8088", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&altershieldAddr, "altershield-address", "192.168.171.46:8080", "The address the althershield endpoint binds to.")
	flag.StringVar(&metricProviderType, "metric-provider", "prometheus",
		"The name of metric provider. Valid options are prometheus or metrics-api. Defaults to prometheus.")
	flag.StringVar(&promConfig.Address, "prometheus-address", "", "The address of the Prometheus to connect to.")
	flag.BoolVar(&promConfig.AuthInCluster, "prometheus-auth-incluster", false,
		"Use auth details from the in-cluster kubeconfig when connecting to prometheus.")
	flag.StringVar(&promConfig.AuthConfigFile, "prometheus-auth-config", "",
		"The kubeconfig file used to configure auth when connecting to Prometheus.")
	flag.StringVar(&promConfig.MetricsConfigFile, "prometheus-metrics-config", "",
		"The configuration file containing details of how to transform between Prometheus metrics and metrics API resources.")
	flag.DurationVar(&promConfig.MetricsRelistInterval, "prometheus-metrics-relist-interval", 10*time.Minute,
		"The interval at which to re-list the set of all available metrics from Prometheus.")
	flag.DurationVar(&promConfig.MetricsMaxAge, "prometheus-metrics-max-age", 0,
		"The period for which to query the set of available metrics from Prometheus. If not set, it defaults to prometheus-metrics-relist-interval.")
	flag.Parse()

	cfg := ctrl.GetConfigOrDie()
	//go webhook.StartWebhookServer(setupLog)

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   1443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "altershield-operator.altershield.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	}
	if os.Getenv("ENVIRONMENT") == "DEV" {
		path, err := os.Getwd()
		if err != nil {
			setupLog.Error(err, "unable to get work dir")
			os.Exit(1)
		}
		options.CertDir = path + "/certs"
	}

	mgr, err := ctrl.NewManager(cfg, options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = mgr.GetFieldIndexer().IndexField(context.TODO(), &appv1alpha1.ChangeDefense{}, "status.currentExecutionID",
		func(obj client.Object) []string {
			cd, ok := obj.(*appv1alpha1.ChangeDefense)
			if !ok {
				return nil
			}
			return []string{cd.Status.CurrentExecutionID}
		}); err != nil {
		setupLog.Error(err, "failed to index change defense field")
		os.Exit(1)
	}
	if err = mgr.GetFieldIndexer().IndexField(context.TODO(), &appv1alpha1.ChangeDefenseExecution{}, "spec.id",
		func(obj client.Object) []string {
			cde, ok := obj.(*appv1alpha1.ChangeDefenseExecution)
			if !ok {
				return nil
			}
			return []string{cde.Spec.ID}
		}); err != nil {
		setupLog.Error(err, "failed to index change defense execution field")
		os.Exit(1)
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "unable to build dynamic client")
		os.Exit(1)
	}
	var metricProvider metricprovider.Interface
	switch metricProviderType {
	case "prometheus":
		metricProvider, err = newPrometheusProviderFromConfig(mgr.GetClient(), dynamicClient, promConfig)
	case "metrics-api":
		metricProvider, err = newMetricsAPIProviderFromConfig(cfg)
	default:
		err = fmt.Errorf("unknown metric provider type %q", metricProviderType)
	}
	if err != nil {
		setupLog.Error(err, "unable to build metric provider", "metricProviderType", metricProviderType)
		os.Exit(1)
	}

	if err = (&controllers.ChangeDefenseReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		AsClient: altershield.NewAltershieldClient(altershieldAddr),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ChangeDefense")
		os.Exit(1)
	}

	if err = (&controllers.ChangeDefenseExecutionReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		AsClient:     altershield.NewAltershieldClient(altershieldAddr),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ChangeDefenseExecution")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	// create webhooks
	mgr.GetWebhookServer().Register("/validate", &webhook.Admission{
		Handler: validating.NewHandler(),
	})
	mgr.GetWebhookServer().Register("/mutate", &webhook.Admission{
		Handler: mutating.NewHandler(altershieldAddr),
	})

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	r := routers.SetupRouter(mgr.GetClient(), metricProvider)
	go func() {
		if err := r.Run(":8080"); err != nil {
			klog.Fatal("ListenAndServe: ", err)
		}
	}()

	if runnable, ok := metricProvider.(manager.Runnable); ok {
		if err := mgr.Add(runnable); err != nil {
			setupLog.Error(err, "failed to add runnable metric provider to manager",
				"metricProviderType", metricProviderType)
			os.Exit(1)
		}
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
	os.Exit(1)
}

type promConfig struct {
	Address                              string
	AuthInCluster                        bool
	AuthConfigFile                       string
	MetricsConfigFile                    string
	MetricsRelistInterval, MetricsMaxAge time.Duration
}

func newPrometheusProviderFromConfig(kubeClient client.Client, kubeDynamicClient dynamic.Interface, config promConfig) (metricprovider.Interface, error) {
	if config.MetricsMaxAge == 0 {
		config.MetricsMaxAge = config.MetricsRelistInterval
	}

	promClient, err := buildPrometheusClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build prometheus client: %v", err)
	}

	metricsConfig, err := prometheus.MetricsConfigFromFile(config.MetricsConfigFile)

	if err != nil {
		return nil, fmt.Errorf("failed to load metrics config from file %q: %v", config.MetricsConfigFile, err)
	}

	return prometheus.NewMetricProvider(kubeClient, kubeDynamicClient, promClient, metricsConfig, config.MetricsRelistInterval, config.MetricsMaxAge)
}

func buildPrometheusClient(config promConfig) (promapi.Client, error) {
	if config.AuthInCluster && config.AuthConfigFile != "" {
		return nil, fmt.Errorf("may not use both in-cluster auth and an explicit kubeconfig at the same time")
	}
	var (
		rt http.RoundTripper = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}
		authConfig *rest.Config
		err        error
	)
	if config.AuthInCluster {
		authConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build in-cluster auth config: %v", err)
		}
	} else if config.AuthConfigFile != "" {
		authConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: config.AuthConfigFile},
			&clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build auth config from kubeconfig %q: %v", config.AuthConfigFile, err)
		}
	}
	if authConfig != nil {
		rt, err = rest.TransportFor(authConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build transport from auth config: %v", err)
		}
	}
	return promapi.NewClient(promapi.Config{
		Address:      config.Address,
		RoundTripper: rt,
	})
}

func newMetricsAPIProviderFromConfig(kubeConfig *rest.Config) (metricprovider.Interface, error) {
	resourceMetricsClient, err := resourceclient.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build resource metrics client: %v", err)
	}
	return metricsapi.NewMetricProvider(resourceMetricsClient), nil
}
