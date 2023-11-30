package cleaners

import (
	"context"

	"github.com/manicminer/hamilton/msgraph"
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

// MsGraphCleaners : the order of these is intentional, to resolve graph dependencies
var MsGraphCleaners = []MsGraphCleaner{
	accessPackagesCleaner{},
	accessPackageAssignmentPoliciesCleaner{},
	accessPackageAssignmentRequestsCleaner{},
	accessPackageCatalogsCleaner{},
	administrativeUnitsCleaner{},
	claimsMappingPoliciesCleaner{},
	conditionalAccessPoliciesCleaner{},
	connectedOrganizationsCleaner{},
	namedLocationsCleaner{},
	authenticationStrengthPoliciesCleaner{},
	servicePrincipalsCleaner{},
	applicationsCleaner{},
	groupsCleaner{},
	usersCleaner{},
	roleDefinitionsCleaner{},
	termsOfUseAgreementsCleaner{},
}

type MsGraphCleaner interface {
	// Name specifies the name of this SubscriptionCleaner
	Name() string

	// Cleanup performs this clean-up operation against the given Subscription
	Cleanup(ctx context.Context, azure *clients.AzureClient, opts options.Options) error
}

func configureMsGraphClient(baseClient *msgraph.Client, azure *clients.AzureClient) {
	baseClient.Authorizer = azure.MicrosoftGraph.Authorizer
	baseClient.Endpoint = azure.MicrosoftGraph.Endpoint
}
