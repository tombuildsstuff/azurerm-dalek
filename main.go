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
	"github.com/manicminer/hamilton/odata"
)

func main() {
	log.Print("Starting Azure Dalek..")

	ctx := context.TODO()

	client, err := buildAzureClient(ctx)
	if err != nil {
		panic(fmt.Errorf("[ERROR] Unable to create Azure Clients: %+v", err))
		return
	}

	prefix := flag.String("prefix", "acctest", "-prefix=acctest")

	flag.Parse()

	log.Printf("[DEBUG] Required Prefix Match is %q", *prefix)

	//numberOfResourceGroupsToDelete := 1000
	actuallyDelete := strings.EqualFold(os.Getenv("YES_I_REALLY_WANT_TO_DELETE_THINGS"), "true")

	//log.Printf("[DEBUG] Preparing to delete Resource Groups (actually delete: %t)..", actuallyDelete)
	//err = client.deleteResourceGroups(ctx, numberOfResourceGroupsToDelete, *prefix, actuallyDelete)
	//if err != nil {
	//	panic(err)
	//}

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

	if c.msGraph != nil {
		log.Println("[DEBUG]   Using Microsoft Graph to delete Applications")
		apps, _, err := c.msGraph.applicationsClient.List(ctx, odata.Query{Filter: fmt.Sprintf("startswith(displayName, '%s')", prefix)})
		if err != nil {
			return fmt.Errorf("[ERROR] Unable to list AAD Applications with prefix %q: %v", prefix, err)
		}
		if apps == nil {
			return fmt.Errorf("[ERROR] Unable to list AAD Applications with prefix %q: apps result was nil", prefix)
		}

		for _, app := range *apps {
			if app.ID == nil || app.DisplayName == nil {
				log.Printf("[INFO]   Skipping AAD Application with nil ID or DisplayName")
			}
			if strings.TrimPrefix(*app.DisplayName, prefix) != *app.DisplayName {
				if !actuallyDelete {
					log.Printf("[DEBUG]   Would have deleted AAD Application %q (ID: %s)", *app.DisplayName, *app.ID)
					continue
				}

				log.Printf("[DEBUG]   Deleting AAD Application %q (ID: %s)...", *app.DisplayName, *app.ID)
				_, err := c.aadGraph.applicationsClient.Delete(ctx, *app.ID)
				if err != nil {
					log.Printf("[DEBUG]   Error during deletion of AAD Application %q (ID: %s): %v", *app.DisplayName, *app.ID, err)
					continue
				}
				log.Printf("[DEBUG]   Deleted AAD Application %q (ID: %s)", *app.DisplayName, *app.ID)
			}
		}
	} else {
		log.Println("[DEBUG]   Using Azure Active Directory Graph to delete Applications")
		apps, err := c.aadGraph.applicationsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
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
				_, err := c.aadGraph.applicationsClient.Delete(ctx, id)
				if err != nil {
					log.Printf("[DEBUG]   Error during deletion of AAD Application %q (AppID: %s, ObjID: %s): %s", displayName, appID, id, err)
					continue
				}
				log.Printf("[DEBUG]   Deleted AAD Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
			}
		}
	}

	return nil
}

func (c AzureClient) deleteAADGroups(ctx context.Context, prefix string, actuallyDelete bool) error {
	if len(prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Groups for safety; prefix not specified")
	}

	if c.msGraph != nil {
		log.Println("[DEBUG]   Using Microsoft Graph to delete Groups")

		groups, _, err := c.msGraph.groupsClient.List(ctx, odata.Query{Filter: fmt.Sprintf("startswith(displayName, '%s')", prefix)})
		if err != nil {
			return fmt.Errorf("[ERROR] Unable to list AAD Groups with prefix %q: %v", prefix, err)
		}
		if groups == nil {
			return fmt.Errorf("[ERROR] Unable to list AAD Groups with prefix %q: groups result was nil", prefix)
		}

		for _, group := range *groups {
			if group.ID == nil || group.DisplayName == nil {
				log.Printf("[INFO]   Skipping AAD Group with nil ID or DisplayName")
			}
			if strings.TrimPrefix(*group.DisplayName, prefix) != *group.DisplayName {
				if !actuallyDelete {
					log.Printf("[DEBUG]   Would have deleted AAD Group %q (ID: %s)", *group.DisplayName, *group.ID)
					continue
				}

				log.Printf("[DEBUG]   Deleting AAD Group %q (ID: %s)...", *group.DisplayName, *group.ID)
				_, err := c.aadGraph.groupsClient.Delete(ctx, *group.ID)
				if err != nil {
					log.Printf("[DEBUG]   Error during deletion of AAD Group %q (ID: %s): %v", *group.DisplayName, *group.ID, err)
					continue
				}
				log.Printf("[DEBUG]   Deleted AAD Group %q (ID: %s)", *group.DisplayName, *group.ID)
			}
		}
	} else {
		log.Println("[DEBUG]   Using Azure Active Directory Graph to delete Groups")

		groups, err := c.aadGraph.groupsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
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
				_, err := c.aadGraph.groupsClient.Delete(ctx, id)
				if err != nil {
					log.Printf("[DEBUG]   Error during deletion of AAD Group %q (ObjID: %s): %s", displayName, id, err)
					continue
				}
				log.Printf("[DEBUG]   Deleted AAD Group %q (ObjID: %s)", displayName, id)
			}
		}
	}

	return nil
}

func (c AzureClient) deleteAADServicePrincipals(ctx context.Context, prefix string, actuallyDelete bool) error {
	if len(prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Service Principals for safety; prefix not specified")
	}

	if c.msGraph != nil {
		log.Println("[DEBUG]   Using Microsoft Graph to delete Service Principals")

		servicePrincipals, _, err := c.msGraph.servicePrincipalsClient.List(ctx, odata.Query{Filter: fmt.Sprintf("startswith(displayName, '%s')", prefix)})
		if err != nil {
			return fmt.Errorf("[ERROR] Unable to list AAD Service Principals with prefix %q: %v", prefix, err)
		}
		if servicePrincipals == nil {
			return fmt.Errorf("[ERROR] Unable to list AAD Service Principals with prefix %q: servicePrincipals result was nil", prefix)
		}

		for _, servicePrincipal := range *servicePrincipals {
			if servicePrincipal.ID == nil || servicePrincipal.DisplayName == nil {
				log.Printf("[INFO]   Skipping AAD Service Principal with nil ID or DisplayName")
			}
			if strings.TrimPrefix(*servicePrincipal.DisplayName, prefix) != *servicePrincipal.DisplayName {
				if !actuallyDelete {
					log.Printf("[DEBUG]   Would have deleted AAD Service Principal %q (ID: %s)", *servicePrincipal.DisplayName, *servicePrincipal.ID)
					continue
				}

				log.Printf("[DEBUG]   Deleting AAD Service Principal %q (ID: %s)...", *servicePrincipal.DisplayName, *servicePrincipal.ID)
				_, err := c.aadGraph.servicePrincipalsClient.Delete(ctx, *servicePrincipal.ID)
				if err != nil {
					log.Printf("[DEBUG]   Error during deletion of AAD Service Principal %q (ID: %s): %v", *servicePrincipal.DisplayName, *servicePrincipal.ID, err)
					continue
				}
				log.Printf("[DEBUG]   Deleted AAD Service Principal %q (ID: %s)", *servicePrincipal.DisplayName, *servicePrincipal.ID)
			}
		}
	} else {
		log.Println("[DEBUG]   Using Azure Active Directory Graph to delete Service Principals")

		servicePrincipals, err := c.aadGraph.servicePrincipalsClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix))
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
				_, err := c.aadGraph.servicePrincipalsClient.Delete(ctx, id)
				if err != nil {
					log.Printf("[DEBUG]   Error during deletion of AAD Service Principal %q (ObjID: %s): %s", displayName, id, err)
					continue
				}
				log.Printf("[DEBUG]   Deleted AAD Service Principal %q (ObjID: %s)", displayName, id)
			}
		}
	}

	return nil
}

func (c AzureClient) deleteAADUsers(ctx context.Context, prefix string, actuallyDelete bool) error {
	if len(prefix) == 0 {
		return errors.New("[ERROR] Not proceeding to delete AAD Users for safety; prefix not specified")
	}

	if c.msGraph != nil {
		log.Println("[DEBUG]   Using Microsoft Graph to delete Users")

		users, _, err := c.msGraph.usersClient.List(ctx, odata.Query{Filter: fmt.Sprintf("startswith(displayName, '%s')", prefix)})
		if err != nil {
			return fmt.Errorf("[ERROR] Unable to list AAD Users with prefix %q: %v", prefix, err)
		}
		if users == nil {
			return fmt.Errorf("[ERROR] Unable to list AAD Users with prefix %q: users result was nil", prefix)
		}

		for _, user := range *users {
			if user.ID == nil || user.DisplayName == nil {
				log.Printf("[INFO]   Skipping AAD User with nil ID or DisplayName")
			}
			if strings.TrimPrefix(*user.DisplayName, prefix) != *user.DisplayName {
				if !actuallyDelete {
					log.Printf("[DEBUG]   Would have deleted AAD User %q (ID: %s)", *user.DisplayName, *user.ID)
					continue
				}

				log.Printf("[DEBUG]   Deleting AAD User %q (ID: %s)...", *user.DisplayName, *user.ID)
				_, err := c.aadGraph.usersClient.Delete(ctx, *user.ID)
				if err != nil {
					log.Printf("[DEBUG]   Error during deletion of AAD User %q (ID: %s): %v", *user.DisplayName, *user.ID, err)
					continue
				}
				log.Printf("[DEBUG]   Deleted AAD User %q (ID: %s)", *user.DisplayName, *user.ID)
			}
		}
	} else {
		log.Println("[DEBUG]   Using Azure Active Directory Graph to delete Users")

		users, err := c.aadGraph.usersClient.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", prefix), "")
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
				_, err := c.aadGraph.usersClient.Delete(ctx, id)
				if err != nil {
					log.Printf("[DEBUG]   Error during deletion of AAD User %q (ObjID: %s): %s", displayName, id, err)
					continue
				}
				log.Printf("[DEBUG]   Deleted AAD User %q (ObjID: %s)", displayName, id)
			}
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
