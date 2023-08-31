package cleaners

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/recoveryservices/2023-04-01/vaults"
	"github.com/hashicorp/go-azure-sdk/resource-manager/recoveryservicesbackup/2023-02-01/protectioncontainers"
	"github.com/hashicorp/go-azure-sdk/resource-manager/recoveryservicesbackup/2023-04-01/backupprotecteditems"
	"github.com/hashicorp/go-azure-sdk/resource-manager/recoveryservicesbackup/2023-04-01/backupprotectioncontainers"
	"github.com/hashicorp/go-azure-sdk/resource-manager/recoveryservicesbackup/2023-04-01/protecteditems"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resourcegraph/2022-10-01/resources"
	"github.com/hashicorp/go-azure-sdk/sdk/client/pollers"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

type recoveryServicesResourceGroupCleaner struct{}

var _ ResourceGroupCleaner = recoveryServicesResourceGroupCleaner{}

func (c recoveryServicesResourceGroupCleaner) Name() string {
	return "Recovery Services"
}

func (c recoveryServicesResourceGroupCleaner) Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient, opts options.Options) error {
	vaultsClient := client.ResourceManager.RecoverServicesVaultsClient
	protectionContainersClient := client.ResourceManager.RecoveryServicesProtectionContainersClient
	backupProtectedItemsClient := client.ResourceManager.RecoveryServicesBackupProtectedItemsClient
	protectedItemsClient := client.ResourceManager.RecoveryServicesProtectedItemsClient

	log.Printf("[DEBUG] Retrieving Recovery Services Vaults in %s..", id)
	vaultIds, err := c.findRecoveryServiceVaultIDs(ctx, id, client)
	if err != nil {
		return fmt.Errorf("finding the Recovery Services Vaults IDs within %s: %+v", id, err)
	}

	for _, vaultId := range *vaultIds {
		// we then need to find any Backup Protected Items within this Recovery Services Vault
		// list all of the items within the protection container

		backupContainerVaultId := backupprotectioncontainers.NewVaultID(vaultId.SubscriptionId, vaultId.ResourceGroupName, vaultId.VaultName)
		filter := "backupManagementType eq 'AzureStorage'"
		listOpts := backupprotectioncontainers.ListOperationOptions{Filter: &filter}
		results, err := client.ResourceManager.RecoveryServicesBackupProtectionContainersClient.ListComplete(ctx, backupContainerVaultId, listOpts)
		if err != nil {
			return fmt.Errorf("listing Backup Protection Containers within %s: %+v", backupContainerVaultId, err)
		}

		for _, item := range results.Items {
			protectionContainerId, err := protectioncontainers.ParseProtectionContainerID(*item.Id)
			if err != nil {
				return err
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", protectionContainerId)
				continue
			}

			if _, err = protectionContainersClient.Unregister(ctx, *protectionContainerId); err != nil {
				return fmt.Errorf("unable to unregister %s: %+v", protectionContainerId, err)
			}

			log.Printf("[DEBUG] Polling until Protection Container is unregistered for %s..", *protectionContainerId)
			pollerType := recoveryServicesProtectionContainerPoller{
				client:   protectionContainersClient,
				configId: *protectionContainerId,
			}
			poller := pollers.NewPoller(pollerType, 30*time.Second, pollers.DefaultNumberOfDroppedConnectionsToAllow)
			if err := poller.PollUntilDone(ctx); err != nil {
				return fmt.Errorf("polling until the Protection Container is unregistered for %s: %+v", *protectionContainerId, err)
			}
			log.Printf("[DEBUG] Protection Container Unregistered for %s", vaultId)
		}

		protectedItemVaultId := backupprotecteditems.NewVaultID(vaultId.SubscriptionId, vaultId.ResourceGroupName, vaultId.VaultName)
		protectedItems, err := backupProtectedItemsClient.ListComplete(ctx, protectedItemVaultId, backupprotecteditems.DefaultListOperationOptions())
		if err != nil {
			return fmt.Errorf("listing Backup Protected Items with %s: %+v", protectedItemVaultId, err)
		}

		for _, item := range protectedItems.Items {
			protectedItemId, err := protecteditems.ParseProtectedItemID(*item.Id)
			if err != nil {
				return err
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", protectedItemId)
				continue
			}

			if _, err = protectedItemsClient.Delete(ctx, *protectedItemId); err != nil {
				return fmt.Errorf("unable to delete %s: %+v", protectedItemId, err)
			}

			log.Printf("[DEBUG] Polling until Protected Item is deleted for %s..", *protectedItemId)
			pollerType := recoveryServicesProtectedItemPoller{
				client:   protectedItemsClient,
				configId: *protectedItemId,
			}
			poller := pollers.NewPoller(pollerType, 30*time.Second, pollers.DefaultNumberOfDroppedConnectionsToAllow)
			if err := poller.PollUntilDone(ctx); err != nil {
				return fmt.Errorf("polling until the Protected Item is deleted for %s: %+v", *protectedItemId, err)
			}
			log.Printf("[DEBUG] Protected Item Deleted for %s", vaultId)

			// removing the protected item is eventually consistent against the vault so we will poll the protected items list until it is removed.
			listPollerType := recoveryServicesBackupProtectedItemListPoller{
				client:          backupProtectedItemsClient,
				vaultId:         protectedItemVaultId,
				protectedItemId: protectedItemId.String(),
			}
			listPoller := pollers.NewPoller(listPollerType, 30*time.Second, pollers.DefaultNumberOfDroppedConnectionsToAllow)
			if err := listPoller.PollUntilDone(ctx); err != nil {
				return fmt.Errorf("polling until the Protected Item removed from Vault ListComplete command for %s: %+v", *protectedItemId, err)
			}
			log.Printf("[DEBUG] Protected Item Removed for %s", vaultId)
		}

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted %s..", vaultId)
			continue
		}

		log.Printf("[DEBUG] Deleting %s..", vaultId)

		if _, err = vaultsClient.Delete(ctx, vaultId); err != nil {
			return fmt.Errorf("deleting %s: %+v", vaultId, err)
		}
		log.Printf("[DEBUG] Deleted %s.", vaultId)
	}

	return nil
}

func (c recoveryServicesResourceGroupCleaner) ResourceTypes() []string {
	return []string{
		"Microsoft.RecoveryServices/vaults",
	}
}

func (c recoveryServicesResourceGroupCleaner) findRecoveryServiceVaultIDs(ctx context.Context, resourceGroupId commonids.ResourceGroupId, client *clients.AzureClient) (*[]vaults.VaultId, error) {
	query := strings.TrimSpace(fmt.Sprintf(`
resources
| where type =~ "Microsoft.RecoveryServices/vaults"
| where resourceGroup =~ '%s'
| project id
| sort by (tolower(tostring(id))) asc
`, resourceGroupId.ResourceGroupName))
	payload := resources.QueryRequest{
		Options: &resources.QueryRequestOptions{
			Top: pointer.To(int64(1000)),
		},
		Query: query,
		Subscriptions: &[]string{
			resourceGroupId.SubscriptionId,
		},
	}
	resp, err := client.ResourceManager.ResourceGraphClient.Resources(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("performing graph query %q: %+v", query, err)
	}

	if resp.Model == nil {
		return nil, fmt.Errorf("performing graph query %q: response was nil", query)
	}
	if resp.Model.Data == nil {
		return nil, fmt.Errorf("performing graph query %q: response.data was nil", query)
	}

	vaultIds := make([]vaults.VaultId, 0)
	itemsRaw, ok := resp.Model.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected the data to be an []interface but got %+v", resp.Model.Data)
	}
	for index, itemRaw := range itemsRaw {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected index %d to be a map[string]interface{} but it wasn't", index)
		}
		id, ok := item["id"]
		if !ok {
			return nil, fmt.Errorf("expected an id field for item %d but didn't get one", index)
		}
		idRaw := id.(string)
		namespaceId, err := vaults.ParseVaultIDInsensitively(idRaw)
		if err != nil {
			return nil, fmt.Errorf("parsing %q for index %d: %+v", idRaw, index, err)
		}
		vaultIds = append(vaultIds, *namespaceId)
	}

	return &vaultIds, nil
}

type recoveryServicesProtectionContainerPoller struct {
	client   *protectioncontainers.ProtectionContainersClient
	configId protectioncontainers.ProtectionContainerId
}

func (s recoveryServicesProtectionContainerPoller) Poll(ctx context.Context) (*pollers.PollResult, error) {
	// poll until the protection container is deleted
	result, err := s.client.Get(ctx, s.configId)
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

	return &pollers.PollResult{
		Status:       pollers.PollingStatusInProgress,
		PollInterval: 30 * time.Second,
	}, nil
}

type recoveryServicesProtectedItemPoller struct {
	client   *protecteditems.ProtectedItemsClient
	configId protecteditems.ProtectedItemId
}

func (s recoveryServicesProtectedItemPoller) Poll(ctx context.Context) (*pollers.PollResult, error) {
	// poll until the status protected item is deleted
	result, err := s.client.Get(ctx, s.configId, protecteditems.DefaultGetOperationOptions())
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

	return &pollers.PollResult{
		Status:       pollers.PollingStatusInProgress,
		PollInterval: 30 * time.Second,
	}, nil
}

type recoveryServicesBackupProtectedItemListPoller struct {
	client          *backupprotecteditems.BackupProtectedItemsClient
	vaultId         backupprotecteditems.VaultId
	protectedItemId string
}

func (s recoveryServicesBackupProtectedItemListPoller) Poll(ctx context.Context) (*pollers.PollResult, error) {
	// poll until the list doesn't contain the protectedItemId
	result, err := s.client.ListComplete(ctx, s.vaultId, backupprotecteditems.DefaultListOperationOptions())
	if err != nil {
		if len(result.Items) == 0 {
			return &pollers.PollResult{
				Status: pollers.PollingStatusSucceeded,
			}, nil
		}
		return nil, pollers.PollingFailedError{
			Message: err.Error(),
		}
	}

	for _, item := range result.Items {
		if item.Id == nil {
			continue
		}
		if strings.EqualFold(*item.Id, s.protectedItemId) {
			return &pollers.PollResult{
				Status:       pollers.PollingStatusInProgress,
				PollInterval: 30 * time.Second,
			}, nil
		}
	}

	return &pollers.PollResult{
		Status: pollers.PollingStatusSucceeded,
	}, nil
}
