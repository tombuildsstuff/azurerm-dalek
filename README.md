# Azure Dalek

Azure Dalek will work through your Azure Subscription and delete everything it comes across (matching a filter).

~> **NOTE / BE AWARE:** This will delete resources in your Azure Subscription which do not include the tag `DoNotDelete` - please read the source to understand before running.

The Dalek supports using both a Service Principal and your Azure CLI credentials. The following Environment Variables can be configure:

* `ARM_CLIENT_ID` - (Optional) The Client ID associated with the Service Principal used for authentication
* `ARM_CLIENT_SECRET` - (Optional) The Client Secret associated with the Service Principal used for authentication
* `ARM_ENVIRONMENT` - (Optional) The Azure Environment which the tests should be run against, e.g. `public`, `german`, `azurestackcloud`. Defaults to `public`.
* `ARM_SUBSCRIPTION_ID` - The ID of the Azure Subscription within the Tenant
* `ARM_TENANT_ID` - The ID of the Azure Tenant
* `ARM_ENDPOINT` - (Optional) The URI of a Custom Resource Manager Endpoint, intended for use with Azure Stack.
* `YES_I_REALLY_WANT_TO_DELETE_THINGS` - (Optional) Set this to any value to actually delete resources

## Dependencies

* Go 1.11

## Example Usage

To delete using your Azure CLI Credentials against Azure Public:

```
$ go build .
$ YES_I_REALLY_WANT_TO_DELETE_THINGS="true" ./azurerm-dalek
```

To delete with a Service Principal against Azure Public:

```
$ export ARM_CLIENT_ID="00000000-0000-0000-0000-000000000000"
$ export ARM_CLIENT_SECRET="00000000-0000-0000-0000-000000000000"
$ export ARM_ENVIRONMENT="public"
$ export ARM_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
$ export ARM_TENANT_ID="00000000-0000-0000-0000-000000000000"
$ go build .
$ YES_I_REALLY_WANT_TO_DELETE_THINGS="true" ./azurerm-dalek
```

To delete using your Azure CLI Credentials against Azure Stack:

```
$ go build .
$ YES_I_REALLY_WANT_TO_DELETE_THINGS="true" ./azurerm-dalek
```

To delete with a Service Principal against Azure Stack:

```
$ export ARM_CLIENT_ID="00000000-0000-0000-0000-000000000000"
$ export ARM_CLIENT_SECRET="00000000-0000-0000-0000-000000000000"
$ export ARM_ENVIRONMENT="public"
$ export ARM_ENDPOINT="https://management.westus.mydomain.com
$ export ARM_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000000"
$ export ARM_TENANT_ID="00000000-0000-0000-0000-000000000000"
$ go build .
$ YES_I_REALLY_WANT_TO_DELETE_THINGS="true" ./azurerm-dalek
```

## Licence

MIT
