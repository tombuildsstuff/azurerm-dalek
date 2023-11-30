package cleaners

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
	"github.com/manicminer/hamilton/msgraph"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

var _ MsGraphCleaner = applicationsCleaner{}

type applicationsCleaner struct{}

func (d applicationsCleaner) Name() string {
	return "Delete MS Graph Applications"
}

func (d applicationsCleaner) Cleanup(ctx context.Context, azure *clients.AzureClient, opts options.Options) error {
	if len(opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete Microsoft Graph Applications for safety; prefix not specified")
	}

	client := msgraph.NewApplicationsClient()
	configureMsGraphClient(&client.BaseClient, azure)

	result, _, err := client.List(ctx, odata.Query{Filter: fmt.Sprintf("startsWith(displayName, '%s')", opts.Prefix)})
	if err != nil {
		return fmt.Errorf("listing Microsoft Graph Applications with prefix %q: %+v", opts.Prefix, err)
	}

	for _, item := range *result {
		if item.ID() == nil || item.DisplayName == nil {
			log.Printf("[DEBUG] Skipping deletion of Microsoft Graph Application with nil ID or DisplayName")
			continue
		}

		id := *item.ID()
		displayName := *item.DisplayName

		if strings.TrimPrefix(*item.DisplayName, opts.Prefix) != *item.DisplayName {
			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted Microsoft Graph Application %s (DisplayName: %q)", id, displayName)
				continue
			}

			log.Printf("[DEBUG] Deleting Microsoft Graph Application %s (DisplayName: %q)...", id, displayName)
			if _, err = client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG] Failed to delete Microsoft Graph Application %s (DisplayName: %q): %+v", id, displayName, err)
				continue
			}

			log.Printf("[DEBUG] Permanently deleting Microsoft Graph Application %s (DisplayName: %q)...", id, displayName)
			if _, err = client.DeletePermanently(ctx, id); err != nil {
				log.Printf("[DEBUG] Failed to permanently delete Microsoft Graph Application %s (DisplayName: %q): %+v", id, displayName, err)
			}
		}
	}

	return nil
}
