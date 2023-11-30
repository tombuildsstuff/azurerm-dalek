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

var _ MsGraphCleaner = accessPackageAssignmentRequestsCleaner{}

type accessPackageAssignmentRequestsCleaner struct{}

func (d accessPackageAssignmentRequestsCleaner) Name() string {
	return "Delete MS Graph Access Package Assignment Requests"
}

func (d accessPackageAssignmentRequestsCleaner) Cleanup(ctx context.Context, azure *clients.AzureClient, opts options.Options) error {
	// Assignment requests have no user-specified description or other such field that we can overload, so for now we delete them all

	client := msgraph.NewAccessPackageAssignmentRequestClient()
	configureMsGraphClient(&client.BaseClient, azure)

	result, _, err := client.List(ctx, odata.Query{})
	if err != nil {
		return fmt.Errorf("listing Microsoft Graph Access Package Assignment Requests with prefix %q: %+v", opts.Prefix, err)
	}

	for _, item := range *result {
		if item.ID == nil {
			log.Printf("[DEBUG] Skipping deletion of Microsoft Graph Access Package Assignment Request with nil ID")
			continue
		}

		id := *item.ID

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted Microsoft Graph Access Package Assignment Request %s", id)
			continue
		}

		log.Printf("[DEBUG] Deleting Microsoft Graph Access Package Assignment Request %s...", id)
		if _, err = client.Delete(ctx, id); err != nil {
			log.Printf("[DEBUG] Failed to delete Microsoft Graph Access Package Assignment Request %s: %+v", id, err)
			continue
		}
	}

	return nil
}
