package dalek

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-azure-sdk/sdk/odata"
)

func (d *Dalek) ActiveDirectory(ctx context.Context) error {
	log.Printf("[DEBUG] Preparing to delete Service Principals")
	if err := d.deleteMicrosoftGraphServicePrincipals(ctx); err != nil {
		return fmt.Errorf("deleting Service Principals: %+v", err)
	}

	log.Printf("[DEBUG] Preparing to delete Applications")
	if err := d.deleteMicrosoftGraphApplications(ctx); err != nil {
		return fmt.Errorf("deleting Applications: %+v", err)
	}

	log.Printf("[DEBUG] Preparing to delete Groups")
	if err := d.deleteMicrosoftGraphGroups(ctx); err != nil {
		return fmt.Errorf("deleting Groups: %+v", err)
	}

	log.Printf("[DEBUG] Preparing to delete Users")
	if err := d.deleteMicrosoftGraphUsers(ctx); err != nil {
		return fmt.Errorf("deleting Users: %+v", err)
	}

	return nil
}

func (d *Dalek) deleteMicrosoftGraphApplications(ctx context.Context) error {
	if len(d.opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete Microsoft Graph Applications for safety; prefix not specified")
	}

	client := d.client.MicrosoftGraph.Applications
	listFilter := odata.Query{
		Filter: fmt.Sprintf("startswith(displayName, '%s')", d.opts.Prefix),
	}
	apps, _, err := client.List(ctx, listFilter)
	if err != nil {
		return fmt.Errorf("listing Microsoft Graph Applications with prefix %q: %+v", d.opts.Prefix, err)
	}

	for _, app := range *apps {
		id := *app.ObjectId
		appID := *app.AppId
		displayName := *app.DisplayName

		if strings.TrimPrefix(displayName, d.opts.Prefix) != displayName {
			if !d.opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted Microsoft Graph Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
				continue
			}

			log.Printf("[DEBUG] Deleting Microsoft Graph Application %q (AppID: %s, ObjectId: %s)...", displayName, appID, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG] Error during deletion of Microsoft Graph Application %q (AppID: %s, ObjID: %s): %s", displayName, appID, id, err)
				continue
			}
			log.Printf("[DEBUG] Deleted Microsoft Graph Application %q (AppID: %s, ObjID: %s)", displayName, appID, id)
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

func (d *Dalek) deleteMicrosoftGraphServicePrincipals(ctx context.Context) error {
	if len(d.opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete Microsoft Graph Service Principals for safety; prefix not specified")
	}

	client := d.client.MicrosoftGraph.ServicePrincipals
	listFilter := odata.Query{
		Filter: fmt.Sprintf("startswith(displayName, '%s')", d.opts.Prefix),
	}
	servicePrincipals, _, err := client.List(ctx, listFilter)
	if err != nil {
		return fmt.Errorf("listing Microsoft Graph Service Principals with prefix %q: %+v", d.opts.Prefix, err)
	}

	for _, servicePrincipal := range *servicePrincipals {
		id := *servicePrincipal.ObjectId
		displayName := *servicePrincipal.DisplayName

		if strings.TrimPrefix(displayName, d.opts.Prefix) != displayName {
			if !d.opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted Microsoft Graph Service Principal %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG] Deleting Microsoft Graph Service Principal %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG] Error during deletion of Microsoft Graph Service Principal %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG] Deleted Microsoft Graph Service Principal %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}

func (d *Dalek) deleteMicrosoftGraphUsers(ctx context.Context) error {
	if len(d.opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete Microsoft Graph Users for safety; prefix not specified")
	}

	client := d.client.MicrosoftGraph.Users
	listFilter := odata.Query{
		Filter: fmt.Sprintf("startswith(displayName, '%s')", d.opts.Prefix),
	}
	users, _, err := client.List(ctx, listFilter)
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list Microsoft Graph Users with prefix: %q", d.opts.Prefix)
	}

	for _, user := range *users {
		id := *user.ObjectId
		displayName := *user.DisplayName

		if strings.TrimPrefix(displayName, d.opts.Prefix) != displayName {
			if !d.opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted Microsoft Graph User %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG] Deleting Microsoft Graph User %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG] Error during deletion of Microsoft Graph User %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG] Deleted Microsoft Graph User %q (ObjID: %s)", displayName, id)
		}
	}

	return nil
}
