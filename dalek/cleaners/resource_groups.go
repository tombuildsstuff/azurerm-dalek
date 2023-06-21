package cleaners

import (
	"context"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
)

var ResourceGroupCleaners = []ResourceGroupCleaner{
	// NOTE: the ordering is important here, we want to remove Locks first because a Write or Delete lock
	// would prevent us from doing anything else, so that needs to be first.
	removeLocksFromResourceGroupCleaner{},
	serviceBusNamespaceBreakPairingCleaner{},
}

type ResourceGroupCleaner interface {
	// Name returns the name of this ResourceGroupCleaner
	Name() string

	// Cleanup performs the cleanup operation for this ResourceGroupCleaner
	Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient) error
}
