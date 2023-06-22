package dalek

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

func (d *Dalek) ActiveDirectory(ctx context.Context) error {
	log.Printf("[DEBUG] Preparing to delete AAD Service Principals")
	if err := d.deleteAADServicePrincipals(ctx); err != nil {
		return fmt.Errorf("deleting Service Principals: %+v", err)
	}

	log.Printf("[DEBUG] Preparing to delete Applications")
	if err := d.deleteAADApplications(ctx); err != nil {
		return fmt.Errorf("deleting Applications: %+v", err)
	}

	log.Printf("[DEBUG] Preparing to delete Groups")
	if err := d.deleteMicrosoftGraphGroups(ctx); err != nil {
		return fmt.Errorf("deleting Groups: %+v", err)
	}

	log.Printf("[DEBUG] Preparing to delete Users")
	if err := d.deleteAADUsers(ctx); err != nil {
		return fmt.Errorf("deleting Users: %+v", err)
	}

	return nil
}

func (d *Dalek) deleteAADApplications(ctx context.Context) error {
	if len(d.opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete AAD Applications for safety; prefix not specified")
	}

	client := d.client.ActiveDirectory.ApplicationsClient
	apps, err := client.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", d.opts.Prefix))
	if err != nil {
		return fmt.Errorf("listing AAD Applications with prefix %q: %+v", d.opts.Prefix, err)
	}

	for _, app := range apps.Values() {
		id := *app.ObjectID
		appID := *app.AppID
		displayName := *app.DisplayName

		if strings.TrimPrefix(displayName, d.opts.Prefix) != displayName {
			if !d.opts.ActuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Application %q (AppID: %s, ObjectId: %s)...", displayName, appID, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Application %q (AppID: %s, ObjID: %s): %s", displayName, appID, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
		}
	}

	return nil
}

func (d *Dalek) deleteMicrosoftGraphGroups(ctx context.Context) error {
	if len(d.opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete Microsoft Graph Groups for safety; prefix not specified")
	}

	client := d.client.MicrosoftGraph.Groups
	listFilter := odata.Query{
		Filter: fmt.Sprintf("startswith(displayName, '%s')", d.opts.Prefix),
	}
	groups, _, err := client.List(ctx, listFilter)
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list Microsoft Graph Groups with prefix: %q", d.opts.Prefix)
	}

	for _, group := range *groups {
		id := *group.ObjectId
		displayName := *group.DisplayName

		if strings.TrimPrefix(displayName, d.opts.Prefix) != displayName {
			if !d.opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted Microsoft Graph Group %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG] Deleting Microsoft Graph Group %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG] Error during deletion of Microsoft Graph Group %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG] Deleted Microsoft Graph Group %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (d *Dalek) deleteAADServicePrincipals(ctx context.Context) error {
	if len(d.opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete AAD Service Principals for safety; prefix not specified")
	}

	client := d.client.ActiveDirectory.ServicePrincipalsClient
	servicePrincipals, err := client.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", d.opts.Prefix))
	if err != nil {
		return fmt.Errorf("listing AAD Service Principals with prefix %q: %+v", d.opts.Prefix, err)
	}

	for _, servicePrincipal := range servicePrincipals.Values() {
		id := *servicePrincipal.ObjectID
		displayName := *servicePrincipal.DisplayName

		if strings.TrimPrefix(displayName, d.opts.Prefix) != displayName {
			if !d.opts.ActuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Service Principal %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Service Principal %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Service Principal %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Service Principal %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (d *Dalek) deleteAADUsers(ctx context.Context) error {
	if len(d.opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete AAD Users for safety; prefix not specified")
	}

	client := d.client.ActiveDirectory.UsersClient
	users, err := client.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", d.opts.Prefix), "")
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list AAD Users with prefix: %q", d.opts.Prefix)
	}

	for _, user := range users.Values() {
		id := *user.ObjectID
		displayName := *user.DisplayName

		if strings.TrimPrefix(displayName, d.opts.Prefix) != displayName {
			if !d.opts.ActuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD User %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD User %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD User %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD User %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}
