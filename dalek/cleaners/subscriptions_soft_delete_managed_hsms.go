package cleaners

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-02-01/managedhsms"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

var _ SubscriptionCleaner = purgeSoftDeletedManagedHSMsInSubscriptionCleaner{}

type purgeSoftDeletedManagedHSMsInSubscriptionCleaner struct {
}

func (p purgeSoftDeletedManagedHSMsInSubscriptionCleaner) Name() string {
	return "Purging Soft Deleted Key Vaults in Subscription"
}

func (p purgeSoftDeletedManagedHSMsInSubscriptionCleaner) Cleanup(ctx context.Context, subscriptionId commonids.SubscriptionId, client *clients.AzureClient, opts options.Options) error {
	softDeletedHSMs, err := client.ResourceManager.ManagedHSMsClient.ListDeletedComplete(ctx, subscriptionId)
	if err != nil {
		return fmt.Errorf("loading the Soft-Deleted Managed HSMs within %s: %+v", subscriptionId, err)
	}
	for _, hsm := range softDeletedHSMs.Items {
		hsmId, err := managedhsms.ParseDeletedManagedHSMIDInsensitively(*hsm.Id)
		if err != nil {
			return fmt.Errorf("parsing Managed HSM ID %q: %+v", *hsm.Id, err)
		}
		log.Printf("[DEBUG] Purging Soft-Deleted %s..", *hsmId)

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have purged soft-deleted Managed HSM %q..", *hsmId)
			continue
		}

		log.Printf("[DEBUG] Purging Soft-Deleted %s..", *hsmId)
		if err := client.ResourceManager.ManagedHSMsClient.PurgeDeletedThenPoll(ctx, *hsmId); err != nil {
			return fmt.Errorf("purging %s: %+v", *hsmId, err)
		}
		log.Printf("[DEBUG] Purged Soft-Deleted %s.", *hsmId)
	}
	return nil
}
