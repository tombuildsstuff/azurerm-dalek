package cleaners

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/certificateobjectlocalrulestack"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/fqdnlistlocalrulestack"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/localrules"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/localrulestacks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/prefixlistlocalrulestack"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
)

type PaloAltoLocalRulestackCleaner struct{}

var _ ResourceGroupCleaner = PaloAltoLocalRulestackCleaner{}

func (p PaloAltoLocalRulestackCleaner) Name() string {
	return "Removing Rulestack Rules"
}

func (p PaloAltoLocalRulestackCleaner) Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient) error {
	rulestacksClient := client.ResourceManager.PaloAltoLocalRulestacksClient

	rulestacks, err := rulestacksClient.ListByResourceGroupComplete(ctx, id)
	if err != nil {
		return err
	}

	// Rules
	rulesClient := client.ResourceManager.PaloAltoLocalRulestackRuleClient
	for _, rg := range rulestacks.Items {
		rulestackId := localrules.NewLocalRulestackID(id.SubscriptionId, id.ResourceGroupName, pointer.From(rg.Name))
		rulesInRulestack, err := rulesClient.ListByLocalRulestacksComplete(ctx, rulestackId)
		if err != nil {
			return fmt.Errorf("listing rules for %s: %+v", id, err)
		}
		for _, v := range rulesInRulestack.Items {
			if ruleId, err := localrules.ParseLocalRuleID(pointer.From(v.Id)); err != nil {
				if _, err := rulesClient.Delete(ctx, *ruleId); err != nil {
					return fmt.Errorf("deleting rule %s from rulestack %s: %+v", ruleId, id, err)
				}
			}
		}
	}

	// FQDN Lists
	fqdnClient := client.ResourceManager.PaloAltoLocalRulestackFQDNClient
	for _, rg := range rulestacks.Items {
		rulestackId := fqdnlistlocalrulestack.NewLocalRulestackID(id.SubscriptionId, id.ResourceGroupName, pointer.From(rg.Name))
		fqdnInRulestack, err := fqdnClient.ListByLocalRulestacksComplete(ctx, rulestackId)
		if err != nil {
			return fmt.Errorf("listing FQDNs for %s: %+v", id, err)
		}
		for _, v := range fqdnInRulestack.Items {
			if fqdnId, err := localrules.ParseLocalRuleID(pointer.From(v.Id)); err != nil {
				if _, err := rulesClient.Delete(ctx, *fqdnId); err != nil {
					return fmt.Errorf("deleting fqdn %s from rulestack %s: %+v", fqdnId, id, err)
				}
			}
		}
	}

	// Certificates
	certClient := client.ResourceManager.PaloAltoLocalRulestackCertificatesClient
	for _, rg := range rulestacks.Items {
		// Remove inspection config - blocks removal of certs if referenced
		rulestackId := certificateobjectlocalrulestack.NewLocalRulestackID(id.SubscriptionId, id.ResourceGroupName, pointer.From(rg.Name))
		rs, err := rulestacksClient.Get(ctx, localrulestacks.LocalRulestackId(rulestackId))
		if err != nil {
			return err
		}
		sec := pointer.From(rs.Model.Properties.SecurityServices)
		if pointer.From(sec.OutboundTrustCertificate) != "" || pointer.From(sec.OutboundUnTrustCertificate) != "" {
			sec.OutboundTrustCertificate = nil
			sec.OutboundUnTrustCertificate = nil
			rs.Model.Properties.SecurityServices = pointer.To(sec)
			if err = rulestacksClient.CreateOrUpdateThenPoll(ctx, localrulestacks.LocalRulestackId(rulestackId), *rs.Model); err != nil {
				return fmt.Errorf("removing certificate usage on %s: %+v", rulestackId, err)
			}
		}
		// Remove certs
		fqdnInRulestack, err := certClient.ListByLocalRulestacksComplete(ctx, rulestackId)
		if err != nil {
			return fmt.Errorf("listing FQDNs for %s: %+v", id, err)
		}
		for _, v := range fqdnInRulestack.Items {
			if fqdnId, err := localrules.ParseLocalRuleID(pointer.From(v.Id)); err != nil {
				if _, err := rulesClient.Delete(ctx, *fqdnId); err != nil {
					return fmt.Errorf("deleting fqdn %s from rulestack %s: %+v", fqdnId, id, err)
				}
			}
		}
	}

	// Prefixes
	prefixClient := client.ResourceManager.PaloAltoLocalRulestackPrefixClient
	for _, rg := range rulestacks.Items {
		rulestackId := prefixlistlocalrulestack.NewLocalRulestackID(id.SubscriptionId, id.ResourceGroupName, pointer.From(rg.Name))
		prefixInRulestack, err := prefixClient.ListByLocalRulestacksComplete(ctx, rulestackId)
		if err != nil {
			return fmt.Errorf("listing FQDNs for %s: %+v", id, err)
		}
		for _, v := range prefixInRulestack.Items {
			if prefixId, err := localrules.ParseLocalRuleID(pointer.From(v.Id)); err != nil {
				if _, err := rulesClient.Delete(ctx, *prefixId); err != nil {
					return fmt.Errorf("deleting prefix %s from rulestack %s: %+v", prefixId, id, err)
				}
			}
		}
	}

	return nil
}
