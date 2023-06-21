package clients

import "github.com/hashicorp/go-azure-helpers/lang/pointer"

type azureActiveDirectoryGraph struct {
}

func (a azureActiveDirectoryGraph) AppId() (*string, bool) {
	return nil, false
}

func (a azureActiveDirectoryGraph) DomainSuffix() (*string, bool) {
	return nil, false
}

func (a azureActiveDirectoryGraph) Endpoint() (*string, bool) {
	return pointer.To("https://graph.windows.net/"), true
}

func (a azureActiveDirectoryGraph) Name() string {
	return "AzureAD Graph (Legacy)"
}

func (a azureActiveDirectoryGraph) ResourceIdentifier() (*string, bool) {
	return pointer.To("https://graph.windows.net/"), true
}
