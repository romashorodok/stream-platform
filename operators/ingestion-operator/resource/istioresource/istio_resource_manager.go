package istioresource

import (
	"context"
	"errors"
	"log"

	"github.com/romashorodok/stream-platform/operators/ingestion-operator/api/networking.istio.io/v1alpha3"
	"go.uber.org/fx"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// The `istio` label on gateway loadbalancer
	ISTIO_SELECTOR       = "istio"
	GATEWAY_KIND         = "Gateway"
	VIRTUAL_SERVICE_KIND = "VirtualService"
)

type IstioResourceManager struct {
	k8s client.Client
}

type IstioGatewayParams struct {
	Number                 int32
	Protocol               string
	IngressGatewaySelector string
	IngressGatewayAppName  string
	IngestHostName         string /* Should be in that format username.localhost */
	Namespace              string
	Owner                  string
}

const OWNED_BY = "app.kubernetes.io/owned-by"

func (mgr *IstioResourceManager) IstioGateway(params IstioGatewayParams) *v1alpha3.Gateway {
	gateway := &v1alpha3.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.IngressGatewayAppName,
			Namespace: params.Namespace,
			Labels:    labels.Set{OWNED_BY: params.Owner},
		},
		Spec: v1alpha3.GatewaySpec{
			Selector: labels.Set{ISTIO_SELECTOR: params.IngressGatewaySelector},
			Servers: []v1alpha3.Server{
				{
					Port: v1alpha3.Port{
						Number:   params.Number,
						Name:     params.Protocol,
						Protocol: params.Protocol,
					},
					Hosts: []v1alpha3.Host{v1alpha3.Host(params.IngestHostName)},
				},
			},
		},
	}

	gateway.Kind = GATEWAY_KIND

	return gateway
}

type IstioVirtualServiceParams struct {
	IngestHostName        string /* Should be in that format username.localhost */
	IngestServiceNameHost string /* The name of stateless app service */
	IngestPorts           []corev1.ContainerPort

	VirtualServiceAppName string
	IngressGatewayAppName string
	Namespace             string
	Owner                 string
}

func (mgr *IstioResourceManager) IstioVirtualService(params IstioVirtualServiceParams) *v1alpha3.VirtualService {
	virtService := &v1alpha3.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      params.VirtualServiceAppName,
			Namespace: params.Namespace,
			Labels:    labels.Set{OWNED_BY: params.Owner},
		},
		Spec: v1alpha3.VirtualServiceSpec{
			Hosts:    []v1alpha3.Host{v1alpha3.Host(params.IngestHostName)},
			Gateways: []string{params.IngressGatewayAppName},
			Http:     []v1alpha3.HTTPRoute{},
		},
	}

	for _, port := range params.IngestPorts {
		if port.Protocol != "TCP" {
			log.Println("Support only TCP")
			continue
		}

		virtService.Spec.Http = append(virtService.Spec.Http,
			v1alpha3.HTTPRoute{
				Route: []v1alpha3.HTTPRouteDestination{
					{
						Destination: v1alpha3.Destination{
							Host: v1alpha3.Host(params.IngestServiceNameHost),
							Port: v1alpha3.PortSelector{
								Number: port.ContainerPort,
							},
						},
					},
				},
			},
		)
	}

	virtService.Kind = VIRTUAL_SERVICE_KIND

	return virtService
}

type GetIstioVirtualServiceByAppNameParams struct {
	Context               context.Context
	VirtualServiceAppName string
	Namespace             string
	Owner                 string
}

func (mgr *IstioResourceManager) GetIstioVirtualServiceByAppName(params GetIstioVirtualServiceByAppNameParams) (*v1alpha3.VirtualService, error) {
	var virtualService v1alpha3.VirtualService
	namespacedName := types.NamespacedName{Namespace: params.Namespace, Name: params.VirtualServiceAppName}

	if err := mgr.k8s.Get(params.Context, namespacedName, &virtualService); err != nil {
		return nil, err
	}

	owner, ok := virtualService.Labels[OWNED_BY]
	if !ok {
		return nil, errors.New("trying to get non-owner resource access")
	}

	if owner != params.Owner {
		return nil, errors.New("trying to get non-owner resource access")
	}

	return &virtualService, nil
}

type GetIstioGatewayByAppNameParams struct {
	Context              context.Context
	IngresGatewayAppName string
	Namespace            string
	Owner                string
}

func (mgr *IstioResourceManager) GetIstioGatewayByAppName(params GetIstioGatewayByAppNameParams) (*v1alpha3.Gateway, error) {
	var gateway v1alpha3.Gateway
	namespacedName := types.NamespacedName{Namespace: params.Namespace, Name: params.IngresGatewayAppName}

	if err := mgr.k8s.Get(params.Context, namespacedName, &gateway); err != nil {
		return nil, err
	}

	owner, ok := gateway.Labels[OWNED_BY]
	if !ok {
		return nil, errors.New("trying to get non-owner resource access")
	}

	if owner != params.Owner {
		return nil, errors.New("trying to get non-owner resource access")
	}

	return &gateway, nil
}

type IstioResourceManagerParams struct {
	fx.In

	K8s client.Client
}

func NewIstioResourceManager(params IstioResourceManagerParams) *IstioResourceManager {
	return &IstioResourceManager{k8s: params.K8s}
}
