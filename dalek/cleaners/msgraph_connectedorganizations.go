package cleaners

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/manicminer/hamilton/msgraph"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

var _ MsGraphCleaner = connectedOrganizationsCleaner{}

type connectedOrganizationsCleaner struct{}

func (d connectedOrganizationsCleaner) Name() string {
	return "Delete MS Graph Connected Organizations"
}

func (d connectedOrganizationsCleaner) Cleanup(ctx context.Context, azure *clients.AzureClient, opts options.Options) error {
	// Connected Organizations have no user-specified description or other such field that we can overload, so for now we delete them all

	client := msgraph.NewConnectedOrganizationClient()
	configureMsGraphClient(&client.BaseClient, azure)

	result, _, err := client.List(ctx, odata.Query{Filter: fmt.Sprintf("startsWith(displayName, '%s')", opts.Prefix)})
	if err != nil {
		return fmt.Errorf("listing Microsoft Graph Connected Organizations with prefix %q: %+v", opts.Prefix, err)
	}

	for _, item := range *result {
		if item.ID == nil {
			log.Printf("[DEBUG] Skipping deletion of Microsoft Graph Connected Organization with nil ID or DisplayName")
			continue
		}

		id := *item.ID

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted Microsoft Graph Connected Organization: %s", id)
			continue
		}

		log.Printf("[DEBUG] Deleting Microsoft Graph Connected Organization %s...", id)
		if _, err = client.Delete(ctx, id); err != nil {
			log.Printf("[DEBUG] Failed to delete Microsoft Graph Connected Organization %s: %+v", id, err)
			continue
		}
	}

	return nil
}
