package dalek

import (
	"context"
	"fmt"
	"log"
	"strings"
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
	if err := d.deleteAADGroups(ctx); err != nil {
		return fmt.Errorf("deleting Groups: %+v", err)
	}

	log.Printf("[DEBUG] Preparing to delete Users")
	if err := d.deleteAADUsers(ctx); err != nil {
		return fmt.Errorf("deleting Users: %+v", err)
	}

	log.Printf("[DEBUG] Preparing to delete Management Groups")
	if err := d.deleteManagementGroups(ctx); err != nil {
		return fmt.Errorf("deleting Management Groups: %+v", err)
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

func (d *Dalek) deleteAADGroups(ctx context.Context) error {
	if len(d.opts.Prefix) == 0 {
		return fmt.Errorf("[ERROR] Not proceeding to delete AAD Groups for safety; prefix not specified")
	}

	client := d.client.ActiveDirectory.GroupsClient
	groups, err := client.List(ctx, fmt.Sprintf("startswith(displayName, '%s')", d.opts.Prefix))
	if err != nil {
		return fmt.Errorf("[ERROR] Unable to list AAD Groups with prefix: %q", d.opts.Prefix)
	}

	for _, group := range groups.Values() {
		id := *group.ObjectID
		displayName := *group.DisplayName

		if strings.TrimPrefix(displayName, d.opts.Prefix) != displayName {
			if !d.opts.ActuallyDelete {
				log.Printf("[DEBUG]   Would have deleted AAD Group %q (ObjID: %s)", displayName, id)
				continue
			}

			log.Printf("[DEBUG]   Deleting AAD Group %q (ObjectId: %s)...", displayName, id)
			if _, err := client.Delete(ctx, id); err != nil {
				log.Printf("[DEBUG]   Error during deletion of AAD Group %q (ObjID: %s): %s", displayName, id, err)
				continue
			}
			log.Printf("[DEBUG]   Deleted AAD Group %q (ObjID: %s)", displayName, id)
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
