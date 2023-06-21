package dalek

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2020-05-01/managementlocks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
)

func (d *Dalek) ResourceGroups(ctx context.Context) error {
	if err := d.deleteResourceGroups(ctx); err != nil {
		return fmt.Errorf("deleting Resource Groups: %+v", err)
	}
	return nil
}

func (d *Dalek) deleteResourceGroups(ctx context.Context) error {
	log.Printf("[DEBUG] Loading the first %d resource groups to delete", d.opts.NumberOfResourceGroupsToDelete)

	client := d.client.ResourceManager.ResourcesClient
	subscriptionId := commonids.NewSubscriptionID(d.client.SubscriptionID)
	listOpts := resourcegroups.ListOperationOptions{
		Top: pointer.To(d.opts.NumberOfResourceGroupsToDelete),
	}
	groups, err := client.List(ctx, subscriptionId, listOpts)
	if err != nil {
		return fmt.Errorf("listing Resource Groups: %+v", err)
	}

	if groups.Model == nil {
		log.Printf("[DEBUG]   No Resource Groups found")
		return nil
	}
	for _, resource := range *groups.Model {
		groupName := *resource.Name
		log.Printf("[DEBUG] Resource Group: %q", groupName)

		id := commonids.NewResourceGroupID(subscriptionId.SubscriptionId, groupName)
		if strings.EqualFold(*resource.Properties.ProvisioningState, "Deleting") {
			log.Println("[DEBUG]   Already being deleted - Skipping..")
			continue
		}

		if !shouldDeleteResourceGroup(resource, d.opts.Prefix) {
			log.Println("[DEBUG]   Shouldn't Delete - Skipping..")
			continue
		}

		if !d.opts.ActuallyDelete {
			log.Printf("[DEBUG]   Would have deleted group %q..", groupName)
			continue
		}

		// TODO: refactor this to support processing the items within the Resource Group, to determine if we need to unlock it
		// or break the SB namespace/purge the managed hsms etc https://github.com/tombuildsstuff/azurerm-dalek/issues/6

		locks, lerr := d.client.ResourceManager.LocksClient.ListAtResourceGroupLevel(ctx, id, managementlocks.DefaultListAtResourceGroupLevelOperationOptions())
		if lerr != nil {
			log.Printf("[DEBUG] Error obtaining Resource Group Locks : %+v", err)
		} else {
			if model := locks.Model; model != nil {
				for _, lock := range *model {
					if lock.Id == nil {
						log.Printf("[DEBUG]   Lock with nil id on %q", groupName)
						continue
					}
					id := *lock.Id

					if lock.Name == nil {
						log.Printf("[DEBUG]   Lock %s with nil name on %q", id, groupName)
						continue
					}

					log.Printf("[DEBUG]   Attemping to remove lock %s from : %s", id, groupName)

					lockId, err := managementlocks.ParseScopedLockID(id)
					if err != nil {
						continue
					}

					if _, lerr = d.client.ResourceManager.LocksClient.DeleteByScope(ctx, *lockId); lerr != nil {
						log.Printf("[DEBUG]   Unable to delete lock %s on resource group %q", *lock.Name, groupName)
					}
				}
			}
		}

		log.Printf("[DEBUG]   Deleting Resource Group %q..", groupName)
		if _, err := client.Delete(ctx, id, resourcegroups.DefaultDeleteOperationOptions()); err != nil {
			log.Printf("[DEBUG]   Error during deletion of Resource Group %q: %s", groupName, err)
			continue
		}
		log.Printf("[DEBUG]   Deletion triggered for Resource Group %q", groupName)
	}

	return nil
}

func shouldDeleteResourceGroup(input resourcegroups.ResourceGroup, prefix string) bool {
	if prefix != "" {
		if !strings.HasPrefix(strings.ToLower(*input.Name), strings.ToLower(prefix)) {
			return false
		}
	}

	if tags := input.Tags; tags != nil {
		for k := range *tags {
			if strings.EqualFold(k, "donotdelete") {
				return false
			}
		}
	}

	return true
}
