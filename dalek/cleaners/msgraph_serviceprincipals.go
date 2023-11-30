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

var _ MsGraphCleaner = servicePrincipalsCleaner{}

type servicePrincipalsCleaner struct{}

func (d servicePrincipalsCleaner) Name() string {
	return "Delete MS Graph Service Principals"
}

func (d servicePrincipalsCleaner) Cleanup(ctx context.Context, azure *clients.AzureClient, opts options.Options) error {
	if len(opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete Microsoft Graph Service Principals for safety; prefix not specified")
	}

	client := msgraph.NewServicePrincipalsClient()
	configureMsGraphClient(&client.BaseClient, azure)

	result, _, err := client.List(ctx, odata.Query{Filter: fmt.Sprintf("startsWith(displayName, '%s')", opts.Prefix)})
	if err != nil {
		return fmt.Errorf("listing Microsoft Graph Service Principals with prefix %q: %+v", opts.Prefix, err)
	}

	for _, item := range *result {
		if item.ID() == nil || item.DisplayName == nil {
			log.Printf("[DEBUG] Skipping deletion of Microsoft Graph Service Principal with nil ID or DisplayName")
			continue
		}

		id := *item.ID()
		displayName := *item.DisplayName

		if strings.TrimPrefix(*item.DisplayName, opts.Prefix) != *item.DisplayName {
			skip := false
			if item.AppMetadata != nil && item.AppMetadata.Data != nil {
				for _, datum := range *item.AppMetadata.Data {
					if datum.Key != nil && *datum.Key == "Identity" {
						log.Printf("[DEBUG] Skipping deletion of Microsoft Graph Service Principal %s (DisplayName: %q) as it is linked to a Managed Identity", id, displayName)
						skip = true
						break
					}
				}
			}
			if skip {
				continue
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted Microsoft Graph Service Principal %s (DisplayName: %q)", id, displayName)
				continue
			}

			log.Printf("[DEBUG] Deleting Microsoft Graph Service Principal %s (DisplayName: %q)...", id, displayName)
			if _, err = client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG] Failed to delete Microsoft Graph Service Principal %s (DisplayName: %q): %+v", id, displayName, err)
				continue
			}
		}
	}

	return nil
}
