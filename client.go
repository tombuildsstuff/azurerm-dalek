package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

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
	applicationsClient      *graphrbac.ApplicationsClient
	authClient              *authentication.Config
	groupsClient            *graphrbac.GroupsClient
	locksClient             *managementlocks.ManagementLocksClient
	managementClient        *managementgroups.ManagementGroupsClient
	resourcesClient         *resourcegroups.ResourceGroupsClient
	servicePrincipalsClient *graphrbac.ServicePrincipalsClient
	usersClient             *graphrbac.UsersClient
}

func buildAzureClient() (*AzureClient, error) {
	environmentName := os.Getenv("ARM_ENVIRONMENT")

	var environment *azure.Environment
	endpoint := os.Getenv("ARM_ENDPOINT")
	if strings.Contains(strings.ToLower(environmentName), "stack") {
		// for Azure Stack we have to load the Environment from the URI
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

	api, err := environments.EnvironmentFromString(environmentName)
	if err != nil {
		return nil, fmt.Errorf("unable to find environment %q from endpoint %q: %+v", environmentName, endpoint, err)
	}

	sender := sender.BuildSender("Azure Dalek")

	oauthConfig, err := client.BuildOAuthConfig(environment.ActiveDirectoryEndpoint)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
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
		applicationsClient:      &applicationsClient,
		authClient:              client,
		groupsClient:            &groupsClient,
		locksClient:             &locksClient,
		managementClient:        &managementClient,
		resourcesClient:         &resourcesClient,
		servicePrincipalsClient: &servicePrincipalsClient,
		usersClient:             &usersClient,
	}

	return &azureClient, nil
}
