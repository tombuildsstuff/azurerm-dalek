package clients

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/hashicorp/go-azure-helpers/authentication"
	"github.com/hashicorp/go-azure-helpers/sender"
	"github.com/hashicorp/go-azure-sdk/resource-manager/managementgroups/2021-04-01/managementgroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2020-05-01/managementlocks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	"github.com/manicminer/hamilton/environments"
)

type AzureClient struct {
	ActiveDirectory ActiveDirectoryClient
	ResourceManager ResourceManagerClient

	AuthClient *authentication.Config
}

type ActiveDirectoryClient struct {
	// TODO: refactor to use Graph

	GroupsClient            *graphrbac.GroupsClient
	ServicePrincipalsClient *graphrbac.ServicePrincipalsClient
	UsersClient             *graphrbac.UsersClient
	ApplicationsClient      *graphrbac.ApplicationsClient
}

type ResourceManagerClient struct {
	LocksClient      *managementlocks.ManagementLocksClient
	ManagementClient *managementgroups.ManagementGroupsClient
	ResourcesClient  *resourcegroups.ResourceGroupsClient
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
	var environment *azure.Environment
	if strings.Contains(strings.ToLower(credentials.EnvironmentName), "stack") {
		// for Azure Stack we have to load the Environment from the URI
		env, err := authentication.LoadEnvironmentFromUrl(credentials.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("loading Environment from Endpoint %q: %s", credentials.Endpoint, err)
		}

		environment = env
	} else {
		env, err := authentication.DetermineEnvironment(credentials.EnvironmentName)
		if err != nil {
			return nil, fmt.Errorf("determining Environment %q: %s", credentials.EnvironmentName, err)
		}

		environment = env
	}

	builder := authentication.Builder{
		TenantID:       credentials.TenantID,
		SubscriptionID: credentials.SubscriptionID,
		ClientID:       credentials.ClientID,
		ClientSecret:   credentials.ClientSecret,
		Environment:    credentials.EnvironmentName,

		// Feature Toggles
		SupportsClientSecretAuth: true,
		SupportsAzureCliToken:    true,
	}

	client, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("Error building ARM Client: %s", err)
	}

	api, err := environments.EnvironmentFromString(credentials.EnvironmentName)
	if err != nil {
		return nil, fmt.Errorf("unable to find environment %q from endpoint %q: %+v", credentials.EnvironmentName, credentials.Endpoint, err)
	}

	sender := sender.BuildSender("Azure Dalek")

	oauthConfig, err := client.BuildOAuthConfig(environment.ActiveDirectoryEndpoint)
	if err != nil {
		return nil, err
	}

	resourceManagerAuth, err := client.GetMSALToken(ctx, api.ResourceManager, sender, oauthConfig, environment.TokenAudience)
	if err != nil {
		return nil, err
	}

	resourcesClient := resourcegroups.NewResourceGroupsClientWithBaseURI(environment.ResourceManagerEndpoint)
	resourcesClient.Client.Authorizer = resourceManagerAuth

	locksClient := managementlocks.NewManagementLocksClientWithBaseURI(environment.ResourceManagerEndpoint)
	locksClient.Client.Authorizer = resourceManagerAuth

	managementClient := managementgroups.NewManagementGroupsClientWithBaseURI(environment.ResourceManagerEndpoint)
	managementClient.Client.Authorizer = resourceManagerAuth

	graphAuth, err := client.GetMSALToken(ctx, api.MsGraph, sender, oauthConfig, environment.TokenAudience)
	if err != nil {
		return nil, err
	}

	applicationsClient := graphrbac.NewApplicationsClientWithBaseURI(environment.GraphEndpoint, client.TenantID)
	applicationsClient.Authorizer = graphAuth

	groupsClient := graphrbac.NewGroupsClientWithBaseURI(environment.GraphEndpoint, client.TenantID)
	groupsClient.Authorizer = graphAuth

	servicePrincipalsClient := graphrbac.NewServicePrincipalsClientWithBaseURI(environment.GraphEndpoint, client.TenantID)
	servicePrincipalsClient.Authorizer = graphAuth

	usersClient := graphrbac.NewUsersClientWithBaseURI(environment.GraphEndpoint, client.TenantID)
	usersClient.Authorizer = graphAuth

	azureClient := AzureClient{
		ActiveDirectory: ActiveDirectoryClient{
			ApplicationsClient:      &applicationsClient,
			GroupsClient:            &groupsClient,
			ServicePrincipalsClient: &servicePrincipalsClient,
			UsersClient:             &usersClient,
		},
		ResourceManager: ResourceManagerClient{
			LocksClient:      &locksClient,
			ManagementClient: &managementClient,
			ResourcesClient:  &resourcesClient,
		},

		AuthClient: client,
	}

	return &azureClient, nil
}
