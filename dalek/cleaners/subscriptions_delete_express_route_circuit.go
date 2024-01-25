package cleaners

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonids"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-05-01/expressroutecircuitauthorizations"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-05-01/expressroutecircuitconnections"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-05-01/expressroutecircuitpeerings"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-05-01/expressroutecircuits"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

type deleteExpressRouteCircuitsSubscriptionCleaner struct{}

var _ SubscriptionCleaner = deleteExpressRouteCircuitsSubscriptionCleaner{}

func (p deleteExpressRouteCircuitsSubscriptionCleaner) Name() string {
	return "Removing Express Route Circuit"
}

func (p deleteExpressRouteCircuitsSubscriptionCleaner) Cleanup(ctx context.Context, subscriptionId commonids.SubscriptionId, client *clients.AzureClient, opts options.Options) error {
	expressRouteCircuitsClient := client.ResourceManager.ExpressRouteCircuitsClient
	expressRouteCircuitAuthorizationsClient := client.ResourceManager.ExpressRouteCircuitAuthorizationsClient
	expressRouteCircuitConnectionsClient := client.ResourceManager.ExpressRouteCircuitConnectionsClient
	expressRouteCircuitPeeringsClient := client.ResourceManager.ExpressRouteCircuitPeeringsClient

	expressRouteCircuits, err := expressRouteCircuitsClient.ListAllComplete(ctx, subscriptionId)
	if err != nil {
		return fmt.Errorf("listing Express Route Circuits for %s: %+v", subscriptionId, err)
	}

	for _, expressRouteCircuit := range expressRouteCircuits.Items {
		if expressRouteCircuit.Id == nil {
			continue
		}

		expressRouteCircuitIdForAuthorizations, err := expressroutecircuitauthorizations.ParseExpressRouteCircuitID(*expressRouteCircuit.Id)
		if err != nil {
			return err
		}

		authorizations, err := expressRouteCircuitAuthorizationsClient.ListComplete(ctx, *expressRouteCircuitIdForAuthorizations)
		if err != nil {
			return fmt.Errorf("listing Express Route Circuit Authorizations for %s: %+v", expressRouteCircuitIdForAuthorizations, err)
		}

		for _, authorization := range authorizations.Items {
			if authorization.Id == nil {
				continue
			}

			authorizationId, err := expressroutecircuitauthorizations.ParseAuthorizationID(*authorization.Id)
			if err != nil {
				return err
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", authorizationId)
				continue
			}

			if err = expressRouteCircuitAuthorizationsClient.DeleteThenPoll(ctx, *authorizationId); err != nil {
				log.Printf("[DEBUG] Unable to delete %s: %+v", authorizationId, err)
			}
		}

		expressRouteCircuitIdForPeerings, err := expressroutecircuitpeerings.ParseExpressRouteCircuitID(*expressRouteCircuit.Id)
		if err != nil {
			return err
		}

		peerings, err := expressRouteCircuitPeeringsClient.ListComplete(ctx, *expressRouteCircuitIdForPeerings)
		if err != nil {
			return fmt.Errorf("listing Express Route Circuit Peerings for %s: %+v", expressRouteCircuitIdForPeerings, err)
		}

		for _, peering := range peerings.Items {
			if peering.Id == nil {
				continue
			}

			peeringId, err := commonids.ParseExpressRouteCircuitPeeringID(*peering.Id)
			if err != nil {
				return err
			}

			connections, err := expressRouteCircuitConnectionsClient.ListComplete(ctx, *peeringId)
			if err != nil {
				return fmt.Errorf("listing express route circuit connections for %s: %+v", peeringId, err)
			}

			for _, connection := range connections.Items {
				if connection.Id == nil {
					continue
				}

				connectionid, err := expressroutecircuitconnections.ParsePeeringConnectionID(*connection.Id)
				if err != nil {
					return err
				}

				if !opts.ActuallyDelete {
					log.Printf("[DEBUG] Would have deleted %s..", connectionid)
					continue
				}

				if err = expressRouteCircuitConnectionsClient.DeleteThenPoll(ctx, *connectionid); err != nil {
					log.Printf("[DEBUG] Unable to delete %s: %+v", connectionid, err)
				}
			}

			if !opts.ActuallyDelete {
				log.Printf("[DEBUG] Would have deleted %s..", peeringId)
				continue
			}

			if err = expressRouteCircuitPeeringsClient.DeleteThenPoll(ctx, *peeringId); err != nil {
				log.Printf("[DEBUG] Unable to delete %s: %+v", peeringId, err)
			}
		}

		expressRouteCircuitId, err := expressroutecircuits.ParseExpressRouteCircuitID(*expressRouteCircuit.Id)
		if err != nil {
			return err
		}

		if !opts.ActuallyDelete {
			log.Printf("[DEBUG] Would have deleted %s..", expressRouteCircuitId)
			continue
		}

		if err = expressRouteCircuitsClient.DeleteThenPoll(ctx, *expressRouteCircuitId); err != nil {
			log.Printf("[DEBUG] Unable to delete %s: %+v", expressRouteCircuitId, err)
		}
	}

	return nil
}
