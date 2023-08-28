package cleaners

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/notificationhubs/2017-04-01/namespaces"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

type notificationHubNamespacesCleaner struct{}

var _ SubscriptionCleaner = notificationHubNamespacesCleaner{}

func (p notificationHubNamespacesCleaner) Name() string {
	return "Removing New Relic"
}

func (p notificationHubNamespacesCleaner) Cleanup(ctx context.Context, subscriptionId commonids.SubscriptionId, client *clients.AzureClient, opts options.Options) error {
	namespacesClient := client.ResourceManager.NotificationHubNamespaceClient

	namespaceId := namespaces.NewNamespaceID(subscriptionId.SubscriptionId, "acctestRG-220927011356988888", "acctestnhn-220927011356988888")
	resourceGroupId := commonids.NewResourceGroupID(namespaceId.SubscriptionId, namespaceId.ResourceGroupName)
	namespaceList, err := namespacesClient.List(ctx, resourceGroupId)
	if err != nil {
		return err
	}

	return fmt.Errorf("stuff: %+v", *namespaceList.Model)

	/*
		for _, namespace := range namespaceList.Items {
			if namespace.Id == nil {
				continue
			}
			namespaceId, err := namespaces.ParseNamespaceID(*namespace.Id)
			if err != nil {
				return err
			}
			if err = namespacesClient.DeleteThenPoll(ctx, *namespaceId); err != nil {
				return fmt.Errorf("error deleting Notification Hub Namespace %s: %+v", namespaceId, err)
			}
		}
	*/
	return nil
}
