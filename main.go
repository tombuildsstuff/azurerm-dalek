package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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
	"github.com/tombuildsstuff/azurerm-dalek/clients"
)

func main() {
	log.Print("Starting Azure Dalek..")

	prefix := flag.String("prefix", "acctest", "-prefix=acctest")
	flag.Parse()

	credentials := clients.Credentials{
		ClientID:        os.Getenv("ARM_CLIENT_ID"),
		ClientSecret:    os.Getenv("ARM_CLIENT_SECRET"),
		SubscriptionID:  os.Getenv("ARM_SUBSCRIPTION_ID"),
		TenantID:        os.Getenv("ARM_TENANT_ID"),
		EnvironmentName: os.Getenv("ARM_ENVIRONMENT"),
		Endpoint:        os.Getenv("ARM_ENDPOINT"),
	}
	opts := DalekOptions{
		Prefix:                         *prefix,
		NumberOfResourceGroupsToDelete: int64(1000),
		ActuallyDelete:                 strings.EqualFold(os.Getenv("YES_I_REALLY_WANT_TO_DELETE_THINGS"), "true"),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
	defer cancel()
	if err := run(ctx, credentials, opts); err != nil {
		log.Fatalf(err.Error())
	}
}

type DalekOptions struct {
	Prefix                         string
	NumberOfResourceGroupsToDelete int64
	ActuallyDelete                 bool
}

func (o DalekOptions) String() string {
	return fmt.Sprintf("Prefix %q / Number RGs to Delete %d / Actually Delete %t", o.Prefix, o.NumberOfResourceGroupsToDelete, o.ActuallyDelete)
}

func run(ctx context.Context, credentials clients.Credentials, opts DalekOptions) error {
	sdkClient, err := clients.BuildAzureClient(ctx, credentials)
	if err != nil {
		return fmt.Errorf("building Azure Clients: %+v", err)
	}

	log.Printf("[DEBUG] Options: %s", opts)

	client := Dalek{
		client: *sdkClient,
	}

	log.Printf("[DEBUG] Preparing to delete Resource Groups..")
	if err = client.deleteResourceGroups(ctx, opts); err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete AAD Service Principals")
	if err = client.deleteAADServicePrincipals(ctx, opts); err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete AAD Applications")
	if err = client.deleteAADApplications(ctx, opts); err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete AAD Groups")
	if err = client.deleteAADGroups(ctx, opts); err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete AAD Users")
	if err = client.deleteAADUsers(ctx, opts); err != nil {
		panic(err)
	}

	log.Printf("[DEBUG] Preparing to delete Management Groups")
	if err = client.deleteManagementGroups(ctx, opts); err != nil {
		panic(err)
	}

	return nil
}

type Dalek struct {
	client clients.AzureClient
}

func (c Dalek) deleteAADApplications(ctx context.Context, opts DalekOptions) error {
	if len(opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete AAD Applications for safety; prefix not specified")
	}

	client := c.client.ActiveDirectory.ApplicationsClient
	apps, err := client.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", opts.Prefix))
	if err != nil {
		return fmt.Errorf("listing AAD Applications with prefix: %q", opts.Prefix)
	}

	for _, app := range apps.Values() {
		id := *app.ObjectID
		appID := *app.AppID
		displayName := *app.DisplayName

		if strings.TrimPrefix(displayName, opts.Prefix) != displayName {
			if !opts.ActuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Application %q (AppID: %s, ObjectId: %s)...", displayName, appID, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Application %q (AppID: %s, ObjID: %s): %s", displayName, appID, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
		}
	}

	return nil
}

func (c Dalek) deleteAADGroups(ctx context.Context, opts DalekOptions) error {
	if len(opts.Prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Groups for safety; prefix not specified")
	}

	client := c.client.ActiveDirectory.GroupsClient
	groups, err := client.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", opts.Prefix))
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list AAD Groups with prefix: %q", opts.Prefix)
	}

	for _, group := range groups.Values() {
		id := *group.ObjectID
		displayName := *group.DisplayName

		if strings.TrimPrefix(displayName, opts.Prefix) != displayName {
			if !opts.ActuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Group %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Group %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Group %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Group %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (c Dalek) deleteAADServicePrincipals(ctx context.Context, opts DalekOptions) error {
	if len(opts.Prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Service Principals for safety; prefix not specified")
	}

	client := c.client.ActiveDirectory.ServicePrincipalsClient
	servicePrincipals, err := client.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", opts.Prefix))
	if err != nil {
		return fmt.Errorf("listing AAD Service Principals with Prefix: %q", opts.Prefix)
	}

	for _, servicePrincipal := range servicePrincipals.Values() {
		id := *servicePrincipal.ObjectID
		displayName := *servicePrincipal.DisplayName

		if strings.TrimPrefix(displayName, opts.Prefix) != displayName {
			if !opts.ActuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Service Principal %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Service Principal %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Service Principal %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Service Principal %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (c Dalek) deleteAADUsers(ctx context.Context, opts DalekOptions) error {
	if len(opts.Prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Users for safety; prefix not specified")
	}

	client := c.client.ActiveDirectory.UsersClient
	users, err := client.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", opts.Prefix), "")
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list AAD Users with prefix: %q", opts.Prefix)
	}

	for _, user := range users.Values() {
		id := *user.ObjectID
		displayName := *user.DisplayName

		if strings.TrimPrefix(displayName, opts.Prefix) != displayName {
			if !opts.ActuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD User %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD User %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD User %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD User %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (c Dalek) deleteResourceGroups(ctx context.Context, opts DalekOptions) error {
	log.Printf("[DEBUG] Loading the first %d resource groups to delete", opts.NumberOfResourceGroupsToDelete)

	subscriptionId := commonids.NewSubscriptionID(c.client.AuthClient.SubscriptionID)
	listOpts := resourcegroups.ListOperationOptions{
		Top: pointer.To(opts.NumberOfResourceGroupsToDelete),
	}
	groups, err := c.client.ResourceManager.ResourcesClient.List(ctx, subscriptionId, listOpts)
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

		locks, lerr := c.client.ResourceManager.LocksClient.ListAtResourceGroupLevel(ctx, id, managementlocks.DefaultListAtResourceGroupLevelOperationOptions())
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

					if _, lerr = c.client.ResourceManager.LocksClient.DeleteByScope(ctx, *lockId); lerr != nil {
						log.Printf("[DEBUG]   Unable to delete lock %s on resource group %q", *lock.Name, groupName)
					}
				}
			}
		}

		log.Printf("[DEBUG]   Deleting Resource Group %q..", groupName)
		if _, err := c.client.ResourceManager.ResourcesClient.Delete(ctx, id, resourcegroups.DefaultDeleteOperationOptions()); err != nil {
			log.Printf("[DEBUG]   Error during deletion of Resource Group %q: %s", groupName, err)
			continue
		}
		log.Printf("[DEBUG]   Deletion triggered for Resource Group %q", groupName)
	}

	return nil
}

func (c Dalek) deleteManagementGroups(ctx context.Context, opts DalekOptions) error {
	// TODO: support prefix matching and actuallyDeleting
	groups, err := c.client.ResourceManager.ManagementClient.List(ctx, managementgroups.DefaultListOperationOptions())
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

		if _, err := c.client.ResourceManager.ManagementClient.Delete(ctx, id, managementgroups.DefaultDeleteOperationOptions()); err != nil {
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
