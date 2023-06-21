package dalek

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/managementgroups/2021-04-01/managementgroups"
	"github.com/hashicorp/go-uuid"
)

func (d *Dalek) ManagementGroups(ctx context.Context) error {
	if err := d.deleteManagementGroups(ctx); err != nil {
		return fmt.Errorf("processing Management Groups: %+v", err)
	}
	return nil
}

func (d *Dalek) deleteManagementGroups(ctx context.Context) error {
	// TODO: support prefix matching and actuallyDeleting
	client := d.client.ResourceManager.ManagementClient
	groups, err := client.List(ctx, managementgroups.DefaultListOperationOptions())
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

		if _, err := client.Delete(ctx, id, managementgroups.DefaultDeleteOperationOptions()); err != nil {
			log.Printf("[DEBUG]   Error during deletion of %s: %s", id, err)
			continue
		}
		log.Printf("[DEBUG]   Deleted %s", id)
	}
	return nil
}
