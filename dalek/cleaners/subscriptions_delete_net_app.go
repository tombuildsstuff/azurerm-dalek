package cleaners

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2023-05-01/capacitypools"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2023-05-01/netappaccounts"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2023-05-01/volumes"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2023-05-01/volumesreplication"
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
	netAppVolumeReplicationClient := client.ResourceManager.NetAppVolumeReplicationClient

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

		accountIdForCapacityPool, err := capacitypools.ParseNetAppAccountID(*account.Id)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(accountIdForCapacityPool.ResourceGroupName, opts.Prefix) {
			log.Printf("[DEBUG] Not deleting %q as it does not match target RG prefix %q", *accountIdForCapacityPool, opts.Prefix)
			continue
		}

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted %s..", accountIdForCapacityPool)
			continue
		}

		capacityPoolList, err := netAppCapcityPoolClient.PoolsListComplete(ctx, *accountIdForCapacityPool)
		if err != nil {
			return fmt.Errorf("listing NetApp Capacity Pools for %s: %+v", accountIdForCapacityPool, err)
		}

		for _, capacityPool := range capacityPoolList.Items {
			if capacityPool.Id == nil {
				continue
			}

			capacityPoolForVolumesId, err := volumes.ParseCapacityPoolID(*capacityPool.Id)
			if err != nil {
				return err
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", capacityPoolForVolumesId)
				continue
			}

			volumeList, err := netAppVolumeClient.ListComplete(ctx, *capacityPoolForVolumesId)
			if err != nil {
				return fmt.Errorf("listing NetApp Volumes for %s: %+v", capacityPoolForVolumesId, err)
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

				volumeReplicationId, err := volumesreplication.ParseVolumeID(*volume.Id)
				if err != nil {
					return nil
				}

				if resp, err := netAppVolumeReplicationClient.VolumesDeleteReplication(ctx, *volumeReplicationId); err != nil {
					if !response.WasNotFound(resp.HttpResponse) {
						return fmt.Errorf("deleting replication for %s: %+v", volumeReplicationId, err)
					}
				}

				// sleeping because there is some eventual consistency for when the replication decouples from the volume
				time.Sleep(30 * time.Second)

				forceDelete := true
				if err = netAppVolumeClient.DeleteThenPoll(ctx, *volumeId, volumes.DeleteOperationOptions{ForceDelete: &forceDelete}); err != nil {
					// Potential Eventual Consistency Issues so we'll just log and move on
					log.Printf("[DEBUG] Unable to delete %s: %+v", volumeId, err)
				}
			}

			capacityPoolId, err := capacitypools.ParseCapacityPoolID(*capacityPool.Id)
			if err != nil {
				return err
			}

			if err = netAppCapcityPoolClient.PoolsDeleteThenPoll(ctx, *capacityPoolId); err != nil {
				// Potential Eventual Consistency Issues so we'll just log and move on
				log.Printf("[DEBUG] Unable to delete %s: %+v", capacityPoolId, err)
			}

			// sleeping because there is some eventual consistency for when the capacity pool decouples from the account
			time.Sleep(30 * time.Second)
		}

		accountId, err := netappaccounts.ParseNetAppAccountID(*account.Id)
		if err != nil {
			return err
		}

		if err = netAppAccountClient.AccountsDeleteThenPoll(ctx, *accountId); err != nil {
			// Potential Eventual Consistency Issues so we'll just log and move on
			log.Printf("[DEBUG] Unable to delete %s: %+v", accountId, err)
		}
	}

	return nil
}
