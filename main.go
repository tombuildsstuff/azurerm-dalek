package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"log"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/managementgroups/2021-04-01/managementgroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2020-05-01/managementlocks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	"github.com/hashicorp/go-uuid"
)

func main() {
	log.Print("Starting Azure Dalek..")

	credentials := clients.Credentials{
		ClientID:        os.Getenv("ARM_CLIENT_ID"),
		ClientSecret:    os.Getenv("ARM_CLIENT_SECRET"),
		SubscriptionID:  os.Getenv("ARM_SUBSCRIPTION_ID"),
		TenantID:        os.Getenv("ARM_TENANT_ID"),
		EnvironmentName: os.Getenv("ARM_ENVIRONMENT"),
		Endpoint:        os.Getenv("ARM_ENDPOINT"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	sdkClient, err := clients.BuildAzureClient(ctx, credentials)
	if err != nil {
		panic(fmt.Errorf("[ERROR] Unable to create Azure Clients: %+v", err))
		return
	}

	prefix := flag.String("prefix", "acctest", "-prefix=acctest")
	flag.Parse()

	log.Printf("[DEBUG] Required Prefix Match is %q", *prefix)

	numberOfResourceGroupsToDelete := 1000
	actuallyDelete := strings.EqualFold(os.Getenv("YES_I_REALLY_WANT_TO_DELETE_THINGS"), "true")

	client := AzureClient{
		client: *sdkClient,
	}

	log.Printf("[DEBUG] Preparing to delete Resource Groups (actually delete: %t)..", actuallyDelete)
	err = client.deleteResourceGroups(ctx, numberOfResourceGroupsToDelete, *prefix, actuallyDelete)
	if err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete AAD Service Principals (actually delete: %t)..", actuallyDelete)
	err = client.deleteAADServicePrincipals(ctx, *prefix, actuallyDelete)
	if err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete AAD Applications (actually delete: %t)..", actuallyDelete)
	err = client.deleteAADApplications(ctx, *prefix, actuallyDelete)
	if err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete AAD Groups (actually delete: %t)..", actuallyDelete)
	err = client.deleteAADGroups(ctx, *prefix, actuallyDelete)
	if err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete AAD Users (actually delete: %t)..", actuallyDelete)
	err = client.deleteAADUsers(ctx, *prefix, actuallyDelete)
	if err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete Management Groups (actually delete: %t)..", actuallyDelete)
	err = client.deleteManagementGroups(ctx, *prefix, actuallyDelete)
	if err != nil {
		panic(err)
	}

}

type AzureClient struct {
	client clients.AzureClient
}

func (c AzureClient) deleteAADApplications(ctx context.Context, prefix string, actuallyDelete bool) error {
	if len(prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Applications for safety; prefix not specified")
	}

	apps, err := c.client.ApplicationsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list AAD Applications with prefix: %q", prefix)
	}

	for _, app := range apps.Values() {
		id := *app.ObjectID
		appID := *app.AppID
		displayName := *app.DisplayName

		if strings.TrimPrefix(displayName, prefix) != displayName {
			if !actuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Application %q (AppID: %s, ObjectId: %s)...", displayName, appID, id)
			if _, err := c.client.ApplicationsClient.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Application %q (AppID: %s, ObjID: %s): %s", displayName, appID, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
		}
	}

	return nil
}

func (c AzureClient) deleteAADGroups(ctx context.Context, prefix string, actuallyDelete bool) error {
	if len(prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Groups for safety; prefix not specified")
	}

	groups, err := c.client.GroupsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list AAD Groups with prefix: %q", prefix)
	}

	for _, group := range groups.Values() {
		id := *group.ObjectID
		displayName := *group.DisplayName

		if strings.TrimPrefix(displayName, prefix) != displayName {
			if !actuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Group %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Group %q (ObjectId: %s)...", displayName, id)
			if _, err := c.client.GroupsClient.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Group %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Group %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (c AzureClient) deleteAADServicePrincipals(ctx context.Context, prefix string, actuallyDelete bool) error {
	if len(prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Service Principals for safety; prefix not specified")
	}

	servicePrincipals, err := c.client.ServicePrincipalsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list AAD Service Principals with prefix: %q", prefix)
	}

	for _, servicePrincipal := range servicePrincipals.Values() {
		id := *servicePrincipal.ObjectID
		displayName := *servicePrincipal.DisplayName

		if strings.TrimPrefix(displayName, prefix) != displayName {
			if !actuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Service Principal %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Service Principal %q (ObjectId: %s)...", displayName, id)
			if _, err := c.client.ServicePrincipalsClient.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Service Principal %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Service Principal %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (c AzureClient) deleteAADUsers(ctx context.Context, prefix string, actuallyDelete bool) error {
	if len(prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Users for safety; prefix not specified")
	}

	users, err := c.client.UsersClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix), "")
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list AAD Users with prefix: %q", prefix)
	}

	for _, user := range users.Values() {
		id := *user.ObjectID
		displayName := *user.DisplayName

		if strings.TrimPrefix(displayName, prefix) != displayName {
			if !actuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD User %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD User %q (ObjectId: %s)...", displayName, id)
			if _, err := c.client.UsersClient.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD User %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD User %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (c AzureClient) deleteResourceGroups(ctx context.Context, numberOfResourceGroupsToDelete int, prefix string, actuallyDelete bool) error {
	log.Printf("[DEBUG] Loading the first %d resource groups to delete", numberOfResourceGroupsToDelete)

	subscriptionId := commonids.NewSubscriptionID(c.client.AuthClient.SubscriptionID)
	opts := resourcegroups.ListOperationOptions{
		Top: pointer.To(int64(numberOfResourceGroupsToDelete)),
	}
	groups, err := c.client.ResourcesClient.List(ctx, subscriptionId, opts)
	if err != nil {
		return fmt.Errorf("[ERROR] Error obtaining Resource List: %+v", err)
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

		if !shouldDeleteResourceGroup(resource, prefix) {
			log.Println("[DEBUG]   Shouldn't Delete - Skipping..")
			continue
		}

		if !actuallyDelete {
			log.Printf("[DEBUG]   Would have deleted group %q..", groupName)
			continue
		}

		locks, lerr := c.client.LocksClient.ListAtResourceGroupLevel(ctx, id, managementlocks.DefaultListAtResourceGroupLevelOperationOptions())
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

					if _, lerr = c.client.LocksClient.DeleteByScope(ctx, *lockId); lerr != nil {
						log.Printf("[DEBUG]   Unable to delete lock %s on resource group %q", *lock.Name, groupName)
					}
				}
			}
		}

		log.Printf("[DEBUG]   Deleting Resource Group %q..", groupName)
		if _, err := c.client.ResourcesClient.Delete(ctx, id, resourcegroups.DefaultDeleteOperationOptions()); err != nil {
			log.Printf("[DEBUG]   Error during deletion of Resource Group %q: %s", groupName, err)
			continue
		}
		log.Printf("[DEBUG]   Deletion triggered for Resource Group %q", groupName)
	}

	return nil
}

func (c AzureClient) deleteManagementGroups(ctx context.Context, prefix string, actuallyDelete bool) error {
	var listOpts managementgroups.ListOperationOptions
	groups, err := c.client.ManagementClient.List(ctx, listOpts)
	if err != nil {
		return fmt.Errorf("[ERROR] Error obtaining Management Groups List: %+v", err)
	}

	if groups.Model == nil {
		log.Printf("[DEBUG]   No Management Groups found")
		return nil
	}
	for _, group := range *groups.Model {
		if group.Name == nil || group.Id == nil {
			continue
		}

		groupName := *group.Name
		id := commonids.NewManagementGroupID(*group.Id)

		if _, err := uuid.ParseUUID(groupName); err != nil {
			log.Printf("[DEBUG]   Skipping Management Group %q", groupName)
			continue
		}
		log.Printf("[DEBUG]   Deleting %s", id)

		if _, err := c.client.ManagementClient.Delete(ctx, id, managementgroups.DefaultDeleteOperationOptions()); err != nil {
			log.Printf("[DEBUG]   Error during deletion of %s: %s", id, err)
			continue
		}
		log.Printf("[DEBUG]   Deleted %s", id)
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
