package dalek

import "github.com/tombuildsstuff/azurerm-dalek/clients"

type Dalek struct {
	client *clients.AzureClient
	opts   Options
}

func NewDalek(client *clients.AzureClient, opts Options) Dalek {
	return Dalek{
		client: client,
		opts:   opts,
	}
}
