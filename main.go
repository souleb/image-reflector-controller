/*
Copyright 2020 The Flux authors

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
	"os"

	"github.com/dgraph-io/badger"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	crtlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	imagev1alpha1 "github.com/fluxcd/image-reflector-controller/api/v1alpha1"
	"github.com/fluxcd/image-reflector-controller/controllers"
	"github.com/fluxcd/image-reflector-controller/internal/database"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/fluxcd/pkg/runtime/logger"
	"github.com/fluxcd/pkg/runtime/metrics"
	"github.com/fluxcd/pkg/runtime/probes"
	// +kubebuilder:scaffold:imports
)

const controllerName = "image-reflector-controller"

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = imagev1alpha1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var (
		metricsAddr          string
		eventsAddr           string
		healthAddr           string
		enableLeaderElection bool
		logLevel             string
		logJSON              bool
		watchAllNamespaces   bool
		storagePath          string
	)

	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&eventsAddr, "events-addr", "", "The address of the events receiver.")
	flag.StringVar(&healthAddr, "health-addr", ":9440", "The address the health endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&logLevel, "log-level", "info", "Set logging level. Can be debug, info or error.")
	flag.BoolVar(&logJSON, "log-json", false, "Set logging to JSON format.")
	flag.BoolVar(&watchAllNamespaces, "watch-all-namespaces", true,
		"Watch for custom resources in all namespaces, if set to false it will only watch the runtime namespace.")
	flag.StringVar(&storagePath, "storage-path", "/data", "Where to store the persistent database of image metadata")
	flag.Parse()

	ctrl.SetLogger(logger.NewLogger(logLevel, logJSON))

	var eventRecorder *events.Recorder
	if eventsAddr != "" {
		if er, err := events.NewRecorder(eventsAddr, controllerName); err != nil {
			setupLog.Error(err, "unable to create event recorder")
			os.Exit(1)
		} else {
			eventRecorder = er
		}
	}

	metricsRecorder := metrics.NewRecorder()
	crtlmetrics.Registry.MustRegister(metricsRecorder.Collectors()...)

	watchNamespace := ""
	if !watchAllNamespaces {
		watchNamespace = os.Getenv("RUNTIME_NAMESPACE")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: healthAddr,
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "e189b2df.fluxcd.io",
		Namespace:              watchNamespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	probes.SetupChecks(mgr, setupLog)

	badgerDB, err := badger.Open(badger.DefaultOptions(storagePath))
	if err != nil {
		setupLog.Error(err, "unable to open the Badger database")
		os.Exit(1)
	}
	defer badgerDB.Close()
	db := database.NewBadgerDatabase(badgerDB)

	if err = (&controllers.ImageRepositoryReconciler{
		Client:                mgr.GetClient(),
		Log:                   ctrl.Log.WithName("controllers").WithName(imagev1alpha1.ImageRepositoryKind),
		Scheme:                mgr.GetScheme(),
		EventRecorder:         mgr.GetEventRecorderFor(controllerName),
		ExternalEventRecorder: eventRecorder,
		MetricsRecorder:       metricsRecorder,
		Database:              db,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", imagev1alpha1.ImageRepositoryKind)
		os.Exit(1)
	}
	if err = (&controllers.ImagePolicyReconciler{
		Client:                mgr.GetClient(),
		Log:                   ctrl.Log.WithName("controllers").WithName(imagev1alpha1.ImagePolicyKind),
		Scheme:                mgr.GetScheme(),
		EventRecorder:         mgr.GetEventRecorderFor(controllerName),
		ExternalEventRecorder: eventRecorder,
		MetricsRecorder:       metricsRecorder,
		Database:              db,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", imagev1alpha1.ImagePolicyKind)
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
