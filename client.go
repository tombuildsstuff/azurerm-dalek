package main

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2018-03-01/resources/mgmt/locks"
	"github.com/Azure/azure-sdk-for-go/profiles/2018-03-01/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/hashicorp/go-azure-helpers/authentication"
	`github.com/hashicorp/go-azure-helpers/sender`
)

type AzureClient struct {
	applicationsClient      *graphrbac.ApplicationsClient
	groupsClient            *graphrbac.GroupsClient
	locksClient             *locks.ManagementLocksClient
	resourcesClient         *resources.GroupsClient
	servicePrincipalsClient *graphrbac.ServicePrincipalsClient
	usersClient             *graphrbac.UsersClient
}

func buildAzureClient() (*AzureClient, error) {
	environmentName := os.Getenv("ARM_ENVIRONMENT")
	if environmentName == "" {
		environmentName = azure.PublicCloud.Name
	}

	var environment *azure.Environment
	if strings.Contains(strings.ToLower(environmentName), "stack") {
		// for Azure Stack we have to load the Environment from the URI
		endpoint := os.Getenv("ARM_ENDPOINT")
		env, err := authentication.LoadEnvironmentFromUrl(endpoint)
		if err != nil {
			return nil, fmt.Errorf("Error determining Environment from Endpoint %q: %s", endpoint, err)
		}

		environment = env
	} else {
		env, err := authentication.DetermineEnvironment(environmentName)
		if err != nil {
			return nil, fmt.Errorf("Error determining Environment %q: %s", environmentName, err)
		}

		environment = env
	}

	builder := authentication.Builder{
		TenantID:       os.Getenv("ARM_TENANT_ID"),
		SubscriptionID: os.Getenv("ARM_SUBSCRIPTION_ID"),
		ClientID:       os.Getenv("ARM_CLIENT_ID"),
		ClientSecret:   os.Getenv("ARM_CLIENT_SECRET"),
		Environment:    environmentName,

		// Feature Toggles
		SupportsClientSecretAuth: true,
		SupportsAzureCliToken:    true,
	}

	client, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("Error building ARM Client: %s", err)
	}

	sender := sender.BuildSender("Azure Dalek")

	oauthConfig, err := client.BuildOAuthConfig(environment.ActiveDirectoryEndpoint)
	if err != nil {
		return nil, err
	}

	resourceManagerAuth, err := client.GetAuthorizationToken(sender, oauthConfig, environment.TokenAudience)
	if err != nil {
		return nil, err
	}

	resourcesClient := resources.NewGroupsClientWithBaseURI(environment.ResourceManagerEndpoint, client.SubscriptionID)
	resourcesClient.Authorizer = resourceManagerAuth

	locksClient := locks.NewManagementLocksClientWithBaseURI(environment.ResourceManagerEndpoint, client.SubscriptionID)
	locksClient.Authorizer = resourceManagerAuth

	graphAuth, err := client.GetAuthorizationToken(sender, oauthConfig, environment.GraphEndpoint)
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
		applicationsClient:      &applicationsClient,
		groupsClient:            &groupsClient,
		locksClient:             &locksClient,
		resourcesClient:         &resourcesClient,
		servicePrincipalsClient: &servicePrincipalsClient,
		usersClient:             &usersClient,
	}

	return &azureClient, nil
}
