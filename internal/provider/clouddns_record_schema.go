package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func (r *cloudDNSRecordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one netcup CloudDNS record using strict, automatically committed changesets. Other records are preserved. Requires clouddns_api_key; legacy CCP DNS credentials cannot be used. Import with domain_id/record_identifier. Do not manage the same record through multiple resources.",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, Description: "Composite domain_id/record_identifier. Record identifiers can change when a revision replaces a record."},
			"domain_id": schema.Int64Attribute{Required: true, Description: "Domain ID from netcup_clouddns_domains.", Validators: []validator.Int64{int64validator.AtLeast(1)}, PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()}},
			"name":      schema.StringAttribute{Required: true, Description: "Record name relative to the zone; use @ for the apex or * for a wildcard.", Validators: []validator.String{stringvalidator.LengthBetween(1, 255)}},
			"type":      schema.StringAttribute{Required: true, Description: "Uppercase DNS record type.", Validators: []validator.String{stringvalidator.OneOf("A", "AAAA", "CAA", "CNAME", "DS", "MX", "NS", "OPENPGPKEY", "SRV", "SMIMEA", "SSHFP", "TLSA", "TXT")}},
			"rdata":     schema.StringAttribute{Required: true, Description: "Record data in zone-file presentation format. For MX use priority hostname, for SRV priority weight port target. TXT quoting follows the netcup API.", Validators: []validator.String{stringvalidator.LengthBetween(1, 65535)}},
			"ttl":       schema.Int64Attribute{Required: true, Description: "Record TTL in seconds, at least 1.", Validators: []validator.Int64{int64validator.AtLeast(1)}},
		},
	}
}
