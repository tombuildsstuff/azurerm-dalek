package cleaners

import (
	"context"
	"fmt"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/newrelic/2022-07-01/monitors"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
)

type newRelicCleaner struct{}

var _ SubscriptionCleaner = newRelicCleaner{}

func (p newRelicCleaner) Name() string {
	return "Removing New Relic"
}

func (p newRelicCleaner) Cleanup(ctx context.Context, subscriptionId commonids.SubscriptionId, client *clients.AzureClient, opts options.Options) error {
	newRelicClient := client.ResourceManager.NewRelicClient

	monitorList, err := newRelicClient.ListBySubscriptionComplete(ctx, subscriptionId)
	if err != nil {
		return err
	}

	for _, monitor := range monitorList.Items {
		if monitor.Id == nil {
			continue
		}
		monitorId, err := monitors.ParseMonitorID(*monitor.Id)
		if err != nil {
			return err
		}
		if err = newRelicClient.DeleteThenPoll(ctx, *monitorId, monitors.DefaultDeleteOperationOptions()); err != nil {
			return fmt.Errorf("error deleting New Relic Monitor %s: %+v", monitorId, err)
		}
	}

	return nil
}
