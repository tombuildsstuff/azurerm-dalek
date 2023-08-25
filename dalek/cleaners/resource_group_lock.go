package cleaners

import (
	"context"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2020-05-01/managementlocks"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

var _ ResourceGroupCleaner = removeLocksFromResourceGroupCleaner{}

type removeLocksFromResourceGroupCleaner struct {
}

func (r removeLocksFromResourceGroupCleaner) Name() string {
	return "Removing Locks.."
}

func (r removeLocksFromResourceGroupCleaner) Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient, opts options.Options) error {
	locks, err := client.ResourceManager.LocksClient.ListAtResourceGroupLevel(ctx, id, managementlocks.DefaultListAtResourceGroupLevelOperationOptions())
	if err != nil {
		log.Printf("[DEBUG] Error obtaining Resource Group Locks : %+v", err)
	}

	if model := locks.Model; model != nil {
		for _, lock := range *model {
			if lock.Id == nil {
				log.Printf("[DEBUG]   Lock with nil id on %q", id.ResourceGroupName)
				continue
			}
			lockId, err := managementlocks.ParseScopedLockID(*lock.Id)
			if err != nil {
				log.Printf("[ERROR] Parsing Scoped Lock ID %q: %+v", *lock.Id, err)
				continue
			}

			if lock.Name == nil {
				log.Printf("[DEBUG]   Lock %s with nil name on %q", id, id.ResourceGroupName)
				continue
			}

			log.Printf("[DEBUG]   Attemping to remove lock %s from: %s", id, id.ResourceGroupName)

			if _, err := client.ResourceManager.LocksClient.DeleteByScope(ctx, *lockId); err != nil {
				log.Printf("[DEBUG]   Unable to delete lock %s on resource group %q", *lock.Name, id.ResourceGroupName)
			}
		}
	}
	return nil
}
