package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2018-03-01/resources/mgmt/resources"
)

func main() {
	log.Print("Starting Azure Dalek..")

	client, err := buildAzureClient()
	if err != nil {
		panic(fmt.Errorf("[ERROR] Unable to create Azure Clients: %+v", err))
		return
	}

	prefix := flag.String("prefix", "acctest", "-prefix=acctest")

	flag.Parse()

	log.Printf("[DEBUG] Required Prefix Match is %q", *prefix)

	ctx := context.TODO()
	numberOfResourceGroupsToDelete := 1000
	actuallyDelete := strings.EqualFold(os.Getenv("YES_I_REALLY_WANT_TO_DELETE_THINGS"), "true")

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
}

func (c AzureClient) deleteAADApplications(ctx context.Context, prefix string, actuallyDelete bool) error {
	if len(prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Applications for safety; prefix not specified")
	}

	apps, err := c.applicationsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
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
			_, err := c.applicationsClient.Delete(ctx, id)
			if err != nil {
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

	groups, err := c.groupsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
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
			_, err := c.groupsClient.Delete(ctx, id)
			if err != nil {
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

	servicePrincipals, err := c.servicePrincipalsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
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
			_, err := c.servicePrincipalsClient.Delete(ctx, id)
			if err != nil {
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

	users, err := c.usersClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
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
			_, err := c.usersClient.Delete(ctx, id)
			if err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD User %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD User %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (c AzureClient) deleteResourceGroups(ctx context.Context, numberOfResourceGroupsToDelete int, prefix string, actuallyDelete bool) error {
	max := int32(numberOfResourceGroupsToDelete)
	log.Printf("[DEBUG] Loading the first %d resource groups to delete", numberOfResourceGroupsToDelete)
	groups, err := c.resourcesClient.List(ctx, "", &max)
	if err != nil {
		return fmt.Errorf("[ERROR] Error obtaining Resource List: %+v", err)
	}

	for _, resource := range groups.Values() {
		groupName := *resource.Name
		log.Printf("[DEBUG] Resource Group: %q", groupName)

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

		locks, lerr := c.locksClient.ListAtResourceGroupLevel(ctx, groupName, "")
		if lerr != nil {
			log.Printf("[DEBUG] Error obtaining Resource Group Locks : %+v", err)
		} else {
			for _, lock := range locks.Values() {
				if lock.ID == nil {
					log.Printf("[DEBUG]   Lock with nil id on %q", groupName)
					continue
				}
				id := *lock.ID

				if lock.Name == nil {
					log.Printf("[DEBUG]   Lock %s with nil name on %q", id, groupName)
					continue
				}

				log.Printf("[DEBUG]   Atemping to remove lock %s from : %s", id, groupName)
				parts := strings.Split(id, "/providers/Microsoft.Authorization/locks/")
				if len(parts) != 2 {
					log.Printf("[DEBUG]   Error splitting %s on /providers/Microsoft.Authorization/locks/", id)
				}
				scope := parts[0]
				name := parts[1]

				if _, lerr = c.locksClient.DeleteByScope(ctx, scope, name); lerr != nil {
					log.Printf("[DEBUG]   Unable to delete lock %s on resource group %q", *lock.Name, groupName)
				}
			}
		}

		log.Printf("[DEBUG]   Deleting Resource Group %q..", groupName)
		_, err := c.resourcesClient.Delete(ctx, groupName)
		if err != nil {
			log.Printf("[DEBUG]   Error during deletion of Resource Group %q: %s", groupName, err)
			continue
		}
		log.Printf("[DEBUG]   Deletion triggered for Resource Group %q", groupName)
	}

	return nil
}

func shouldDeleteResourceGroup(input resources.Group, prefix string) bool {
	if prefix != "" {
		if !strings.HasPrefix(strings.ToLower(*input.Name), strings.ToLower(prefix)) {
			return false
		}
	}

	for k := range input.Tags {
		if strings.EqualFold(k, "donotdelete") {
			return false
		}
	}

	return true
}
