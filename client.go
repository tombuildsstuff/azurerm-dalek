package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2018-03-01/resources/mgmt/locks"
	"github.com/Azure/azure-sdk-for-go/profiles/2018-03-01/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/hashicorp/go-azure-helpers/authentication"
	"github.com/hashicorp/go-azure-helpers/sender"
	"github.com/manicminer/hamilton/auth"
	"github.com/manicminer/hamilton/environments"
	"github.com/manicminer/hamilton/msgraph"
)

type aadGraphClients struct {
	applicationsClient      *graphrbac.ApplicationsClient
	groupsClient            *graphrbac.GroupsClient
	servicePrincipalsClient *graphrbac.ServicePrincipalsClient
	usersClient             *graphrbac.UsersClient
}

type msGraphClients struct {
	applicationsClient      *msgraph.ApplicationsClient
	groupsClient            *msgraph.GroupsClient
	servicePrincipalsClient *msgraph.ServicePrincipalsClient
	usersClient             *msgraph.UsersClient
}

type AzureClient struct {
	aadGraph        *aadGraphClients
	msGraph         *msGraphClients
	locksClient     *locks.ManagementLocksClient
	resourcesClient *resources.GroupsClient
}

func buildAzureClient(ctx context.Context) (*AzureClient, error) {
	environmentName := os.Getenv("ARM_ENVIRONMENT")
	if environmentName == "" {
		environmentName = "public"
	}

	var environment environments.Environment
	if strings.Contains(strings.ToLower(environmentName), "stack") {
		// for Azure Stack we have to load the Environment from the URI
		endpoint := os.Getenv("ARM_ENDPOINT")
		env, err := environments.EnvironmentFromMetadata(endpoint)
		if err != nil {
			return nil, fmt.Errorf("determining Environment from endpoint %q: %s", endpoint, err)
		}
		if env == nil {
			return nil, fmt.Errorf("returned Environment from Endpoint %q was nil", endpoint)
		}

		environment = *env
	} else {
		env, err := environments.EnvironmentFromString(environmentName)
		if err != nil {
			return nil, fmt.Errorf("determining Environment %q: %s", environmentName, err)
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

	config, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("building config: %s", err)
	}

	sender := sender.BuildSender("Azure Dalek")

	oauthConfig, err := config.BuildOAuthConfig(string(environment.AzureADEndpoint))
	if err != nil {
		return nil, err
	}

	resourceManagerAuth, err := config.GetMSALToken(ctx, environment.ResourceManager, sender, oauthConfig, string(environment.ResourceManager.Endpoint))
	if err != nil {
		return nil, err
	}

	resourcesClient := resources.NewGroupsClientWithBaseURI(string(environment.ResourceManager.Endpoint), config.SubscriptionID)
	resourcesClient.Authorizer = resourceManagerAuth

	locksClient := locks.NewManagementLocksClientWithBaseURI(string(environment.ResourceManager.Endpoint), config.SubscriptionID)
	locksClient.Authorizer = resourceManagerAuth

	aadGraphAuth, err := config.GetMSALToken(ctx, environment.AadGraph, sender, oauthConfig, string(environment.AadGraph.Endpoint))
	if err != nil {
		return nil, err
	}

	aadGraphApplicationsClient := graphrbac.NewApplicationsClientWithBaseURI(string(environment.AadGraph.Endpoint), config.TenantID)
	aadGraphApplicationsClient.Authorizer = aadGraphAuth

	aadGraphGroupsClient := graphrbac.NewGroupsClientWithBaseURI(string(environment.AadGraph.Endpoint), config.TenantID)
	aadGraphGroupsClient.Authorizer = aadGraphAuth

	aadGraphServicePrincipalsClient := graphrbac.NewServicePrincipalsClientWithBaseURI(string(environment.AadGraph.Endpoint), config.TenantID)
	aadGraphServicePrincipalsClient.Authorizer = aadGraphAuth

	aadGraphUsersClient := graphrbac.NewUsersClientWithBaseURI(string(environment.AadGraph.Endpoint), config.TenantID)
	aadGraphUsersClient.Authorizer = aadGraphAuth

	azureClient := AzureClient{
		aadGraph: &aadGraphClients{
			applicationsClient:      &aadGraphApplicationsClient,
			groupsClient:            &aadGraphGroupsClient,
			servicePrincipalsClient: &aadGraphServicePrincipalsClient,
			usersClient:             &aadGraphUsersClient,
		},
		locksClient:     &locksClient,
		resourcesClient: &resourcesClient,
	}

	if environment.MsGraph.IsAvailable() {
		msGraphAuth, err := config.GetMSALToken(ctx, environment.MsGraph, sender, oauthConfig, string(environment.MsGraph.Endpoint))
		if err != nil {
			return nil, err
		}

		authorizer, err := auth.NewAutorestAuthorizerWrapper(msGraphAuth)
		if err != nil {
			return nil, err
		}

		azureClient.msGraph = &msGraphClients{
			applicationsClient:      msgraph.NewApplicationsClient(config.TenantID),
			groupsClient:            msgraph.NewGroupsClient(config.TenantID),
			servicePrincipalsClient: msgraph.NewServicePrincipalsClient(config.TenantID),
			usersClient:             msgraph.NewUsersClient(config.TenantID),
		}

		azureClient.msGraph.applicationsClient.BaseClient.Authorizer = authorizer
		azureClient.msGraph.groupsClient.BaseClient.Authorizer = authorizer
		azureClient.msGraph.servicePrincipalsClient.BaseClient.Authorizer = authorizer
		azureClient.msGraph.usersClient.BaseClient.Authorizer = authorizer
	}

	return &azureClient, nil
}
