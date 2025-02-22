package istio

import (
	"errors"
	"fmt"

	istioapiv1beta1 "istio.io/api/networking/v1beta1"
	istioclientapiv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/lifecycle-manager/api/v1beta2"
)

const (
	contractVersion = "v1"
	prefixFormat    = "/%s/%s/event"
)

func NewVirtualService(namespace string, watcher *v1beta2.Watcher, gateways *istioclientapiv1beta1.GatewayList) (*istioclientapiv1beta1.VirtualService, error) {
	if err := validateArgumentsForNewVirtualService(namespace, watcher, gateways); err != nil {
		return nil, err
	}

	hosts, err := getHosts(gateways.Items)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to construct hosts from gateways: %w", ErrInvalidArgument), err)
	}

	httpRoute, err := NewHTTPRoute(watcher)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("unable to construct httpRoute from watcher: %w", ErrInvalidArgument), err)
	}

	virtualService := &istioclientapiv1beta1.VirtualService{}
	virtualService.SetName(watcher.Name)
	virtualService.SetNamespace(namespace)
	virtualService.Spec.Gateways = getGatewayNames(gateways.Items)
	virtualService.Spec.Hosts = hosts
	virtualService.Spec.Http = []*istioapiv1beta1.HTTPRoute{
		httpRoute,
	}

	return virtualService, nil
}

func getGatewayNames(gateways []*istioclientapiv1beta1.Gateway) []string {
	gatewayNames := make([]string, 0)
	for _, gateway := range gateways {
		gatewayNames = append(gatewayNames, client.ObjectKeyFromObject(gateway).String())
	}
	return gatewayNames
}

func getHosts(gateways []*istioclientapiv1beta1.Gateway) ([]string, error) {
	hosts := make([]string, 0)

	for _, gateway := range gateways {
		gatewayHosts := make([]string, 0)
		for _, server := range gateway.Spec.GetServers() {
			gatewayHosts = append(gatewayHosts, server.GetHosts()...)
		}

		if len(gatewayHosts) == 0 {
			return nil, fmt.Errorf("for gateway %s: %w",
				client.ObjectKeyFromObject(gateway).String(),
				ErrCantFindGatewayServersHost)
		}

		hosts = append(hosts, gatewayHosts...)
	}

	return hosts, nil
}

func destinationHost(serviceName, serviceNamespace string) string {
	return fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, serviceNamespace)
}

func validateArgumentsForNewVirtualService(namespace string, watcher *v1beta2.Watcher, gateways *istioclientapiv1beta1.GatewayList) error {
	if namespace == "" {
		return fmt.Errorf("namespace must not be empty: %w", ErrInvalidArgument)
	}

	if watcher == nil {
		return fmt.Errorf("watcher must not be nil: %w", ErrInvalidArgument)
	}

	if watcher.GetName() == "" {
		return fmt.Errorf("watcher.Name must not be empty: %w", ErrInvalidArgument)
	}

	if gateways == nil || len(gateways.Items) == 0 {
		return fmt.Errorf("gateways must not be empty: %w", ErrInvalidArgument)
	}

	return nil
}
