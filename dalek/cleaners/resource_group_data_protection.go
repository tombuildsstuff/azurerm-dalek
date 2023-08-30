package cleaners

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/dataprotection/2023-05-01/backupinstances"
	"github.com/hashicorp/go-azure-sdk/resource-manager/dataprotection/2023-05-01/backuppolicies"
	"github.com/hashicorp/go-azure-sdk/resource-manager/dataprotection/2023-05-01/backupvaults"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

var _ ResourceGroupCleaner = removeDataProtectionFromResourceGroupCleaner{}

type removeDataProtectionFromResourceGroupCleaner struct {
}

func (removeDataProtectionFromResourceGroupCleaner) Name() string {
	return "Removing Data Protection"
}

func (removeDataProtectionFromResourceGroupCleaner) Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient, opts options.Options) error {
	backupVaults, err := client.ResourceManager.DataProtection.BackupVaults.GetInResourceGroupComplete(ctx, id)
	if err != nil {
		return fmt.Errorf("retrieving the Backup Vaults within %s: %+v", id, err)
	}
	for _, vault := range backupVaults.Items {
		vaultId := backupvaults.NewBackupVaultID(id.SubscriptionId, id.ResourceGroupName, *vault.Name)

		// list the Backup Instances within it, those need to be removed first
		backupInstancesVaultId := backupinstances.NewBackupVaultID(vaultId.SubscriptionId, vaultId.ResourceGroupName, vaultId.BackupVaultName)
		instances, err := client.ResourceManager.DataProtection.BackupInstances.ListComplete(ctx, backupInstancesVaultId)
		if err != nil {
			return fmt.Errorf("listing Backup Instances within %s: %+v", backupInstancesVaultId, err)
		}
		for _, instance := range instances.Items {
			instanceId := backupinstances.NewBackupInstanceID(backupInstancesVaultId.SubscriptionId, backupInstancesVaultId.ResourceGroupName, backupInstancesVaultId.BackupVaultName, *instance.Name)
			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", instanceId)
				continue
			}

			log.Printf("[DEBUG] Deleting %s..", instanceId)
			if err := client.ResourceManager.DataProtection.BackupInstances.DeleteThenPoll(ctx, instanceId); err != nil {
				return fmt.Errorf("deleting %s: %+v", instanceId, err)
			}
			log.Printf("[DEBUG] Deleted %s.", instanceId)
		}

		// then let's go through and remove the Backup Policies
		backupPoliciesVaultId := backuppolicies.NewBackupVaultID(vaultId.SubscriptionId, vaultId.ResourceGroupName, vaultId.BackupVaultName)
		policies, err := client.ResourceManager.DataProtection.BackupPolicies.ListComplete(ctx, backupPoliciesVaultId)
		if err != nil {
			return fmt.Errorf("listing Backup Policies within %s: %+v", backupPoliciesVaultId, err)
		}
		for _, policy := range policies.Items {
			policyId := backuppolicies.NewBackupPolicyID(backupPoliciesVaultId.SubscriptionId, backupPoliciesVaultId.ResourceGroupName, backupPoliciesVaultId.BackupVaultName, *policy.Name)
			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", policyId)
				continue
			}

			log.Printf("[DEBUG] Deleting %s..", policyId)
			if _, err := client.ResourceManager.DataProtection.BackupPolicies.Delete(ctx, policyId); err != nil {
				return fmt.Errorf("deleting %s: %+v", policyId, err)
			}
			log.Printf("[DEBUG] Deleted %s.", policyId)
		}

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted %s..", vaultId)
			continue
		}
		log.Printf("[DEBUG] Deleting %s..", vaultId)
		if err := client.ResourceManager.DataProtection.BackupVaults.DeleteThenPoll(ctx, vaultId); err != nil {
			return fmt.Errorf("deleting %s: %+v", vaultId, err)
		}
		log.Printf("[DEBUG] Deleted %s.", vaultId)
	}

	return nil
}
