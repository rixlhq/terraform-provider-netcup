package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/rixlhq/terraform-provider-netcup/internal/clouddnsclient"
)

var (
	_ resource.Resource                = &cloudDNSRecordResource{}
	_ resource.ResourceWithConfigure   = &cloudDNSRecordResource{}
	_ resource.ResourceWithImportState = &cloudDNSRecordResource{}
)

type cloudDNSRecordResource struct{ client *clouddnsclient.Client }

type cloudDNSRecordModel struct {
	ID       types.String `tfsdk:"id"`
	DomainID types.Int64  `tfsdk:"domain_id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	RData    types.String `tfsdk:"rdata"`
	TTL      types.Int64  `tfsdk:"ttl"`
}

// NewCloudDNSRecordResource manages one record through CloudDNS atomic changesets.
func NewCloudDNSRecordResource() resource.Resource { return &cloudDNSRecordResource{} }

func (r *cloudDNSRecordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clouddns_record"
}

func (r *cloudDNSRecordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configuredCloudDNSClient(req.ProviderData, &resp.Diagnostics)
}
