package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2017-03-09/resources/mgmt/resources"
)

func main() {
	log.Print("Starting AzureRM Dalek..")

	client, err := buildArmClient()
	if err != nil {
		log.Printf("[ERROR] Unable to create Resource Groups Client: %+v", err)
		return
	}

	ctx := context.TODO()
	numberOfResourceGroupsToDelete := 1000
	actuallyDelete := os.Getenv("YES_I_REALLY_WANT_TO_DELETE_THINGS") != ""

	log.Printf("[DEBUG] Preparing to delete Resource Groups..")
	err = client.deleteResourceGroups(ctx, numberOfResourceGroupsToDelete, actuallyDelete)
	if err != nil {
		panic(err)
	}
}

func (c ArmClient) deleteResourceGroups(ctx context.Context, numberOfResourceGroupsToDelete int, actuallyDelete bool) error {
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

		if !shouldDeleteResourceGroup(resource) {
			log.Println("[DEBUG]   Not a Test Resource Group - Skipping..")
			continue
		}

		if !actuallyDelete {
			log.Printf("[DEBUG]   Would have deleted group %q..", groupName)
			continue
		}

		log.Printf("[DEBUG]   Deleting Resource Group %q..", groupName)
		_, err := c.resourcesClient.Delete(ctx, groupName)
		if err != nil {
			return fmt.Errorf("Error deleting Resource Group %q: %+v", groupName, err)
		}
		log.Printf("[DEBUG]   Deletion triggered for Resource Group %q", groupName)
	}

	return nil
}

func shouldDeleteResourceGroup(input resources.Group) bool {
	for k, _ := range input.Tags {
		if strings.EqualFold(k, "donotdelete") {
			return false
		}
	}

	return true
}
