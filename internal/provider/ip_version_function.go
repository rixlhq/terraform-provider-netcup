package provider

import (
	"context"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

var _ function.Function = &IPVersionFunction{}

// IPVersionFunction returns the IP version (ipv4 or ipv6) for a given address.
type IPVersionFunction struct{}

func NewIPVersionFunction() function.Function {
	return &IPVersionFunction{}
}

func (f *IPVersionFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "ip_version"
}

func (f *IPVersionFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:     "Return the IP version for an address",
		Description: "Given an IPv4 or IPv6 address, returns `ipv4` or `ipv6`. Returns an error if the address is invalid.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "ip",
				Description: "IPv4 or IPv6 address.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *IPVersionFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	if err := req.Arguments.Get(ctx, &input); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, err)
		return
	}

	ip := net.ParseIP(input)
	if ip == nil {
		resp.Error = function.NewFuncError("invalid IP address")
		return
	}

	version := "ipv6"
	if ip.To4() != nil {
		version = "ipv4"
	}

	if err := resp.Result.Set(ctx, version); err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, err)
	}
}
