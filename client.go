package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2018-03-01/resources/mgmt/locks"
	"github.com/Azure/azure-sdk-for-go/profiles/2018-03-01/resources/mgmt/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/hashicorp/go-azure-helpers/authentication"
	`github.com/hashicorp/go-azure-helpers/sender`
)

type ArmClient struct {
	resourcesClient *resources.GroupsClient
	locksClient     *locks.ManagementLocksClient
}

func buildArmClient() (*ArmClient, error) {
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

	sender := sender.BuildSender("AzureRM Dalek")

	oauthConfig, err := client.BuildOAuthConfig(environment.ActiveDirectoryEndpoint)
	if err != nil {
		return nil, err
	}

	endpoint := environment.ResourceManagerEndpoint
	auth, err := client.GetAuthorizationToken(sender, oauthConfig, environment.TokenAudience)
	if err != nil {
		return nil, err
	}

	resourcesClient := resources.NewGroupsClientWithBaseURI(endpoint, client.SubscriptionID)
	resourcesClient.Authorizer = auth

	locksClient := locks.NewManagementLocksClientWithBaseURI(endpoint, client.SubscriptionID)
	locksClient.Authorizer = auth

	armClient := ArmClient{
		resourcesClient: &resourcesClient,
		locksClient:     &locksClient,
	}

	return &armClient, nil
}
