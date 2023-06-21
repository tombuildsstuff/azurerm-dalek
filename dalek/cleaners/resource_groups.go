package cleaners

import (
	"context"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
)

var ResourceGroupCleaners = []ResourceGroupCleaner{
	removeLocksFromResourceGroupCleaner{},
}

type ResourceGroupCleaner interface {
	// Name returns the name of this ResourceGroupCleaner
	Name() string

	// Cleanup performs the cleanup operation for this ResourceGroupCleaner
	Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient) error
}
