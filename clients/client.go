package clients

import (
	"context"
	"fmt"
	"strings"

	dataProtection "github.com/hashicorp/go-azure-sdk/resource-manager/dataprotection/2023-05-01"
	"github.com/hashicorp/go-azure-sdk/resource-manager/keyvault/2023-02-01/managedhsms"
	"github.com/hashicorp/go-azure-sdk/resource-manager/machinelearningservices/2023-04-01-preview/workspaces"
	"github.com/hashicorp/go-azure-sdk/resource-manager/managementgroups/2021-04-01/managementgroups"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2022-05-01/capacitypools"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2022-05-01/netappaccounts"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2022-05-01/volumes"
	"github.com/hashicorp/go-azure-sdk/resource-manager/netapp/2022-05-01/volumesreplication"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-05-01/expressroutecircuitauthorizations"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-05-01/expressroutecircuitconnections"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-05-01/expressroutecircuitpeerings"
	"github.com/hashicorp/go-azure-sdk/resource-manager/network/2023-05-01/expressroutecircuits"
	"github.com/hashicorp/go-azure-sdk/resource-manager/notificationhubs/2017-04-01/namespaces"
	paloAltoNetworks "github.com/hashicorp/go-azure-sdk/resource-manager/paloaltonetworks/2022-08-29"
	resourceGraph "github.com/hashicorp/go-azure-sdk/resource-manager/resourcegraph/2022-10-01/resources"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2020-05-01/managementlocks"
	"github.com/hashicorp/go-azure-sdk/resource-manager/resources/2022-09-01/resourcegroups"
	serviceBus "github.com/hashicorp/go-azure-sdk/resource-manager/servicebus/2022-01-01-preview"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storagesync/2020-03-01/cloudendpointresource"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storagesync/2020-03-01/storagesyncservicesresource"
	"github.com/hashicorp/go-azure-sdk/resource-manager/storagesync/2020-03-01/syncgroupresource"
	"github.com/hashicorp/go-azure-sdk/sdk/auth"
	"github.com/hashicorp/go-azure-sdk/sdk/client/resourcemanager"
	"github.com/hashicorp/go-azure-sdk/sdk/environments"
	"github.com/manicminer/hamilton/msgraph"
)

type AzureClient struct {
	MicrosoftGraph  MicrosoftGraphClient
	ResourceManager ResourceManagerClient
	SubscriptionID  string
}

type MicrosoftGraphClient struct {
	Applications      *msgraph.ApplicationsClient
	Groups            *msgraph.GroupsClient
	ServicePrincipals *msgraph.ServicePrincipalsClient
	Users             *msgraph.UsersClient
}

type ResourceManagerClient struct {
	DataProtection                          *dataProtection.Client
	ExpressRouteCircuitsClient              *expressroutecircuits.ExpressRouteCircuitsClient
	ExpressRouteCircuitAuthorizationsClient *expressroutecircuitauthorizations.ExpressRouteCircuitAuthorizationsClient
	ExpressRouteCircuitConnectionsClient    *expressroutecircuitconnections.ExpressRouteCircuitConnectionsClient
	ExpressRouteCircuitPeeringsClient       *expressroutecircuitpeerings.ExpressRouteCircuitPeeringsClient
	LocksClient                             *managementlocks.ManagementLocksClient
	MachineLearningWorkspacesClient         *workspaces.WorkspacesClient
	ManagedHSMsClient                       *managedhsms.ManagedHsmsClient
	ManagementClient                        *managementgroups.ManagementGroupsClient
	NetAppAccountClient                     *netappaccounts.NetAppAccountsClient
	NetAppCapacityPoolClient                *capacitypools.CapacityPoolsClient
	NetAppVolumeClient                      *volumes.VolumesClient
	NetAppVolumeReplicationClient           *volumesreplication.VolumesReplicationClient
	NotificationHubNamespaceClient          *namespaces.NamespacesClient
	PaloAlto                                *paloAltoNetworks.Client
	ResourceGraphClient                     *resourceGraph.ResourcesClient
	ResourcesGroupsClient                   *resourcegroups.ResourceGroupsClient
	ServiceBus                              *serviceBus.Client
	StorageSyncClient                       *storagesyncservicesresource.StorageSyncServicesResourceClient
	StorageSyncGroupClient                  *syncgroupresource.SyncGroupResourceClient
	StorageSyncCloudEndpointClient          *cloudendpointresource.CloudEndpointResourceClient
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
	environment, err := environmentFromCredentials(ctx, credentials)
	if err != nil {
		return nil, fmt.Errorf("determining Environment: %+v", err)
	}

	creds := auth.Credentials{
		ClientID:     credentials.ClientID,
		ClientSecret: credentials.ClientSecret,
		TenantID:     credentials.TenantID,
		Environment:  *environment,

		EnableAuthenticatingUsingClientSecret: true,
	}

	resourceManager, err := buildResourceManagerClient(ctx, creds, *environment)
	if err != nil {
		return nil, fmt.Errorf("building Resource Manager client: %+v", err)
	}

	microsoftGraph, err := buildMicrosoftGraphClient(ctx, creds, *environment)
	if err != nil {
		return nil, fmt.Errorf("building Microsoft Graph client: %+v", err)
	}

	azureClient := AzureClient{
		MicrosoftGraph:  *microsoftGraph,
		ResourceManager: *resourceManager,
		SubscriptionID:  credentials.SubscriptionID,
	}

	return &azureClient, nil
}

func environmentFromCredentials(ctx context.Context, credentials Credentials) (*environments.Environment, error) {
	if strings.Contains(strings.ToLower(credentials.EnvironmentName), "stack") {
		// for Azure Stack we have to load the Environment from the URI
		env, err := environments.FromEndpoint(ctx, credentials.Endpoint, credentials.EnvironmentName)
		if err != nil {
			return nil, fmt.Errorf("loading from Endpoint %q: %s", credentials.Endpoint, err)
		}

		return env, nil
	}

	env, err := environments.FromName(credentials.EnvironmentName)
	if err != nil {
		return nil, fmt.Errorf("loading with Name %q: %s", credentials.EnvironmentName, err)
	}

	return env, nil
}

func buildMicrosoftGraphClient(ctx context.Context, creds auth.Credentials, environment environments.Environment) (*MicrosoftGraphClient, error) {
	microsoftGraphAuthorizer, err := auth.NewAuthorizerFromCredentials(ctx, creds, environment.MicrosoftGraph)
	if err != nil {
		return nil, fmt.Errorf("building Microsoft Graph authorizer: %+v", err)
	}
	microsoftGraphEndpoint, ok := environment.MicrosoftGraph.Endpoint()
	if !ok {
		return nil, fmt.Errorf("environment %q was missing a Microsoft Graph endpoint", environment.Name)
	}

	applicationsClient := msgraph.NewApplicationsClient()
	applicationsClient.BaseClient.Authorizer = microsoftGraphAuthorizer
	applicationsClient.BaseClient.Endpoint = *microsoftGraphEndpoint

	groupsClient := msgraph.NewGroupsClient()
	groupsClient.BaseClient.Authorizer = microsoftGraphAuthorizer
	groupsClient.BaseClient.Endpoint = *microsoftGraphEndpoint

	servicePrincipalsClient := msgraph.NewServicePrincipalsClient()
	servicePrincipalsClient.BaseClient.Authorizer = microsoftGraphAuthorizer
	servicePrincipalsClient.BaseClient.Endpoint = *microsoftGraphEndpoint

	usersClient := msgraph.NewUsersClient()
	usersClient.BaseClient.Authorizer = microsoftGraphAuthorizer
	usersClient.BaseClient.Endpoint = *microsoftGraphEndpoint

	return &MicrosoftGraphClient{
		Applications:      applicationsClient,
		Groups:            groupsClient,
		ServicePrincipals: servicePrincipalsClient,
		Users:             usersClient,
	}, nil
}

func buildResourceManagerClient(ctx context.Context, creds auth.Credentials, environment environments.Environment) (*ResourceManagerClient, error) {
	resourceManagerAuthorizer, err := auth.NewAuthorizerFromCredentials(ctx, creds, environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Resource Manager authorizer: %+v", err)
	}

	dataProtectionClient, err := dataProtection.NewClientWithBaseURI(environment.ResourceManager, func(c *resourcemanager.Client) {
		c.Authorizer = resourceManagerAuthorizer
	})

	expressRouteCircuitsClient, err := expressroutecircuits.NewExpressRouteCircuitsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Express Route Circuits client: %+v", err)
	}
	expressRouteCircuitsClient.Client.Authorizer = resourceManagerAuthorizer

	expressRouteCircuitAuthorizationsClient, err := expressroutecircuitauthorizations.NewExpressRouteCircuitAuthorizationsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Express Route Circuit Authorizations client: %+v", err)
	}
	expressRouteCircuitAuthorizationsClient.Client.Authorizer = resourceManagerAuthorizer

	expressRouteCircuitConnectionsClient, err := expressroutecircuitconnections.NewExpressRouteCircuitConnectionsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Express Route Circuit Connections client: %+v", err)
	}
	expressRouteCircuitConnectionsClient.Client.Authorizer = resourceManagerAuthorizer

	expressRouteCircuitPeeringsClient, err := expressroutecircuitpeerings.NewExpressRouteCircuitPeeringsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Express Route Circuit Peerings client: %+v", err)
	}
	expressRouteCircuitPeeringsClient.Client.Authorizer = resourceManagerAuthorizer

	locksClient, err := managementlocks.NewManagementLocksClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building ManagementLocks client: %+v", err)
	}
	locksClient.Client.Authorizer = resourceManagerAuthorizer

	workspacesClient, err := workspaces.NewWorkspacesClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Machine Learning Workspaces Client: %+v", err)
	}
	workspacesClient.Client.Authorizer = resourceManagerAuthorizer

	managementClient, err := managementgroups.NewManagementGroupsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building ManagementGroups client: %+v", err)
	}
	managementClient.Client.Authorizer = resourceManagerAuthorizer

	managedHsmsClient, err := managedhsms.NewManagedHsmsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Managed HSM Client: %+v", err)
	}
	managedHsmsClient.Client.Authorizer = resourceManagerAuthorizer

	netAppAccountClient, err := netappaccounts.NewNetAppAccountsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building NetApp Account Client: %+v", err)
	}
	netAppAccountClient.Client.Authorizer = resourceManagerAuthorizer

	netAppCapacityPoolClient, err := capacitypools.NewCapacityPoolsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building NetApp Capacity Pool Client: %+v", err)
	}
	netAppCapacityPoolClient.Client.Authorizer = resourceManagerAuthorizer

	netAppVolumeClient, err := volumes.NewVolumesClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building NetApp Volume Client: %+v", err)
	}
	netAppVolumeClient.Client.Authorizer = resourceManagerAuthorizer

	netAppVolumeReplicationClient, err := volumesreplication.NewVolumesReplicationClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building NetApp Volume Replication Client: %+v", err)
	}
	netAppVolumeReplicationClient.Client.Authorizer = resourceManagerAuthorizer

	notificationHubNamespacesClient, err := namespaces.NewNamespacesClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Notification Hub Namespaces Client: %+v", err)
	}
	notificationHubNamespacesClient.Client.Authorizer = resourceManagerAuthorizer

	paloAltoClient, err := paloAltoNetworks.NewClientWithBaseURI(environment.ResourceManager, func(c *resourcemanager.Client) {
		c.Authorizer = resourceManagerAuthorizer
	})

	resourceGraphClient, err := resourceGraph.NewResourcesClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building ResourceGraph client: %+v", err)
	}
	resourceGraphClient.Client.Authorizer = resourceManagerAuthorizer

	resourcesClient, err := resourcegroups.NewResourceGroupsClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building Resources client: %+v", err)
	}
	resourcesClient.Client.Authorizer = resourceManagerAuthorizer

	serviceBusClient, err := serviceBus.NewClientWithBaseURI(environment.ResourceManager, func(c *resourcemanager.Client) {
		c.Authorizer = resourceManagerAuthorizer
	})
	if err != nil {
		return nil, fmt.Errorf("building ServiceBus Client: %+v", err)
	}

	storageSyncClient, err := storagesyncservicesresource.NewStorageSyncServicesResourceClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building StorageSync Client: %+v", err)
	}
	storageSyncClient.Client.Authorizer = resourceManagerAuthorizer

	storageSyncGroupClient, err := syncgroupresource.NewSyncGroupResourceClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building StorageSyncGroup Client: %+v", err)
	}
	storageSyncGroupClient.Client.Authorizer = resourceManagerAuthorizer

	storageSyncCloudEndpointClient, err := cloudendpointresource.NewCloudEndpointResourceClientWithBaseURI(environment.ResourceManager)
	if err != nil {
		return nil, fmt.Errorf("building StorageSyncCloudEndpoint Client: %+v", err)
	}
	storageSyncCloudEndpointClient.Client.Authorizer = resourceManagerAuthorizer

	return &ResourceManagerClient{
		DataProtection:                          dataProtectionClient,
		ExpressRouteCircuitsClient:              expressRouteCircuitsClient,
		ExpressRouteCircuitAuthorizationsClient: expressRouteCircuitAuthorizationsClient,
		ExpressRouteCircuitConnectionsClient:    expressRouteCircuitConnectionsClient,
		ExpressRouteCircuitPeeringsClient:       expressRouteCircuitPeeringsClient,
		LocksClient:                             locksClient,
		MachineLearningWorkspacesClient:         workspacesClient,
		ManagedHSMsClient:                       managedHsmsClient,
		ManagementClient:                        managementClient,
		NetAppAccountClient:                     netAppAccountClient,
		NetAppCapacityPoolClient:                netAppCapacityPoolClient,
		NetAppVolumeClient:                      netAppVolumeClient,
		NetAppVolumeReplicationClient:           netAppVolumeReplicationClient,
		NotificationHubNamespaceClient:          notificationHubNamespacesClient,
		PaloAlto:                                paloAltoClient,
		ResourceGraphClient:                     resourceGraphClient,
		ResourcesGroupsClient:                   resourcesClient,
		ServiceBus:                              serviceBusClient,
		StorageSyncClient:                       storageSyncClient,
		StorageSyncGroupClient:                  storageSyncGroupClient,
		StorageSyncCloudEndpointClient:          storageSyncCloudEndpointClient,
	}, nil
}
