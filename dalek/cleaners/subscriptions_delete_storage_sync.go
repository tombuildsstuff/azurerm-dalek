package cleaners

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storagesync/2020-03-01/cloudendpointresource"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storagesync/2020-03-01/storagesyncservicesresource"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storagesync/2020-03-01/syncgroupresource"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

type deleteStorageSyncSubscriptionCleaner struct{}

var _ SubscriptionCleaner = deleteStorageSyncSubscriptionCleaner{}

func (p deleteStorageSyncSubscriptionCleaner) Name() string {
	return "Removing Storage Sync"
}

func (p deleteStorageSyncSubscriptionCleaner) Cleanup(ctx context.Context, subscriptionId commonids.SubscriptionId, client *clients.AzureClient, opts options.Options) error {
	storageSyncClient := client.ResourceManager.StorageSyncClient
	storageSyncGroupClient := client.ResourceManager.StorageSyncGroupClient
	storageSyncCloudEndpointClient := client.ResourceManager.StorageSyncCloudEndpointClient

	storageSyncList, err := storageSyncClient.StorageSyncServicesListBySubscription(ctx, subscriptionId)
	if err != nil {
		return fmt.Errorf("listing storage syncs: %+v", err)
	}

	if storageSyncList.Model == nil || storageSyncList.Model.Value == nil {
		return fmt.Errorf("listing storage syncs: model/value was nil")
	}

	for _, storageSync := range *storageSyncList.Model.Value {
		if storageSync.Id == nil {
			continue
		}

		storageSyncForGroupId, err := syncgroupresource.ParseStorageSyncServiceID(*storageSync.Id)
		if err != nil {
			return err
		}
		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted %s..", storageSyncForGroupId)
			continue
		}

		groupList, err := storageSyncGroupClient.SyncGroupsListByStorageSyncService(ctx, *storageSyncForGroupId)
		if err != nil {
			return fmt.Errorf("listing storage sync groups for %s: %+v", storageSyncForGroupId, err)
		}

		if groupList.Model == nil || groupList.Model.Value == nil {
			continue
		}

		for _, group := range *groupList.Model.Value {
			if group.Id == nil {
				continue
			}

			groupIdForCloudEndpoint, err := cloudendpointresource.ParseSyncGroupID(*group.Id)
			if err != nil {
				return err
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", groupIdForCloudEndpoint)
				continue
			}

			cloudEndpointList, err := storageSyncCloudEndpointClient.CloudEndpointsListBySyncGroup(ctx, *groupIdForCloudEndpoint)
			if err != nil {
				return fmt.Errorf("listing cloud endpoints for %s: %+v", groupIdForCloudEndpoint, err)
			}

			if cloudEndpointList.Model == nil || cloudEndpointList.Model.Value == nil {
				continue
			}

			for _, endpoint := range *cloudEndpointList.Model.Value {
				if endpoint.Id == nil {
					continue
				}

				endpointId, err := cloudendpointresource.ParseCloudEndpointID(*endpoint.Id)
				if err != nil {
					return err
				}

				if !opts.ActuallyDelete {
					log.Printf("[DEBUG] Would have deleted %s..", endpointId)
					continue
				}

				if err = storageSyncCloudEndpointClient.CloudEndpointsDeleteThenPoll(ctx, *endpointId); err != nil {
					return fmt.Errorf("deleting %s: %+v", endpointId, err)
				}
			}

			groupId, err := syncgroupresource.ParseSyncGroupID(*group.Id)
			if err != nil {
				return err
			}

			if _, err = storageSyncGroupClient.SyncGroupsDelete(ctx, *groupId); err != nil {
				return fmt.Errorf("deleting %s: %+v", groupId, err)
			}
		}

		storageSyncId, err := storagesyncservicesresource.ParseStorageSyncServiceID(*storageSync.Id)
		if err = storageSyncClient.StorageSyncServicesDeleteThenPoll(ctx, *storageSyncId); err != nil {
			return fmt.Errorf("deleting %s: %+v", storageSyncId, err)
		}
	}

	return nil
}
