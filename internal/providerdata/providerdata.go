package providerdata

import (
	"github.com/rixlhq/terraform-provider-netcup/internal/client"
	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

// Data holds the API clients configured by the provider and passed to resources/data sources.
type Data struct {
	CCP *client.Client
	SCP *scpclient.Client
}
