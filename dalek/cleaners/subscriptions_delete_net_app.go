package cleaners

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2022-05-01/capacitypools"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2022-05-01/volumes"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

type deleteNetAppSubscriptionCleaner struct{}

var _ SubscriptionCleaner = deleteNetAppSubscriptionCleaner{}

func (p deleteNetAppSubscriptionCleaner) Name() string {
	return "Removing Net App"
}

func (p deleteNetAppSubscriptionCleaner) Cleanup(ctx context.Context, subscriptionId commonids.SubscriptionId, client *clients.AzureClient, opts options.Options) error {
	netAppAccountClient := client.ResourceManager.NetAppAccountClient
	netAppCapcityPoolClient := client.ResourceManager.NetAppCapacityPoolClient
	netAppVolumeClient := client.ResourceManager.NetAppVolumeClient

	accountLists, err := netAppAccountClient.AccountsListBySubscription(ctx, subscriptionId)
	if err != nil {
		return fmt.Errorf("listing NetApp Accounts for %s: %+v", subscriptionId, err)
	}

	if accountLists.Model == nil {
		return fmt.Errorf("listing NetApp Accounts: model was nil")
	}

	for _, account := range *accountLists.Model {
		if account.Id == nil {
			continue
		}

		accountId, err := capacitypools.ParseNetAppAccountID(*account.Id)
		if err != nil {
			return err
		}

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted %s..", accountId)
			continue
		}

		capacityPoolList, err := netAppCapcityPoolClient.PoolsListComplete(ctx, *accountId)
		if err != nil {
			return fmt.Errorf("listing NetApp Capacity Pools for %s: %+v", accountId, err)
		}

		for _, capacityPool := range capacityPoolList.Items {
			if capacityPool.Id == nil {
				continue
			}

			capacityPoolId, err := volumes.ParseCapacityPoolID(*capacityPool.Id)
			if err != nil {
				return err
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", capacityPoolId)
				continue
			}

			volumeList, err := netAppVolumeClient.ListComplete(ctx, *capacityPoolId)
			if err != nil {
				return fmt.Errorf("listing NetApp Volumes for %s: %+v", capacityPoolId, err)
			}

			for _, volume := range volumeList.Items {
				if volume.Id == nil {
					continue
				}

				volumeId, err := volumes.ParseVolumeID(*volume.Id)
				if err != nil {
					return err
				}

				if !opts.ActuallyDelete {
					log.Printf("[DEBUG] Would have deleted %s..", volumeId)
					continue
				}

				forceDelete := true
				if err = netAppVolumeClient.DeleteThenPoll(ctx, *volumeId, volumes.DeleteOperationOptions{ForceDelete: &forceDelete}); err != nil {
					return fmt.Errorf("deleting %s: %+v", volumeId, err)
				}
			}

		}
	}

	return nil
}
