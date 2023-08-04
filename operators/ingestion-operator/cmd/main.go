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
	"net"
	"os"
	"sync"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"github.com/go-logr/zapr"
	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"

	uberzap "go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	romashorodokcomv1alpha1 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/v1alpha1"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/internal/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(romashorodokcomv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type GRPCRunnable struct {
	server   *grpc.Server
	listener net.Listener
}

func (runnable *GRPCRunnable) Start(ctx context.Context) error {
	go func() {
		select {
		case <-ctx.Done():
			runnable.server.GracefulStop()
		}
	}()

	return runnable.server.Serve(runnable.listener)
}

const (
	GRPC_HOST = "localhost"
	GRPC_PORT = ":9191"
)

func NewGRPCRunnable(client client.Client, log *uberzap.Logger) (*GRPCRunnable, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s%s", GRPC_HOST, GRPC_PORT))

	if err != nil {
		return nil, fmt.Errorf("unable start listener %s", err)
	}

	logrLogger := zapr.NewLogger(log)
	logrLogger.Info("Starting serving grpc server", "grpc.addr", GRPC_PORT)

	return &GRPCRunnable{
		listener: listener,
		server:   grpcserver.NewServer(client, log),
	}, nil
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	zaprLogger := zap.NewRaw(zap.UseFlagOptions(&opts))
	logrLogger := zapr.NewLogger(zaprLogger)

	ctrl.SetLogger(logrLogger)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c0246183.romashorodok",
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

	grpcServer, err := NewGRPCRunnable(mgr.GetClient(), zaprLogger)

	if err != nil {
		setupLog.Error(err, "grpc server bootstrap failed")
		os.Exit(1)
	}

	if err = mgr.Add(grpcServer); err != nil {
		setupLog.Error(err, "unable add grpc server to manager")
		os.Exit(1)
	}

	if err = (&controller.IngestTemplateReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IngestTemplate")
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := mgr.Start(ctx); err != nil {
			setupLog.Error(err, "problem running manager")
			os.Exit(1)
		}
	}()

	select {
	case <-ctrl.SetupSignalHandler().Done():
		container.WithShutdown().Gracefully()
		cancel()
	}

	wg.Wait()
}
