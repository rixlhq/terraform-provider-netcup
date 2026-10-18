package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/rixlhq/terraform-provider-netcup/internal/clouddnsclient"
)

func configuredCloudDNSClient(value any, diagnostics *diag.Diagnostics) *clouddnsclient.Client {
	if value == nil {
		return nil
	}
	data, ok := value.(*providerData)
	if !ok || data == nil {
		diagnostics.AddError("Unexpected Provider Data", fmt.Sprintf("Expected *providerData, got %T", value))
		return nil
	}
	if data.CloudDNS == nil {
		diagnostics.AddError("Missing CloudDNS Credentials", "Configure clouddns_api_key or NETCUP_CLOUDDNS_API_KEY. Legacy CCP keys and SCP tokens cannot authenticate CloudDNS requests.")
		return nil
	}
	return data.CloudDNS
}

func activeCloudDNSRecords(ctx context.Context, client *clouddnsclient.Client, domainID int64) ([]clouddnsclient.Record, error) {
	revisions, err := client.Revisions(ctx, domainID)
	if err != nil {
		return nil, err
	}
	active := ""
	for _, revision := range revisions {
		if strings.EqualFold(revision.State, "active") {
			if active != "" {
				return nil, errors.New("CloudDNS returned more than one active DNS revision")
			}
			active = revision.Identifier
		}
	}
	if active == "" {
		return nil, errors.New("CloudDNS returned no identifiable active DNS revision; refusing to use a draft or historical revision")
	}
	return client.Records(ctx, domainID, active)
}
