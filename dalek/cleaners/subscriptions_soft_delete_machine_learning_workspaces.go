package cleaners

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/machinelearningservices/2023-04-01-preview/workspaces"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

var _ SubscriptionCleaner = purgeSoftDeletedMachineLearningWorkspacesInSubscriptionCleaner{}

type purgeSoftDeletedMachineLearningWorkspacesInSubscriptionCleaner struct {
}

func (p purgeSoftDeletedMachineLearningWorkspacesInSubscriptionCleaner) Name() string {
	return "Purging Soft Deleted Machine Learning Workspaces in Subscription"
}

func (p purgeSoftDeletedMachineLearningWorkspacesInSubscriptionCleaner) Cleanup(ctx context.Context, subscriptionId commonids.SubscriptionId, client *clients.AzureClient, opts options.Options) error {
	softDeletedWorkspaces, err := client.ResourceManager.MachineLearningWorkspacesClient.ListBySubscriptionComplete(ctx, subscriptionId, workspaces.DefaultListBySubscriptionOperationOptions())
	if err != nil {
		return fmt.Errorf("loading the Machine Learning Workspaces within %s: %+v", subscriptionId, err)
	}

	for _, workspace := range softDeletedWorkspaces.Items {
		workspaceId, err := workspaces.ParseWorkspaceIDInsensitively(*workspace.Id)
		if err != nil {
			return fmt.Errorf("parsing Machine Learning Workspace ID %q: %+v", *workspace.Id, err)
		}
		log.Printf("[DEBUG] Purging Soft-Deleted %s..", *workspaceId)
		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have purged soft-deleted Machine Learning Workspace %q..", *workspaceId)
			continue
		}

		purge := true
		log.Printf("[DEBUG] Purging Soft-Deleted %s..", *workspaceId)
		if err := client.ResourceManager.MachineLearningWorkspacesClient.DeleteThenPoll(ctx, *workspaceId, workspaces.DeleteOperationOptions{ForceToPurge: &purge}); err != nil {
			return fmt.Errorf("purging %s: %+v", *workspaceId, err)
		}
		log.Printf("[DEBUG] Purged Soft-Deleted %s.", *workspaceId)
	}
	return nil
}
