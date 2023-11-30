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

var _ MsGraphCleaner = namedLocationsCleaner{}

type namedLocationsCleaner struct{}

func (d namedLocationsCleaner) Name() string {
	return "Delete MS Graph Named Locations"
}

func (d namedLocationsCleaner) Cleanup(ctx context.Context, azure *clients.AzureClient, opts options.Options) error {
	// Named Locations have no user-specified description or other such field that we can overload, so for now we delete them all

	client := msgraph.NewNamedLocationsClient()
	configureMsGraphClient(&client.BaseClient, azure)

	result, _, err := client.List(ctx, odata.Query{})
	if err != nil {
		return fmt.Errorf("listing Microsoft Graph Named Locations with prefix %q: %+v", opts.Prefix, err)
	}

	for _, item := range *result {
		var id string
		if countryNamedLocation, ok := item.(msgraph.CountryNamedLocation); ok {
			if countryNamedLocation.ID == nil {
				log.Printf("[DEBUG] Skipping deletion of Microsoft Graph Country Named Location with nil ID or DisplayName")
				continue
			}
			id = *countryNamedLocation.ID
		} else if ipNamedLocation, ok := item.(msgraph.IPNamedLocation); ok {
			if ipNamedLocation.ID == nil {
				log.Printf("[DEBUG] Skipping deletion of Microsoft Graph IP Named Location with nil ID or DisplayName")
				continue
			}
			id = *ipNamedLocation.ID
		}

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted Microsoft Graph Named Location: %s", id)
			continue
		}

		if ipNamedLocation, ok := item.(msgraph.IPNamedLocation); ok {
			trusted := false
			properties := msgraph.IPNamedLocation{
				BaseNamedLocation: &msgraph.BaseNamedLocation{
					ID: ipNamedLocation.ID,
				},
				IsTrusted: &trusted,
			}
			log.Printf("[DEBUG] Marking Microsoft Graph IP Named Location %s as not trusted...", id)
			if _, err = client.UpdateIP(ctx, properties); err != nil {
				log.Printf("[DEBUG] Failed to update Microsoft Graph IP Named Location %s: %+v", id, err)
				continue
			}
		}

		log.Printf("[DEBUG] Deleting Microsoft Graph Named Location %s...", id)
		if _, err = client.Delete(ctx, id); err != nil {
			log.Printf("[DEBUG] Failed to delete Microsoft Graph Named Location %s: %+v", id, err)
			continue
		}
	}

	return nil
}
