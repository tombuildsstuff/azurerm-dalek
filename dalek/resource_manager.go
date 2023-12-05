package dalek

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/cleaners"
)

func (d *Dalek) ResourceManager(ctx context.Context) (errors []error) {
	subscriptionId := commonids.NewSubscriptionID(d.client.SubscriptionID)
	for _, cleaner := range cleaners.SubscriptionCleaners {
		log.Printf("[DEBUG] Running Subscription Cleaner %q in %q", cleaner.Name(), subscriptionId)
		if err := cleaner.Cleanup(ctx, subscriptionId, d.client, d.opts); err != nil {
			errors = append(errors, fmt.Errorf("running Subscription Cleaner %q in %q: %+v", cleaner.Name(), subscriptionId, err))
		}
	}

	return
}
