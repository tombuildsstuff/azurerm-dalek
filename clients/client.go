package clients

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-02-01/managedhsms"
	"github.com/hashicorp/go-azure-sdk/resource-manager/managementgroups/2021-04-01/managementgroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2020-05-01/managementlocks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	servicebusV20220101Preview "github.com/hashicorp/go-azure-sdk/resource-manager/servicebus/2022-01-01-preview"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/auth/autorest"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/manicminer/hamilton/msgraph"
)

type AzureClient struct {
	ActiveDirectory ActiveDirectoryClient
	MicrosoftGraph  MicrosoftGraphClient
	ResourceManager ResourceManagerClient
	SubscriptionID  string
}

type ActiveDirectoryClient struct {
	// TODO: refactor to use Graph

	UsersClient        *graphrbac.UsersClient
	ApplicationsClient *graphrbac.ApplicationsClient
}

type MicrosoftGraphClient struct {
	Groups            *msgraph.GroupsClient
	ServicePrincipals *msgraph.ServicePrincipalsClient
}

type ResourceManagerClient struct {
	LocksClient       *managementlocks.ManagementLocksClient
	ManagementClient  *managementgroups.ManagementGroupsClient
	ManagedHSMsClient *managedhsms.ManagedHsmsClient
	ResourcesClient   *resourcegroups.ResourceGroupsClient
	ServiceBus        *servicebusV20220101Preview.Client
}

type Credentials struct {
	ClientID        string
	ClientSecret    string
	SubscriptionID  string
	TenantID        string
	EnvironmentName string
	Endpoint        string
}

func BuildAzureClient(ctx context.Context, credentials Credentials) (*AzureClient, error) {
	// TODO: refactor to use go-azure-sdk and MSAL
	var environment *environments.Environment
	if strings.Contains(strings.ToLower(credentials.EnvironmentName), "stack") {
		// for Azure Stack we have to load the Environment from the URI
		env, err := environments.FromEndpoint(ctx, credentials.Endpoint, credentials.EnvironmentName)
		if err != nil {
			return nil, fmt.Errorf("loading Environment from Endpoint %q: %s", credentials.Endpoint, err)
		}

		environment = env
	} else {
		env, err := environments.FromName(credentials.EnvironmentName)
		if err != nil {
			return nil, fmt.Errorf("determining Environment %q: %s", credentials.EnvironmentName, err)
		}

		environment = env
	}

	creds := auth.Credentials{
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
		TenantID:     credentials.TenantID,
		Environment:  *environment,

		EnableAuthenticatingUsingClientSecret: true,
	}
	resourceManagerAuthorizer, err := auth.NewAuthorizerFromCredentials(ctx, creds, environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Resource Manager authorizer: %+v", err)
	}
	resourceManagerEndpoint, ok := environment.ResourceManager.Endpoint()
	if !ok {
		return nil, fmt.Errorf("environment %q was missing a Resource Manager endpoint", environment.Name)
	}

	locksClient := managementlocks.NewManagementLocksClientWithBaseURI(*resourceManagerEndpoint)
	locksClient.Client.Authorizer = autorest.AutorestAuthorizer(resourceManagerAuthorizer)

	managementClient, err := managementgroups.NewManagementGroupsClientWithBaseURI(environment.ResourceManager)
	managementClient.Client.Authorizer = autorest.AutorestAuthorizer(resourceManagerAuthorizer)

	managedHsmsClient := managedhsms.NewManagedHsmsClientWithBaseURI(*resourceManagerEndpoint)
	managedHsmsClient.Client.Authorizer = autorest.AutorestAuthorizer(resourceManagerAuthorizer)

	resourcesClient := resourcegroups.NewResourceGroupsClientWithBaseURI(*resourceManagerEndpoint)
	resourcesClient.Client.Authorizer = autorest.AutorestAuthorizer(resourceManagerAuthorizer)

	serviceBusClient, err := servicebusV20220101Preview.NewClientWithBaseURI(environment.ResourceManager, func(c *resourcemanager.Client) {
		c.Authorizer = resourceManagerAuthorizer
	})
	if err != nil {
		return nil, fmt.Errorf("building ServiceBus Client: %+v", err)
	}

	// Microsoft Graph
	microsoftGraphAuthorizer, err := auth.NewAuthorizerFromCredentials(ctx, creds, environment.MicrosoftGraph)
	if err != nil {
		return nil, fmt.Errorf("building Microsoft Graph authorizer: %+v", err)
	}
	microsoftGraphEndpoint, ok := environment.MicrosoftGraph.Endpoint()
	if !ok {
		return nil, fmt.Errorf("environment %q was missing a Microsoft Graph endpoint", environment.Name)
	}

	groupsClient := msgraph.NewGroupsClient()
	groupsClient.BaseClient.Authorizer = microsoftGraphAuthorizer
	groupsClient.BaseClient.Endpoint = *microsoftGraphEndpoint

	servicePrincipalsClient := msgraph.NewServicePrincipalsClient()
	servicePrincipalsClient.BaseClient.Authorizer = microsoftGraphAuthorizer
	servicePrincipalsClient.BaseClient.Endpoint = *microsoftGraphEndpoint

	// Legacy / AzureAD
	var azureAdGraph environments.Api = azureActiveDirectoryGraph{}
	azureActiveDirectoryAuth, err := auth.NewAuthorizerFromCredentials(ctx, creds, azureAdGraph)
	if err != nil {
		return nil, fmt.Errorf("building Azure AD Graph authorizer: %+v", err)
	}
	azureActiveDirectoryEndpoint, ok := azureAdGraph.Endpoint()
	if !ok {
		return nil, fmt.Errorf("environment %q was missing a Azure AD Graph endpoint", environment.Name)
	}

	legacyApplicationsClient := graphrbac.NewApplicationsClientWithBaseURI(*azureActiveDirectoryEndpoint, credentials.TenantID)
	legacyApplicationsClient.Authorizer = autorest.AutorestAuthorizer(azureActiveDirectoryAuth)

	legacyUsersClient := graphrbac.NewUsersClientWithBaseURI(*azureActiveDirectoryEndpoint, credentials.TenantID)
	legacyUsersClient.Authorizer = autorest.AutorestAuthorizer(azureActiveDirectoryAuth)

	azureClient := AzureClient{
		ActiveDirectory: ActiveDirectoryClient{
			ApplicationsClient: &legacyApplicationsClient,
			UsersClient:        &legacyUsersClient,
		},
		MicrosoftGraph: MicrosoftGraphClient{
			Groups:            groupsClient,
			ServicePrincipals: servicePrincipalsClient,
		},
		ResourceManager: ResourceManagerClient{
			LocksClient:       &locksClient,
			ManagementClient:  managementClient,
			ManagedHSMsClient: &managedHsmsClient,
			ResourcesClient:   &resourcesClient,
			ServiceBus:        serviceBusClient,
		},
		SubscriptionID: credentials.SubscriptionID,
	}

	return &azureClient, nil
}
