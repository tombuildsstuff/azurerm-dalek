package cleaners

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	servicebusV20220101Preview "github.com/hashicorp/go-azure-sdk/resource-manager/servicebus/2022-01-01-preview"
	"github.com/hashicorp/go-azure-sdk/resource-manager/servicebus/2022-01-01-preview/disasterrecoveryconfigs"
	"github.com/hashicorp/go-azure-sdk/sdk/client/pollers"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

var _ ResourceGroupCleaner = serviceBusNamespaceBreakPairingCleaner{}

type serviceBusNamespaceBreakPairingCleaner struct {
}

func (serviceBusNamespaceBreakPairingCleaner) Name() string {
	return "ServiceBus Namespace - Break Pairing"
}

func (serviceBusNamespaceBreakPairingCleaner) Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient, opts options.Options) error {
	serviceBusClient := client.ResourceManager.ServiceBus
	namespacesInResourceGroup, err := serviceBusClient.Namespaces.ListByResourceGroupComplete(ctx, id)
	if err != nil {
		return fmt.Errorf("listing the ServiceBus Namespaces within %s: %+v", id, err)
	}

	for _, namespace := range namespacesInResourceGroup.Items {
		namespaceId, err := disasterrecoveryconfigs.ParseNamespaceIDInsensitively(*namespace.Id)
		if err != nil {
			log.Printf("[ERROR] Parsing ServiceBus Namespace ID %q: %+v", *namespace.Id, err)
			continue
		}
		log.Printf("[DEBUG] Finding Disaster Recovery Configs within %s", *namespaceId)
		configs, err := serviceBusClient.DisasterRecoveryConfigs.ListComplete(ctx, *namespaceId)
		if err != nil {
			return fmt.Errorf("finding Disaster Recovery Configs within %s: %+v", *namespaceId, err)
		}

		for _, config := range configs.Items {
			if props := config.Properties; props == nil || *props.Role == disasterrecoveryconfigs.RoleDisasterRecoverySecondary {
				log.Printf("[DEBUG] Skipping %s for %s: Role is %q", *config.Id, *namespace.Id, *props.Role)
				continue
			}
			configId, err := disasterrecoveryconfigs.ParseDisasterRecoveryConfigIDInsensitively(*config.Id)
			if err != nil {
				return fmt.Errorf("parsing the Disaster Recovery Config ID %q: %+v", *config.Id, err)
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have broken the pairing for %s..", *configId)
				continue
			}

			log.Printf("[DEBUG] Breaking Pairing for %s..", *configId)
			if resp, err := serviceBusClient.DisasterRecoveryConfigs.BreakPairing(ctx, *configId); err != nil {
				if !response.WasNotFound(resp.HttpResponse) {
					return fmt.Errorf("breaking pairing for %s: %+v", *configId, err)
				}
			}
			log.Printf("[DEBUG] Polling until Pairing is broken for %s..", *configId)
			pollerType := serviceBusNamespaceBreakPairingPoller{
				client:   serviceBusClient,
				configId: *configId,
			}
			poller := pollers.NewPoller(pollerType, 30*time.Second, pollers.DefaultNumberOfDroppedConnectionsToAllow)
			if err := poller.PollUntilDone(ctx); err != nil {
				return fmt.Errorf("polling until the Pairing is broken for %s: %+v", *configId, err)
			}
			log.Printf("[DEBUG] Pairing Broken for %s", *configId)
		}
	}
	return nil
}

func (serviceBusNamespaceBreakPairingCleaner) ResourceTypes() []string {
	return []string{
		"Microsoft.ServiceBus/namespaces",
	}
}

type serviceBusNamespaceBreakPairingPoller struct {
	client   *servicebusV20220101Preview.Client
	configId disasterrecoveryconfigs.DisasterRecoveryConfigId
}

func (s serviceBusNamespaceBreakPairingPoller) Poll(ctx context.Context) (*pollers.PollResult, error) {
	// poll until the status is unbroken
	result, err := s.client.DisasterRecoveryConfigs.Get(ctx, s.configId)
	if err != nil {
		if response.WasNotFound(result.HttpResponse) {
			return &pollers.PollResult{
				Status: pollers.PollingStatusSucceeded,
			}, nil
		}
		return nil, pollers.PollingFailedError{
			Message: err.Error(),
		}
	}

	if model := result.Model; model != nil {
		if props := model.Properties; props != nil {
			if props.PartnerNamespace == nil || *props.PartnerNamespace == "" {
				return &pollers.PollResult{
					Status: pollers.PollingStatusSucceeded,
				}, nil
			}
		}
	}

	return &pollers.PollResult{
		Status:       pollers.PollingStatusInProgress,
		PollInterval: 30 * time.Second,
	}, nil
}
