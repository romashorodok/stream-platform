package controller

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/fx"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/go-logr/logr"
	v1alpha1 "github.com/romashorodok/stream-platform/operators/ingestion-operator/api/romashorodok.github.io"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/grpcserver/container"
	"github.com/romashorodok/stream-platform/operators/ingestion-operator/resource/ingestresource"
)

type IngestTemplateReconciler struct {
	client.Client
	Scheme       *runtime.Scheme
	ingestSystem *ingestresource.IngestSystem
}

//+kubebuilder:rbac:groups=romashorodok.github.io,resources=ingesttemplates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=romashorodok.github.io,resources=ingesttemplates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=romashorodok.github.io,resources=ingesttemplates/finalizers,verbs=update

func (r *IngestTemplateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var ingestResourceTemplate v1alpha1.IngestTemplate
	if err := r.Get(ctx, req.NamespacedName, &ingestResourceTemplate); err != nil {
		log.Error(err, "Unable find ingest")
	}

	ingestTemplates := container.WithIngestTemplates()

	ingestTemplates.AddIngestTemplate(ingestResourceTemplate)

	//TODO: remove it form context.
	container.WithShutdown().AddTask(func() {
		labels := labels.Set{ingestresource.CREATED_BY: req.Name}

		var deployments appsv1.DeploymentList

		if err := r.List(ctx, &deployments, client.MatchingLabelsSelector{
			Selector: labels.AsSelector(),
		}); err != nil {
			fmt.Println()
			fmt.Println(err)

			log.Info("Not found deployment servers to remove it")
			return
		}

		for _, deployment := range deployments.Items {
			broadcasterID := deployment.Labels[ingestresource.BROADCASTER_ID]
			appName := deployment.Labels[ingestresource.OWNED_BY]

			log.Info("Destroying ingest system", "appName", appName, "broadcasterID", broadcasterID)

			if err := r.ingestSystem.StopIngestSystem(ctx, appName, deployment.Namespace); err != nil {
				log.Error(err, "Unable stop ingest system", "appName", appName, "broadcasterID", broadcasterID)
			}

			_ = r.Delete(ctx, &deployment)
		}
	})

	return ctrl.Result{}, nil
}

func (r *IngestTemplateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.IngestTemplate{}).
		Complete(r)
}

type IngestControllerParams struct {
	fx.In

	Mgr          ctrl.Manager
	IngestSystem *ingestresource.IngestSystem
	Log          logr.Logger
}

func NewIngestController(params IngestControllerParams) {
	if err := (&IngestTemplateReconciler{
		Client:       params.Mgr.GetClient(),
		Scheme:       params.Mgr.GetScheme(),
		ingestSystem: params.IngestSystem,
	}).SetupWithManager(params.Mgr); err != nil {
		params.Log.Error(err, "unable to create controller", "controller", "IngestTemplate")
		os.Exit(1)
	}
}
