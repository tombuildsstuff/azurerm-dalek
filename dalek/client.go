package dalek

import (
	"github.com/tombuildsstuff/azurerm-dalek/clients"
	"github.com/tombuildsstuff/azurerm-dalek/dalek/options"
)

type Dalek struct {
	client *clients.AzureClient
	opts   options.Options
}

func NewDalek(client *clients.AzureClient, opts options.Options) Dalek {
	return Dalek{
		client: client,
		opts:   opts,
	}
}
