package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2018-03-01/resources/mgmt/resources"
)

func main() {
	log.Print("Starting AzureRM Dalek..")

	client, err := buildArmClient()
	if err != nil {
		panic(fmt.Errorf("[ERROR] Unable to create Resource Groups Client: %+v", err))
		return
	}

	prefix := flag.String("prefix", "", "-prefix=acctest")

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
}

func (c ArmClient) deleteResourceGroups(ctx context.Context, numberOfResourceGroupsToDelete int, prefix string, actuallyDelete bool) error {
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
