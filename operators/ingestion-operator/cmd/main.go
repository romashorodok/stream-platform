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
	"log"
	"net"
	"os"
	"sync"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/romashorodok/stream-platform/pkg/envutils"
	"github.com/romashorodok/stream-platform/pkg/subject"
	"github.com/romashorodok/stream-platform/pkg/variables"

	v1alpha1 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/romashorodok.github.io"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/ingest"
	controller "github.com/romashorodok/stream-platform/operators/ingestion-operator/internal/controller/romashorodok.github.io"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/ingestresource"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/istioresource"

	"go.uber.org/fx"
	uberzap "go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	networkingistioiov1alpha3 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/networking.istio.io/v1alpha3"
	networkingistioiocontroller "github.com/romashorodok/stream-platform/operators/ingestion-operator/internal/controller/networking.istio.io"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(networkingistioiov1alpha3.AddToScheme(scheme))

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
	GRPC_HOST_DEFAULT = "localhost"
	GRPC_PORT_DEFAULT = "9191"

	GRPC_HOST_VAR = "GRPC_HOST"
	GRPC_PORT_VAR = "GRPC_PORT"
)

func AsGRPCServerOption(f interface{}) interface{} {
	return fx.Annotate(
		f,
		fx.As(new(grpc.ServerOption)),
		fx.ResultTags(`group:"grpc.ServerOption"`),
	)
}

type GRPCServerParams struct {
	fx.In

	Lifecycle fx.Lifecycle

	K8s     client.Client
	Logger  *uberzap.Logger
	Options []grpc.ServerOption `group:"grpc.ServerOption"`
}

func NewGRPCServer(params GRPCServerParams) *grpc.Server {
	return grpc.NewServer(params.Options...)
	// return grpcserver.NewServer(params.Options, params.K8s, params.Logger)
}

type GRPCRunnableParams struct {
	fx.In

	K8s    client.Client
	Logger logr.Logger
	Server *grpc.Server
}

func NewGRPCRunnable(params GRPCRunnableParams) *GRPCRunnable {
	PORT := envutils.Env(GRPC_PORT_VAR, GRPC_PORT_DEFAULT)
	HOST := envutils.Env(GRPC_HOST_VAR, GRPC_HOST_DEFAULT)

	listener, err := net.Listen("tcp", net.JoinHostPort(HOST, PORT))
	if err != nil {
		params.Logger.Error(err, fmt.Sprintf("Unable listen on %s:%s", HOST, PORT))
	}

	params.Logger.Info("Starting serving grpc server", "grpc.host", HOST, "grpc.addr", PORT)

	return &GRPCRunnable{listener: listener, server: params.Server}
}

type ManagerOptionsParams struct {
	MetricsAddr          string
	EnableLeaderElection bool
	ProbeAddr            string
}

func NewManagerOptions(params ManagerOptionsParams) func() ctrl.Options {
	return func() ctrl.Options {
		return ctrl.Options{
			Scheme:                 scheme,
			MetricsBindAddress:     params.MetricsAddr,
			Port:                   9443,
			HealthProbeBindAddress: params.ProbeAddr,
			LeaderElection:         params.EnableLeaderElection,
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
		}

	}
}

type ManagerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Options   ctrl.Options
	Logger    logr.Logger
}

func NewManager(params ManagerParams) manager.Manager {

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), params.Options)
	if err != nil {
		params.Logger.Error(err, "Unable start manager")
		os.Exit(1)
	}

	return mgr
}

type ClientK8sClientParams struct {
	fx.In

	Manager manager.Manager
}

func NewK8sClient(params ClientK8sClientParams) client.Client {
	return params.Manager.GetClient()
}

func NewZaprOptions() *zap.Options {
	return &zap.Options{
		Development: true,
	}
}

type ZaprLoggerParams struct {
	fx.In

	Options *zap.Options
}

func NewZaprLogger(params ZaprLoggerParams) *uberzap.Logger {
	return zap.NewRaw(zap.UseFlagOptions(params.Options))
}

type LogrLoggerParams struct {
	fx.In

	ZaprLogger *uberzap.Logger
}

func NewLogrLogger(params LogrLoggerParams) logr.Logger {
	return zapr.NewLogger(params.ZaprLogger)
}

type NatsConfig struct {
	Port string
	Host string
}

func (c *NatsConfig) GetUrl() string {
	return fmt.Sprintf("nats://%s:%s", c.Host, c.Port)
}

func NewNatsConfig() *NatsConfig {
	return &NatsConfig{
		Host: envutils.Env(variables.NATS_HOST, variables.NATS_HOST_DEFAULT),
		Port: envutils.Env(variables.NATS_PORT, variables.NATS_PORT_DEFAULT),
	}
}

type NatsConnectionParams struct {
	fx.In

	Config *NatsConfig
}

func NewNatsConnection(params NatsConnectionParams) *nats.Conn {
	conn, err := nats.Connect(params.Config.GetUrl())
	if err != nil {
		log.Panicf("Unable start nats connection. Err: %s", err)
		os.Exit(1)
	}

	return conn
}

type NatsJetstreamParams struct {
	fx.In

	Conn *nats.Conn
}

func NewNatsJetstream(params NatsJetstreamParams) nats.JetStreamContext {
	js, err := params.Conn.JetStream()
	if err != nil {
		log.Panicf("Unable start nats jetstream connection. Err: %s", err)
		os.Exit(1)
	}

	js.AddStream(subject.INGEST_DESTROYING_STREAM_CONFIG)

	return js
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

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	fx.New(
		fx.Provide(
			NewZaprOptions,
			NewZaprLogger,
			NewLogrLogger,

			AsGRPCServerOption(
				grpcserver.NewLoggerInterceptor,
			),

			NewManagerOptions(ManagerOptionsParams{
				MetricsAddr:          metricsAddr,
				ProbeAddr:            probeAddr,
				EnableLeaderElection: enableLeaderElection,
			}),
			NewManager,
			NewK8sClient,
			NewGRPCServer,
			NewGRPCRunnable,

			NewNatsConfig,
			NewNatsConnection,
			NewNatsJetstream,

			istioresource.NewIstioResourceManager,

			ingestresource.NewIngestResourceManager,
			ingestresource.NewIngestSystem,

			ingest.NewIngestControllerService,
		),
		fx.Invoke(controller.NewIngestController),

		fx.Invoke(func(ZapOption *zap.Options) {
			ZapOption.BindFlags(flag.CommandLine)
		}),
		fx.Invoke(func(Logger logr.Logger) {
			ctrl.SetLogger(Logger)
		}),
		fx.Invoke(grpcserver.RegisterIngestionControllerService),
		fx.Invoke(func(Lifecycle fx.Lifecycle, mgr ctrl.Manager, GrpcRunnable *GRPCRunnable) error {
			Lifecycle.Append(fx.Hook{
				OnStart: func(context.Context) (err error) {
					if err := mgr.Add(GrpcRunnable); err != nil {
						setupLog.Error(err, "unable add grpc server to manager")
						os.Exit(1)
					}

					if err = (&networkingistioiocontroller.GatewayReconciler{
						Client: mgr.GetClient(),
						Scheme: mgr.GetScheme(),
					}).SetupWithManager(mgr); err != nil {
						setupLog.Error(err, "unable to create controller", "controller", "Gateway")
						os.Exit(1)
					}
					if err = (&networkingistioiocontroller.VirtualServiceReconciler{
						Client: mgr.GetClient(),
						Scheme: mgr.GetScheme(),
					}).SetupWithManager(mgr); err != nil {
						setupLog.Error(err, "unable to create controller", "controller", "VirtualService")
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

					go func() {
						defer wg.Done()
						if err := mgr.Start(ctx); err != nil {
							setupLog.Error(err, "problem running manager")
							os.Exit(1)
						}
					}()

					return nil
				},
				OnStop: func(context.Context) error {
					container.WithShutdown().Gracefully()
					cancel()
					wg.Wait()

					return nil
				},
			})

			return nil
		}),
	).Run()
}
