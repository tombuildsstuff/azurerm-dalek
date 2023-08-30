package cleaners

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resourcegraph/2022-10-01/resources"
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
	groups, err := client.ResourceManager.ResourcesGroupsClient.List(ctx, subscriptionId, listOpts)
	if err != nil {
		return fmt.Errorf("listing Resource Groups: %+v", err)
	}

	if groups.Model == nil {
		log.Printf("[DEBUG]   No Resource Groups found")
		return nil
	}

	resourceGroups := make([]string, 0)
	for _, resource := range *groups.Model {
		if strings.EqualFold(*resource.Properties.ProvisioningState, "Deleting") {
			log.Printf("[DEBUG] Resource Group %q is already being deleted - Skipping..", *resource.Name)
			continue
		}
		if !shouldDeleteResourceGroup(resource, opts.Prefix) {
			log.Printf("[DEBUG] Resource Group %q shouldn't be deleted - Skipping..", *resource.Name)
			continue
		}

		resourceGroups = append(resourceGroups, *resource.Name)
	}
	sort.Strings(resourceGroups)

	// pull out a list of Resource Types supported by the cleaners
	resourceTypes := make([]string, 0)
	for _, cleaner := range ResourceGroupCleaners {
		resourceTypes = append(resourceTypes, cleaner.ResourceTypes()...)
	}

	for _, groupName := range resourceGroups {
		log.Printf("[DEBUG] Resource Group: %q", groupName)

		id := commonids.NewResourceGroupID(subscriptionId.SubscriptionId, groupName)
		if !opts.ActuallyDelete {
			log.Printf("[DEBUG]   Would have deleted %s..", id)
			continue
		}

		// Locks and Nested Items within the Resource Group can cause issues during deletion
		// as such we have a set of Cleaners to go through and remove these locks/items
		// which are split out for simplicity since there's a number of them
		//
		// However since there's a non-trivial number of these, let's try and determine if we
		// need to run the cleaners first
		needsCleaners, err := d.resourceGroupContainsResourceTypes(ctx, client, id, resourceTypes)
		if err != nil {
			return fmt.Errorf("determining if %s contains the resource types needed for cleaning: %+v", id, err)
		}

		if *needsCleaners {
			log.Printf("[DEBUG] Running Resource Group Cleaners for %s..", id)
			for _, cleaner := range ResourceGroupCleaners {
				log.Printf("[DEBUG] Running Resource Group Cleaner %q..", cleaner.Name())
				if err := cleaner.Cleanup(ctx, id, client, opts); err != nil {
					return fmt.Errorf("running Cleaner %q for %s: %+v", cleaner.Name(), id, err)
				}
			}
		} else {
			log.Printf("[DEBUG] Skipping Resource Group Cleaners for %s..", id)
		}

		log.Printf("[DEBUG]   Deleting Resource Group %q..", groupName)
		// NOTE: we're intentionally not using DeleteThenPoll since fire-and-forgetting these is fine
		if _, err := client.ResourceManager.ResourcesGroupsClient.Delete(ctx, id, resourcegroups.DefaultDeleteOperationOptions()); err != nil {
			log.Printf("[DEBUG]   Error during deletion of Resource Group %q: %s", groupName, err)
			continue
		}
		log.Printf("[DEBUG]   Deletion triggered for Resource Group %q", groupName)
	}

	return nil
}

func (d deleteResourceGroupsInSubscriptionCleaner) resourceGroupContainsResourceTypes(ctx context.Context, client *clients.AzureClient, id commonids.ResourceGroupId, resourceTypes []string) (*bool, error) {
	items := make([]string, 0)
	for _, resourceType := range resourceTypes {
		items = append(items, fmt.Sprintf("'%s'", resourceType))
	}
	query := fmt.Sprintf(`
resources
| where type in~ (%s)
| where resourceGroup =~ '%s'
| project id
| sort by (tolower(tostring(id))) asc
`, strings.Join(items, ", "), id.ResourceGroupName)
	payload := resources.QueryRequest{
		Options: &resources.QueryRequestOptions{
			Top: pointer.To(int64(1000)),
		},
		Query: query,
		Subscriptions: &[]string{
			id.SubscriptionId,
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

	itemsRaw, ok := resp.Model.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected the data to be an []interface but got %+v", resp.Model.Data)
	}

	containsItems := len(itemsRaw) > 0
	return pointer.To(containsItems), nil
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
