package provider_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccScpUserFirewallPolicy_multipleSources covers a rule with several
// sources. The SCP API declares sources and destinations with uniqueItems:
// true and does not preserve their order, so the mock returns them reversed.
// The second step must produce an empty plan.
func TestAccScpUserFirewallPolicy_multipleSources(t *testing.T) {
	server := newFirewallPolicyMockServer(t)
	defer server.Close()

	config := firewallPolicyRuleConfig(server.URL, "multi", `
      sources = [
        "192.0.2.10/32",
        "192.0.2.20/32",
        "192.0.2.30/32",
      ]`)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.sources.#", "3"),
					resource.TestCheckTypeSetElemAttr("netcup_scp_user_firewall_policy.test", "rules.0.sources.*", "192.0.2.10/32"),
					resource.TestCheckTypeSetElemAttr("netcup_scp_user_firewall_policy.test", "rules.0.sources.*", "192.0.2.20/32"),
					resource.TestCheckTypeSetElemAttr("netcup_scp_user_firewall_policy.test", "rules.0.sources.*", "192.0.2.30/32"),
				),
			},
			{
				Config:   config,
				PlanOnly: true,
			},
		},
	})
}

// TestAccScpUserFirewallPolicy_upgradeFromListState creates state with the
// released v1.2.1 provider, where sources/destinations were lists, then plans
// with this build, where they are sets. The plan must be empty and must not
// error while decoding the existing state.
func TestAccScpUserFirewallPolicy_upgradeFromListState(t *testing.T) {
	server := newFirewallPolicyMockServer(t)
	defer server.Close()

	config := firewallPolicyRuleConfig(server.URL, "upgrade", `
      sources      = ["192.0.2.10/32"]
      destinations = ["192.0.2.1/32"]`)

	resource.Test(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				ExternalProviders: map[string]resource.ExternalProvider{
					"netcup": {Source: "rixlhq/netcup", VersionConstraint: "1.2.1"},
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.sources.#", "1"),
					resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.destinations.#", "1"),
				),
			},
			{
				ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
				Config:                   config,
				PlanOnly:                 true,
			},
		},
	})
}

// TestAccScpUserFirewallPolicy_emptyAndNullSources makes sure the set
// conversion keeps "empty" (any address) and "omitted" apart and that neither
// produces a perpetual diff.
func TestAccScpUserFirewallPolicy_emptyAndNullSources(t *testing.T) {
	server := newFirewallPolicyMockServer(t)
	defer server.Close()

	emptyConfig := firewallPolicyRuleConfig(server.URL, "empty", `
      sources = []`)
	nullConfig := firewallPolicyRuleConfig(server.URL, "null", "")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: emptyConfig,
				Check:  resource.TestCheckResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.sources.#", "0"),
			},
			{
				Config:   emptyConfig,
				PlanOnly: true,
			},
			{
				Config: nullConfig,
				Check:  resource.TestCheckNoResourceAttr("netcup_scp_user_firewall_policy.test", "rules.0.sources.#"),
			},
			{
				Config:   nullConfig,
				PlanOnly: true,
			},
		},
	})
}

// firewallPolicyRuleConfig renders a single-rule policy against the mock at
// serverURL. extraAttrs is appended verbatim inside the rule block.
func firewallPolicyRuleConfig(serverURL, desc, extraAttrs string) string {
	return fmt.Sprintf(`
provider "netcup" {
  scp_access_token = "mock-token"
  scp_base_url     = "%s/scp-core"
}

resource "netcup_scp_user_firewall_policy" "test" {
  user_id     = 12345
  name        = "test-policy"
  description = %q
  rules = [
    {
      action            = "ACCEPT"
      direction         = "INGRESS"
      protocol          = "TCP"
      destination_ports = "443"%s
    }
  ]
}
`, serverURL, desc, extraAttrs)
}
