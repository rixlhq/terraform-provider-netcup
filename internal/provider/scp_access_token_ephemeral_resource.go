package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const defaultTokenURL = "https://www.servercontrolpanel.de/realms/scp/protocol/openid-connect/token"

var _ ephemeral.EphemeralResource = &ScpAccessTokenEphemeralResource{}

// ScpAccessTokenEphemeralResource exchanges a netcup SCP refresh token for a
// short-lived access token. The token is not persisted in Terraform state.
type ScpAccessTokenEphemeralResource struct{}

// ScpAccessTokenModel describes the ephemeral resource configuration and result.
type ScpAccessTokenModel struct {
	RefreshToken types.String `tfsdk:"refresh_token"`
	ClientID     types.String `tfsdk:"client_id"`
	TokenURL     types.String `tfsdk:"token_url"`
	AccessToken  types.String `tfsdk:"access_token"`
	ExpiresIn    types.Int64  `tfsdk:"expires_in"`
	TokenType    types.String `tfsdk:"token_type"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func NewScpAccessTokenEphemeralResource() ephemeral.EphemeralResource {
	return &ScpAccessTokenEphemeralResource{}
}

func (e *ScpAccessTokenEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scp_access_token"
}

func (e *ScpAccessTokenEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Exchanges a netcup SCP refresh token for a short-lived access token.",
		MarkdownDescription: "Exchanges a netcup SCP refresh token for a short-lived access token. The result is not stored in Terraform state.",
		Attributes: map[string]schema.Attribute{
			"refresh_token": schema.StringAttribute{
				MarkdownDescription: "Offline refresh token for the netcup SCP REST API.",
				Required:            true,
				Sensitive:           true,
			},
			"client_id": schema.StringAttribute{
				MarkdownDescription: "OAuth2 client_id. Defaults to `scp`.",
				Optional:            true,
			},
			"token_url": schema.StringAttribute{
				MarkdownDescription: "OAuth2 token endpoint. Defaults to the netcup SCP Keycloak token endpoint.",
				Optional:            true,
			},
			"access_token": schema.StringAttribute{
				MarkdownDescription: "Short-lived bearer access token.",
				Computed:            true,
				Sensitive:           true,
			},
			"expires_in": schema.Int64Attribute{
				MarkdownDescription: "Access token lifetime in seconds.",
				Computed:            true,
			},
			"token_type": schema.StringAttribute{
				MarkdownDescription: "Type of token, typically `Bearer`.",
				Computed:            true,
			},
		},
	}
}

func (e *ScpAccessTokenEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	var data ScpAccessTokenModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clientID := stringOrDefault(data.ClientID, "scp")
	tokenURL := stringOrDefault(data.TokenURL, defaultTokenURL)

	httpReq, err := tokenRequest(ctx, tokenURL, clientID, data.RefreshToken.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Token Request Error", err.Error())
		return
	}

	tr, err := doTokenExchange(httpReq)
	if err != nil {
		resp.Diagnostics.AddError("Token Exchange Error", err.Error())
		return
	}

	data.AccessToken = types.StringValue(tr.AccessToken)
	data.ExpiresIn = types.Int64Value(tr.ExpiresIn)
	data.TokenType = types.StringValue(tr.TokenType)

	resp.Diagnostics.Append(resp.Result.Set(ctx, data)...)
}

func stringOrDefault(v types.String, fallback string) string {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return fallback
	}
	return v.ValueString()
}

func tokenRequest(ctx context.Context, tokenURL, clientID, refreshToken string) (*http.Request, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}

func doTokenExchange(httpReq *http.Request) (tokenResponse, error) {
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return tokenResponse{}, err
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return tokenResponse{}, err
	}

	if httpResp.StatusCode >= 400 {
		return tokenResponse{}, fmt.Errorf("token endpoint returned %d: %s", httpResp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return tokenResponse{}, err
	}
	return tr, nil
}
