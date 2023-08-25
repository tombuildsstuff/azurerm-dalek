package cleaners

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

var _ SubscriptionCleaner = deleteResourceGroupsInSubscriptionCleaner{}

type deleteResourceGroupsInSubscriptionCleaner struct {
}

func (d deleteResourceGroupsInSubscriptionCleaner) Name() string {
	return "Delete Resource Groups in Subscription"
}

func (d deleteResourceGroupsInSubscriptionCleaner) Cleanup(ctx context.Context, subscriptionId commonids.SubscriptionId, client *clients.AzureClient, opts options.Options) error {
	log.Printf("[DEBUG] Loading the first %d resource groups to delete", opts.NumberOfResourceGroupsToDelete)

	listOpts := resourcegroups.ListOperationOptions{
		Top: pointer.To(opts.NumberOfResourceGroupsToDelete),
	}
	groups, err := client.ResourceManager.ResourcesClient.List(ctx, subscriptionId, listOpts)
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

		if !shouldDeleteResourceGroup(resource, opts.Prefix) {
			log.Println("[DEBUG]   Shouldn't Delete - Skipping..")
			continue
		}

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG]   Would have deleted group %q..", groupName)
			continue
		}

		// Locks and Nested Items within the Resource Group can cause issues during deletion
		// as such we have a set of Cleaners to go through and remove these locks/items
		// which are split out for simplicity since there's a number of them
		for _, cleaner := range ResourceGroupCleaners {
			log.Printf("[DEBUG] Running Resource Group Cleaner %q..", cleaner.Name())
			if err := cleaner.Cleanup(ctx, id, client, opts); err != nil {
				return fmt.Errorf("running Cleaner %q for %s: %+v", cleaner.Name(), id, err)
			}
		}

		log.Printf("[DEBUG]   Deleting Resource Group %q..", groupName)
		// NOTE: we're intentionally not using DeleteThenPoll since fire-and-forgetting these is fine
		if _, err := client.ResourceManager.ResourcesClient.Delete(ctx, id, resourcegroups.DefaultDeleteOperationOptions()); err != nil {
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
