package cleaners

import (
	"context"
	"fmt"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"log"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/certificateobjectlocalrulestack"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/fqdnlistlocalrulestack"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/localrules"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/localrulestacks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29/prefixlistlocalrulestack"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

type paloAltoLocalRulestackCleaner struct{}

var _ ResourceGroupCleaner = paloAltoLocalRulestackCleaner{}

func (p paloAltoLocalRulestackCleaner) Name() string {
	return "Removing Rulestack Rules"
}

func (p paloAltoLocalRulestackCleaner) Cleanup(ctx context.Context, id commonids.ResourceGroupId, client *clients.AzureClient, opts options.Options) error {
	rulestacksClient := client.ResourceManager.PaloAlto.LocalRulestacks

	rulestacks, err := rulestacksClient.ListByResourceGroupComplete(ctx, id)
	if err != nil {
		return err
	}

	// Rules
	rulesClient := client.ResourceManager.PaloAlto.LocalRules
	for _, rg := range rulestacks.Items {
		rulestackId := localrules.NewLocalRulestackID(id.SubscriptionId, id.ResourceGroupName, pointer.From(rg.Name))
		rulesInRulestack, err := rulesClient.ListByLocalRulestacks(ctx, rulestackId)
		if err != nil {
			if response.WasStatusCode(rulesInRulestack.HttpResponse, 500) || response.WasNotFound(rulesInRulestack.HttpResponse) || response.WasStatusCode(rulesInRulestack.HttpResponse, 502) {
				continue
			}
			return fmt.Errorf("listing rules for %s: %+v", id, err)
		}
		if model := rulesInRulestack.Model; model != nil {
			for _, v := range *model {
				ruleId, err := localrules.ParseLocalRuleIDInsensitively(pointer.From(v.Id))
				if err != nil {
					return fmt.Errorf("parsing rule %s: %+v", pointer.From(v.Id), err)
				}

				if !opts.ActuallyDelete {
					log.Printf("[DEBUG] Would have deleted the Local Rule for %s..", *ruleId)
					continue
				}

				log.Printf("[DEBUG] Deleting %s..", *ruleId)
				if _, err := rulesClient.Delete(ctx, *ruleId); err != nil {
					return fmt.Errorf("deleting rule %s from rulestack %s: %+v", ruleId, id, err)
				}
				log.Printf("[DEBUG] Deleting %s..", *ruleId)
			}
		}
	}

	// FQDN Lists
	fqdnClient := client.ResourceManager.PaloAlto.FqdnListLocalRulestack
	for _, rg := range rulestacks.Items {
		rulestackId := fqdnlistlocalrulestack.NewLocalRulestackID(id.SubscriptionId, id.ResourceGroupName, pointer.From(rg.Name))
		fqdnInRulestack, err := fqdnClient.ListByLocalRulestacks(ctx, rulestackId)
		if err != nil {
			if response.WasStatusCode(fqdnInRulestack.HttpResponse, 500) || response.WasStatusCode(fqdnInRulestack.HttpResponse, 502) || response.WasNotFound(fqdnInRulestack.HttpResponse) {
				continue
			}
			return fmt.Errorf("listing FQDNs for %s: %+v", id, err)
		}
		if model := fqdnInRulestack.Model; model != nil {
			for _, v := range *model {
				fqdnId, err := localrules.ParseLocalRuleIDInsensitively(pointer.From(v.Id))
				if err != nil {
					return fmt.Errorf("parsing %q as a local rule id: %+v", pointer.From(v.Id), err)
				}

				if !opts.ActuallyDelete {
					log.Printf("[DEBUG] Would have deleted the Local Rule for %s..", *fqdnId)
					continue
				}

				log.Printf("[DEBUG] Deleting %s..", *fqdnId)
				if _, err := rulesClient.Delete(ctx, *fqdnId); err != nil {
					return fmt.Errorf("deleting fqdn %s from rulestack %s: %+v", fqdnId, id, err)
				}
				log.Printf("[DEBUG] Deleted %s..", *fqdnId)
			}
		}
	}

	// Certificates
	certClient := client.ResourceManager.PaloAlto.CertificateObjectLocalRulestack
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
			localRulestackId := localrulestacks.NewLocalRulestackID(rulestackId.SubscriptionId, rulestackId.ResourceGroupName, rulestackId.LocalRulestackName)
			if err = rulestacksClient.CreateOrUpdateThenPoll(ctx, localRulestackId, *rs.Model); err != nil {
				return fmt.Errorf("removing certificate usage on %s: %+v", rulestackId, err)
			}
		}
		// Remove certs
		fqdnInRulestack, err := certClient.ListByLocalRulestacks(ctx, rulestackId)
		if err != nil {
			if response.WasStatusCode(fqdnInRulestack.HttpResponse, 500) || response.WasStatusCode(fqdnInRulestack.HttpResponse, 502) || response.WasNotFound(fqdnInRulestack.HttpResponse) {
				continue
			}
			return fmt.Errorf("listing FQDNs for %s: %+v", id, err)
		}
		if model := fqdnInRulestack.Model; model != nil {
			for _, v := range *model {
				if fqdnId, err := localrules.ParseLocalRuleIDInsensitively(pointer.From(v.Id)); err != nil {
					if _, err := rulesClient.Delete(ctx, *fqdnId); err != nil {
						return fmt.Errorf("deleting fqdn %s from rulestack %s: %+v", fqdnId, id, err)
					}
				}
			}
		}
	}

	// Prefixes
	prefixClient := client.ResourceManager.PaloAlto.PrefixListLocalRulestack
	for _, rg := range rulestacks.Items {
		rulestackId := prefixlistlocalrulestack.NewLocalRulestackID(id.SubscriptionId, id.ResourceGroupName, pointer.From(rg.Name))
		prefixInRulestack, err := prefixClient.ListByLocalRulestacks(ctx, rulestackId)
		if err != nil {
			if response.WasStatusCode(prefixInRulestack.HttpResponse, 500) || response.WasStatusCode(prefixInRulestack.HttpResponse, 502) || response.WasNotFound(prefixInRulestack.HttpResponse) {
				continue
			}
			return fmt.Errorf("listing FQDNs for %s: %+v", id, err)
		}
		if model := prefixInRulestack.Model; model != nil {
			for _, v := range *model {
				if prefixId, err := localrules.ParseLocalRuleIDInsensitively(pointer.From(v.Id)); err != nil {
					if _, err := rulesClient.Delete(ctx, *prefixId); err != nil {
						return fmt.Errorf("deleting prefix %s from rulestack %s: %+v", prefixId, id, err)
					}
				}
			}
		}
	}

	return nil
}
