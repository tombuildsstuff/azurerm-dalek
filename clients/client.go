package clients

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/hashicorp/go-azure-sdk/resource-manager/managementgroups/2021-04-01/managementgroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2020-05-01/managementlocks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/auth/autorest"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
)

type AzureClient struct {
	ActiveDirectory ActiveDirectoryClient
	ResourceManager ResourceManagerClient
	SubscriptionID  string
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

		EnableAuthenticatingUsingAzureCLI:     true,
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

	resourcesClient := resourcegroups.NewResourceGroupsClientWithBaseURI(*resourceManagerEndpoint)
	resourcesClient.Client.Authorizer = autorest.AutorestAuthorizer(resourceManagerAuthorizer)

	locksClient := managementlocks.NewManagementLocksClientWithBaseURI(*resourceManagerEndpoint)
	locksClient.Client.Authorizer = autorest.AutorestAuthorizer(resourceManagerAuthorizer)

	managementClient, err := managementgroups.NewManagementGroupsClientWithBaseURI(environment.ResourceManager)
	managementClient.Client.Authorizer = autorest.AutorestAuthorizer(resourceManagerAuthorizer)

	var azureAdGraph environments.Api = azureActiveDirectoryGraph{}
	azureActiveDirectoryAuth, err := auth.NewAuthorizerFromCredentials(ctx, creds, azureAdGraph)
	if err != nil {
		return nil, fmt.Errorf("building Azure AD Graph authorizer: %+v", err)
	}
	azureActiveDirectoryEndpoint, ok := azureAdGraph.Endpoint()
	if !ok {
		return nil, fmt.Errorf("environment %q was missing a Azure AD Graph endpoint", environment.Name)
	}

	applicationsClient := graphrbac.NewApplicationsClientWithBaseURI(*azureActiveDirectoryEndpoint, credentials.TenantID)
	applicationsClient.Authorizer = autorest.AutorestAuthorizer(azureActiveDirectoryAuth)

	groupsClient := graphrbac.NewGroupsClientWithBaseURI(*azureActiveDirectoryEndpoint, credentials.TenantID)
	groupsClient.Authorizer = autorest.AutorestAuthorizer(azureActiveDirectoryAuth)

	servicePrincipalsClient := graphrbac.NewServicePrincipalsClientWithBaseURI(*azureActiveDirectoryEndpoint, credentials.TenantID)
	servicePrincipalsClient.Authorizer = autorest.AutorestAuthorizer(azureActiveDirectoryAuth)

	usersClient := graphrbac.NewUsersClientWithBaseURI(*azureActiveDirectoryEndpoint, credentials.TenantID)
	usersClient.Authorizer = autorest.AutorestAuthorizer(azureActiveDirectoryAuth)

	azureClient := AzureClient{
		ActiveDirectory: ActiveDirectoryClient{
			ApplicationsClient:      &applicationsClient,
			GroupsClient:            &groupsClient,
			ServicePrincipalsClient: &servicePrincipalsClient,
			UsersClient:             &usersClient,
		},
		ResourceManager: ResourceManagerClient{
			LocksClient:      &locksClient,
			ManagementClient: managementClient,
			ResourcesClient:  &resourcesClient,
		},
		SubscriptionID: credentials.SubscriptionID,
	}

	return &azureClient, nil
}
