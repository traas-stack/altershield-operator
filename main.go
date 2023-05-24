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
	"flag"
	"fmt"
	"log"
	"os"

	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/controllers/utils"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/routers"
	"gitlab.alipay-inc.com/common_release/altershieldoperator/runnable"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	appv1alpha1 "gitlab.alipay-inc.com/common_release/altershieldoperator/apis/app.ops.cloud.alipay.com/v1alpha1"
	appsv1 "gitlab.alipay-inc.com/common_release/altershieldoperator/apis/apps/v1"
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
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8089", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8088", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// Construct a new logr.logger.
	if err := utils.LogInit(); err != nil {
		setupLog.Error(err, "unable to init log")
		os.Exit(1)
	}
	getenv := os.Getenv("test")
	fmt.Println(getenv)
	r := routers.SetupRouter()
	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
	}()
	configDie := ctrl.GetConfigOrDie()
	//go webhook.StartWebhookServer(setupLog)

	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   1443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9f3eb48e.ops.cloud.alipay.com",
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

	for {
		mgr, err := ctrl.NewManager(configDie, options)
		if err != nil {
			setupLog.Error(err, "unable to start manager")
			//os.Exit(1)
			continue
		}

		utils.NewApp(mgr.GetClient(), mgr.GetCache(), configDie)

		if err = (&controllers.DeploymentReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Deployment")
			os.Exit(1)
		}
		if err = (&controllers.PodReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Deployment")
			os.Exit(1)
		}
		if err = (&controllers.OpsConfigInfoReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "OpsConfigInfo")
			os.Exit(1)
		}
		if err = (&controllers.ChangeWorkloadReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ChangeWorkload")
			os.Exit(1)
		}
		if err = (&controllers.ChangePodReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "ChangePod")
			os.Exit(1)
		}
		if err = (&appsv1.DeploymentWebhook{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Deployment")
			os.Exit(1)
		}
		//+kubebuilder:scaffold:builder

		if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
			setupLog.Error(err, "unable to set up health check")
			os.Exit(1)
		}
		if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
			setupLog.Error(err, "unable to set up ready check")
			os.Exit(1)
		}

		utils.ConfigRun()
		runnable.ChangeWorkloadTimeoutValidRun()
		runnable.ChangePodTimeoutValidRun()
		runnable.DeploymentStatusRollBackRun()
		// runable.DeploymentStatusPauseRun()

		setupLog.Info("starting manager")
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			//os.Exit(1)
			continue
		}
		os.Exit(1)
	}
}
