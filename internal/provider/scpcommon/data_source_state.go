package scpcommon

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// SetDataSourceState retains configured path and query inputs that are not echoed
// by the SCP API, while keeping computed values from the decoded response.
func SetDataSourceState(config tfsdk.Config, value tftypes.Value, response *datasource.ReadResponse) {
	var inputs, state map[string]tftypes.Value
	if err := config.Raw.As(&inputs); err != nil {
		response.Diagnostics.AddError("State Conversion Error", err.Error())
		return
	}
	if err := value.As(&state); err != nil {
		response.Diagnostics.AddError("State Conversion Error", err.Error())
		return
	}
	for name, attribute := range config.Schema.GetAttributes() {
		if !attribute.IsRequired() && !attribute.IsOptional() {
			continue
		}
		input := inputs[name]
		if input.IsKnown() && !input.IsNull() {
			state[name] = input
		}
	}
	response.State.Raw = tftypes.NewValue(value.Type(), state)
}
