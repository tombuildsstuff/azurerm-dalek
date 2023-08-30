package cleaners

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/recoveryservices/2023-04-01/vaults"
	"github.com/hashicorp/go-azure-sdk/resource-manager/recoveryservicesbackup/2023-04-01/backupprotectioncontainers"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resourcegraph/2022-10-01/resources"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

type recoveryServicesResourceGroupCleaner struct{}

var _ ResourceGroupCleaner = recoveryServicesResourceGroupCleaner{}

func (c recoveryServicesResourceGroupCleaner) Name() string {
	return "Recovery Services"
}

func (c recoveryServicesResourceGroupCleaner) Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient, opts options.Options) error {
	log.Printf("[DEBUG] Retrieving Recovery Services Vaults in %s..", id)
	vaultIds, err := c.findRecoveryServiceVaultIDs(ctx, id, client)
	if err != nil {
		return fmt.Errorf("finding the Recovery Services Vaults IDs within %s: %+v", id, err)
	}

	for _, vaultId := range *vaultIds {
		// we then need to find any Backup Protected Items within this Recovery Services Vault
		// list all of the items within the protection container

		backupContainerVaultId := backupprotectioncontainers.NewVaultID(vaultId.SubscriptionId, vaultId.ResourceGroupName, vaultId.VaultName)
		listOpts := backupprotectioncontainers.DefaultListOperationOptions()
		results, err := client.ResourceManager.RecoveryServicesBackupProtectionContainersClient.ListComplete(ctx, backupContainerVaultId, listOpts)
		if err != nil {
			return fmt.Errorf("listing Backup Protection Containers within %s: %+v", backupContainerVaultId, err)
		}

		for _, item := range results.Items {
			log.Printf("TOMTOMTOM: %q", *item.Id)
		}

		// protectedItemName := fmt.Sprintf("VM;iaasvmcontainerv2;%s;%s", parsedVmId.ResourceGroup, parsedVmId.Name)
		//	containerName := fmt.Sprintf("iaasvmcontainer;iaasvmcontainerv2;%s;%s", parsedVmId.ResourceGroup, parsedVmId.Name)
		// /subscriptions/1a6092a6-137e-4025-9a7c-ef77f76f2c02/resourceGroups/acctestRG-backup-230222073824999320/providers/Microsoft.RecoveryServices/vaults/acctest-230222073824999320/backupFabrics/Azure/protectionContainers/IaasVMContainer;iaasvmcontainerv2;acctestRG-backup-230222073824999320;acctestvm/protectedItems/VM;iaasvmcontainerv2;acctestRG-backup-230222073824999320;acctestvm

		// we then need to find any Replicated Items within this Recovery Services Vault

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted %s..", vaultId)
			continue
		}

		log.Printf("[DEBUG] Deleting %s..", vaultId)

		//if err := client.ResourceManager.NotificationHubNamespaceClient.DeleteThenPoll(ctx, namespaceId); err != nil {
		//	return fmt.Errorf("deleting %s: %+v", namespaceId, err)
		//}
		log.Printf("[DEBUG] Deleted %s.", vaultId)
	}

	return nil
}

func (c recoveryServicesResourceGroupCleaner) ResourceTypes() []string {
	return []string{
		"Microsoft.RecoveryServices/vaults",
	}
}

func (c recoveryServicesResourceGroupCleaner) findRecoveryServiceVaultIDs(ctx context.Context, resourceGroupId commonids.ResourceGroupId, client *clients.AzureClient) (*[]vaults.VaultId, error) {
	query := strings.TrimSpace(fmt.Sprintf(`
resources
| where type =~ "Microsoft.RecoveryServices/vaults"
| where resourceGroup =~ '%s'
| project id
| sort by (tolower(tostring(id))) asc
`, resourceGroupId.ResourceGroupName))
	payload := resources.QueryRequest{
		Options: &resources.QueryRequestOptions{
			Top: pointer.To(int64(1000)),
		},
		Query: query,
		Subscriptions: &[]string{
			resourceGroupId.SubscriptionId,
		},
	}
	resp, err := client.ResourceManager.ResourceGraphClient.Resources(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("performing graph query %q: %+v", query, err)
	}

	if resp.Model == nil {
		return nil, fmt.Errorf("performing graph query %q: response was nil", query)
	}
	if resp.Model.Data == nil {
		return nil, fmt.Errorf("performing graph query %q: response.data was nil", query)
	}

	vaultIds := make([]vaults.VaultId, 0)
	itemsRaw, ok := resp.Model.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected the data to be an []interface but got %+v", resp.Model.Data)
	}
	for index, itemRaw := range itemsRaw {
		item, ok := itemRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected index %d to be a map[string]interface{} but it wasn't", index)
		}
		id, ok := item["id"]
		if !ok {
			return nil, fmt.Errorf("expected an id field for item %d but didn't get one", index)
		}
		idRaw := id.(string)
		namespaceId, err := vaults.ParseVaultIDInsensitively(idRaw)
		if err != nil {
			return nil, fmt.Errorf("parsing %q for index %d: %+v", idRaw, index, err)
		}
		vaultIds = append(vaultIds, *namespaceId)
	}

	return &vaultIds, nil
}
