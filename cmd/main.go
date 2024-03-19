/*
Copyright 2024 Contributors to EdgeNet Project.

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
	"crypto/tls"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"google.golang.org/appengine/log"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	antreav1alpha1 "antrea.io/antrea/pkg/apis/crd/v1alpha1"

	multitenancyv1 "github.com/edgenet-project/edgenet-software/api/multitenancy/v1"
	labellerscontroller "github.com/edgenet-project/edgenet-software/internal/controller/labellers"
	multitenancycontroller "github.com/edgenet-project/edgenet-software/internal/controller/multitenancy"
	"github.com/edgenet-project/edgenet-software/internal/labeller"
	"github.com/edgenet-project/edgenet-software/internal/utils"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(antreav1alpha1.AddToScheme(scheme))
	utilruntime.Must(multitenancyv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var disabledReconcilers utils.FlagList
	var debug bool
	var maxmindUrl string
	var maxmindAccountId string
	var maxmindToken string
	flag.BoolVar(&debug, "debug", false, "Debug mode for the logger")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&maxmindAccountId, "maxmind-accountid", "", "The account id of the maxmind geodatabase.")
	flag.StringVar(&maxmindToken, "maxmind-token", "", "The access token of the maxmind geodatabase.")
	flag.StringVar(&maxmindUrl, "maxmind-url", "https://geoip.maxmind.com/geoip/v2.1/city/", "The endpoint of the maxmind for the ip lookup to work.")
	flag.Var(&disabledReconcilers, "disabled-reconcilers", "Comma seperated values of the reconciliers, Tenant,TenantResourceQuota,SubNamespace...")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development: debug,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancelation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "154d4dad.edge-net.io",
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
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Try to read the maxmind accountid, and token from the file.
	maxmind, err := labeller.NewMaxMindFromSecret(maxmindUrl, maxmindAccountId, maxmindToken)

	// If the error is not nil then we cannot get the maxmind config and we should not start node labeller reconcilier.
	disableNodeLabeller := err != nil

	if err != nil {
		log.Warningf(context.TODO(), "Cannot retrieve the MaxMind Account Token, running without the NodeLabeller")
	}

	// Setup reconcilers, we might want to add the list of reconcilers. This part is auto generated.
	// If you want to add the functionality to disable reconcilers, put it inside an if.
	// WARNING: This part is semi-auto-generated! By default you cannot disable reconcilers since they are
	// generated autside an if clause. ADD YOUR IF CLAUSE MANUALLY!
	if !disabledReconcilers.Contains("Tenant") {
		if err = (&multitenancycontroller.TenantReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Tenant")
			os.Exit(1)
		}
	}
	if !disabledReconcilers.Contains("TenantResourceQuota") {
		if err = (&multitenancycontroller.TenantResourceQuotaReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "TenantResourceQuota")
			os.Exit(1)
		}
	}
	// For NodeLabeller to be activated, it should not be in the disabled reconciler list and the maxmind from secret
	// should not return an error
	if !disabledReconcilers.Contains("NodeLabeller") && !disableNodeLabeller {
		if err = (&labellerscontroller.NodeLabellerReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			// Add the maxming configuration to the reconcilier.
			MaxMind: maxmind,
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "NodeLabeller")
			os.Exit(1)
		}
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

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
