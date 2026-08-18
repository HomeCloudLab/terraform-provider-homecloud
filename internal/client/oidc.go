package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type AssumeRoleWithWebIdentityInput struct {
	RoleARN          string `json:"role_arn"`
	WebIdentityToken string `json:"web_identity_token"`
	SessionName      string `json:"session_name,omitempty"`
	DurationSeconds  int    `json:"duration_seconds,omitempty"`
}

type AssumeRoleWithWebIdentityOutput struct {
	RoleARN         string `json:"role_arn"`
	AssumedRoleID   string `json:"assumed_role_id"`
	SessionName     string `json:"session_name"`
	FederatedSub    string `json:"federated_sub"`
	AccountID       string `json:"account_id"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	SessionToken    string `json:"session_token"`
	ExpiresAt       string `json:"expires_at"`
}

func (c *Client) AssumeRoleWithWebIdentity(ctx context.Context, in AssumeRoleWithWebIdentityInput) (*AssumeRoleWithWebIdentityOutput, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/sts/assume-role-with-web-identity"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.Endpoint, "/")+path, strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, body)
	}
	var out AssumeRoleWithWebIdentityOutput
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.AccessKeyID == "" || out.SecretAccessKey == "" || out.SessionToken == "" {
		return nil, fmt.Errorf("assume-role-with-web-identity did not return credentials")
	}
	return &out, nil
}

func FetchGitHubOIDCToken(ctx context.Context, audience string) (string, error) {
	reqURL := strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"))
	reqTok := strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN"))
	if reqURL == "" || reqTok == "" {
		return "", fmt.Errorf("not running in GitHub Actions (set HC_WEB_IDENTITY_TOKEN or ACTIONS_ID_TOKEN_REQUEST_*)")
	}
	u := reqURL
	if strings.TrimSpace(audience) != "" {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u += sep + "audience=" + url.QueryEscape(audience)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+reqTok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("GitHub OIDC token request failed: %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	if strings.TrimSpace(parsed.Value) == "" {
		return "", fmt.Errorf("GitHub OIDC token response missing value")
	}
	return parsed.Value, nil
}
