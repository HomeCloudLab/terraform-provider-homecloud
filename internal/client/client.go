package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/sigv1"
)

type Client struct {
	Endpoint     string
	AccessKeyID  string
	Secret       string
	SessionToken string
	AccountID    string
	HTTP         *http.Client
	Now          func() time.Time
}

type APIError struct {
	Status  int
	Code    string
	Message string
	Body    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("homecloud api %d %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("homecloud api %d: %s", e.Status, e.Message)
}

func (e *APIError) NotFound() bool {
	return e.Status == http.StatusNotFound
}

type Whoami struct {
	AccountID      string  `json:"account_id"`
	AccountShortID string  `json:"account_short_id"`
	AccountNumber  string  `json:"account_number"`
	AccessKeyID    string  `json:"access_key_id"`
	PrincipalType  string  `json:"principal_type"`
	PrincipalID    *string `json:"principal_id"`
}

type Account struct {
	ID            string `json:"id"`
	ShortID       string `json:"short_id"`
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	Slug          string `json:"slug"`
	Status        string `json:"status"`
}

type Queue struct {
	ID                       string `json:"id"`
	Name                     string `json:"name"`
	Status                   string `json:"status"`
	IamARN                   string `json:"iam_arn"`
	QueueURL                 string `json:"queue_url"`
	MaxMessageSize           *int64 `json:"max_message_size"`
	VisibilityTimeoutSeconds *int64 `json:"visibility_timeout_seconds"`
	MaxReceiveCount          *int64 `json:"max_receive_count"`
	MessageRetentionSeconds  *int64 `json:"message_retention_seconds"`
}

type QueueCreate struct {
	Name                     string `json:"name"`
	MaxMessageSize           *int64 `json:"max_message_size,omitempty"`
	VisibilityTimeoutSeconds *int64 `json:"visibility_timeout_seconds,omitempty"`
	MaxReceiveCount          *int64 `json:"max_receive_count,omitempty"`
	MessageRetentionSeconds  *int64 `json:"message_retention_seconds,omitempty"`
}

type QueueUpdate struct {
	MaxMessageSize           *int64 `json:"max_message_size,omitempty"`
	VisibilityTimeoutSeconds *int64 `json:"visibility_timeout_seconds,omitempty"`
	MaxReceiveCount          *int64 `json:"max_receive_count,omitempty"`
	MessageRetentionSeconds  *int64 `json:"message_retention_seconds,omitempty"`
}

type Bucket struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	IamARN    string  `json:"iam_arn"`
	Status    string  `json:"status"`
	CreatedAt *string `json:"created_at"`
}

type Secret struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Status          string   `json:"status"`
	IamARN          string   `json:"iam_arn"`
	Description     *string  `json:"description"`
	Version         int64    `json:"version"`
	HasValue        bool     `json:"has_value"`
	KeyNames        []string `json:"key_names"`
	KeyCount        int64    `json:"key_count"`
	ApproxSizeBytes int64    `json:"approx_size_bytes"`
}

type SecretCreate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Client) Do(ctx context.Context, method, path, accountID, idempotencyKey string, payload any) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(raw)
	}
	url := strings.TrimRight(c.Endpoint, "/") + path
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	for k, v := range sigv1.SignHeaders(c.AccessKeyID, c.Secret, method, path, accountID, c.now()) {
		req.Header.Set(k, v)
	}
	if strings.TrimSpace(c.SessionToken) != "" {
		req.Header.Set(sigv1.HeaderSessionToken, c.SessionToken)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		return raw, resp.StatusCode, parseAPIError(resp.StatusCode, raw)
	}
	return raw, resp.StatusCode, nil
}

func parseAPIError(status int, raw []byte) error {
	var envelope struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
		Detail any `json:"detail"`
	}
	_ = json.Unmarshal(raw, &envelope)
	msg := envelope.Error.Message
	code := envelope.Error.Code
	if msg == "" {
		msg = strings.TrimSpace(string(raw))
	}
	return &APIError{Status: status, Code: code, Message: msg, Body: string(raw)}
}

func (c *Client) Whoami(ctx context.Context) (*Whoami, error) {
	raw, _, err := c.Do(ctx, http.MethodGet, "/api/v1/access-key/whoami", sigv1.WhoamiAccountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Whoami
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResolveAccountID(ctx context.Context) (string, error) {
	if strings.TrimSpace(c.AccountID) != "" {
		return c.AccountID, nil
	}
	who, err := c.Whoami(ctx)
	if err != nil {
		return "", err
	}
	if who.AccountID == "" {
		return "", fmt.Errorf("whoami did not return account_id")
	}
	c.AccountID = who.AccountID
	return c.AccountID, nil
}

func (c *Client) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	path := "/api/v1/accounts/" + accountID
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Account
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateQueue(ctx context.Context, accountID string, body QueueCreate) (*Queue, error) {
	path := "/api/v1/accounts/" + accountID + "/queues"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("mq_queue", accountID, body.Name), body)
	if err != nil {
		return nil, err
	}
	var out Queue
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetQueue(ctx context.Context, accountID, name string) (*Queue, error) {
	path := "/api/v1/accounts/" + accountID + "/queues/" + name
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Queue
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateQueue(ctx context.Context, accountID, name string, body QueueUpdate) (*Queue, error) {
	path := "/api/v1/accounts/" + accountID + "/queues/" + name
	raw, _, err := c.Do(ctx, http.MethodPatch, path, accountID, "", body)
	if err != nil {
		return nil, err
	}
	var out Queue
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteQueue(ctx context.Context, accountID, name string) error {
	path := "/api/v1/accounts/" + accountID + "/queues/" + name
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) CreateBucket(ctx context.Context, accountID, name string) (*Bucket, error) {
	path := "/api/v1/accounts/" + accountID + "/storage/buckets"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("so_bucket", accountID, name), map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	var out Bucket
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBucket(ctx context.Context, accountID, name string) (*Bucket, error) {
	path := "/api/v1/accounts/" + accountID + "/storage/buckets/" + name
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Bucket
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteBucket(ctx context.Context, accountID, name string) error {
	path := "/api/v1/accounts/" + accountID + "/storage/buckets/" + name
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) CreateSecret(ctx context.Context, accountID string, body SecretCreate) (*Secret, error) {
	path := "/api/v1/accounts/" + accountID + "/secrets"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("secret", accountID, body.Name), body)
	if err != nil {
		return nil, err
	}
	var out Secret
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSecret(ctx context.Context, accountID, name string) (*Secret, error) {
	path := "/api/v1/accounts/" + accountID + "/secrets/" + name
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Secret
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSecret(ctx context.Context, accountID, name, description string) (*Secret, error) {
	path := "/api/v1/accounts/" + accountID + "/secrets/" + name
	payload := map[string]*string{"description": nil}
	if description != "" {
		payload["description"] = &description
	}
	raw, _, err := c.Do(ctx, http.MethodPatch, path, accountID, "", payload)
	if err != nil {
		return nil, err
	}
	var out Secret
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PutSecretValue(ctx context.Context, accountID, name string, values map[string]string) (*Secret, error) {
	path := "/api/v1/accounts/" + accountID + "/secrets/" + name + "/value"
	raw, _, err := c.Do(ctx, http.MethodPut, path, accountID, "", map[string]map[string]string{"values": values})
	if err != nil {
		return nil, err
	}
	var out Secret
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSecret(ctx context.Context, accountID, name string) error {
	path := "/api/v1/accounts/" + accountID + "/secrets/" + name
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

type IAMPolicy struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	ARN            string          `json:"arn"`
	PolicyType     string          `json:"policy_type"`
	DefaultVersion int             `json:"default_version"`
	Description    *string         `json:"description"`
	Document       json.RawMessage `json:"document"`
	CreatedAt      *string         `json:"created_at"`
}

type IAMRole struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	ARN           string          `json:"arn"`
	Description   *string         `json:"description"`
	TrustDocument json.RawMessage `json:"trust_document"`
	RoleVersion   int             `json:"role_version"`
	PolicyARNs    []string        `json:"policy_arns"`
	CreatedAt     *string         `json:"created_at"`
}

type IAMServiceAccount struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ARN         string  `json:"arn"`
	Description *string `json:"description"`
}

type IAMAttachment struct {
	ID            string `json:"id"`
	PolicyID      string `json:"policy_id"`
	PolicyARN     string `json:"policy_arn"`
	PrincipalType string `json:"principal_type"`
	PrincipalID   string `json:"principal_id"`
}

func (c *Client) CreateIAMPolicy(ctx context.Context, accountID, name, description string, document json.RawMessage) (*IAMPolicy, error) {
	path := "/api/v1/accounts/" + accountID + "/iam/policies"
	body := struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Document    json.RawMessage `json:"document"`
	}{Name: name, Description: description, Document: document}
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, "", body)
	if err != nil {
		return nil, err
	}
	var out IAMPolicy
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func iamPathRef(ref string) string {
	return url.PathEscape(strings.TrimSpace(ref))
}

func (c *Client) GetIAMPolicy(ctx context.Context, accountID, ref string) (*IAMPolicy, error) {
	path := "/api/v1/accounts/" + accountID + "/iam/policies/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out IAMPolicy
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PutIAMPolicyDocument(ctx context.Context, accountID, ref string, document json.RawMessage) (*IAMPolicy, error) {
	path := "/api/v1/accounts/" + accountID + "/iam/policies/" + iamPathRef(ref) + "/versions"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, "", struct {
		Document     json.RawMessage `json:"document"`
		SetAsDefault bool            `json:"set_as_default"`
	}{Document: document, SetAsDefault: true})
	if err != nil {
		return nil, err
	}
	var out IAMPolicy
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteIAMPolicy(ctx context.Context, accountID, ref string) error {
	path := "/api/v1/accounts/" + accountID + "/iam/policies/" + iamPathRef(ref)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) CreateIAMRole(ctx context.Context, accountID, name, description string, trust json.RawMessage) (*IAMRole, error) {
	path := "/api/v1/accounts/" + accountID + "/iam/roles"
	body := struct {
		Name          string          `json:"name"`
		Description   string          `json:"description,omitempty"`
		TrustDocument json.RawMessage `json:"trust_document,omitempty"`
	}{Name: name, Description: description, TrustDocument: trust}
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, "", body)
	if err != nil {
		return nil, err
	}
	var out IAMRole
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetIAMRole(ctx context.Context, accountID, ref string) (*IAMRole, error) {
	path := "/api/v1/accounts/" + accountID + "/iam/roles/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out IAMRole
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateIAMRoleTrust(ctx context.Context, accountID, ref string, trust json.RawMessage) (*IAMRole, error) {
	path := "/api/v1/accounts/" + accountID + "/iam/roles/" + iamPathRef(ref) + "/trust"
	raw, _, err := c.Do(ctx, http.MethodPut, path, accountID, "", struct {
		TrustDocument json.RawMessage `json:"trust_document"`
	}{TrustDocument: trust})
	if err != nil {
		return nil, err
	}
	var out IAMRole
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteIAMRole(ctx context.Context, accountID, ref string) error {
	path := "/api/v1/accounts/" + accountID + "/iam/roles/" + iamPathRef(ref)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) GetIAMServiceAccount(ctx context.Context, accountID, ref string) (*IAMServiceAccount, error) {
	path := "/api/v1/accounts/" + accountID + "/iam/service-accounts/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out IAMServiceAccount
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) AttachIAMPolicy(ctx context.Context, accountID, policyARN, principalType, principalID string) error {
	path := "/api/v1/accounts/" + accountID + "/iam/principals/attachments"
	_, _, err := c.Do(ctx, http.MethodPost, path, accountID, "", map[string]string{
		"policy_arn":     policyARN,
		"principal_type": principalType,
		"principal_id":   principalID,
	})
	return err
}

func (c *Client) DetachIAMPolicy(ctx context.Context, accountID, policyARN, principalType, principalID string) error {
	path := "/api/v1/accounts/" + accountID + "/iam/principals/attachments"
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", map[string]string{
		"policy_arn":     policyARN,
		"principal_type": principalType,
		"principal_id":   principalID,
	})
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) ListIAMAttachments(ctx context.Context, accountID, principalType, principalID string) ([]IAMAttachment, error) {
	path := "/api/v1/accounts/" + accountID + "/iam/principals/" + iamPathRef(principalType) + "/" + iamPathRef(principalID) + "/attachments"
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Items []IAMAttachment `json:"items"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

type MDBConnection struct {
	Endpoint         string `json:"endpoint"`
	InternalEndpoint string `json:"internal_endpoint"`
	Port             int64  `json:"port"`
	Database         string `json:"database"`
	Username         string `json:"username"`
}

type Database struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	Status        string        `json:"status"`
	Engine        string        `json:"engine"`
	InstanceClass string        `json:"instance_class"`
	EngineVersion string        `json:"engine_version"`
	StorageGi     *int64        `json:"storage_gi"`
	IamARN        string        `json:"iam_arn"`
	Phase         string        `json:"phase"`
	Connection    MDBConnection `json:"connection"`
}

type DatabaseCreate struct {
	Name          string `json:"name"`
	Engine        string `json:"engine"`
	InstanceClass string `json:"instance_class,omitempty"`
	EngineVersion string `json:"engine_version,omitempty"`
	StorageGi     *int64 `json:"storage_gi,omitempty"`
	Database      string `json:"database,omitempty"`
	Owner         string `json:"owner,omitempty"`
}

type CacheConnection struct {
	Endpoint          string `json:"endpoint"`
	InternalEndpoint  string `json:"internal_endpoint"`
	Port              int64  `json:"port"`
	CredentialsSecret string `json:"credentials_secret"`
}

type Cache struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Status        string          `json:"status"`
	InstanceClass string          `json:"instance_class"`
	RedisVersion  string          `json:"redis_version"`
	IamARN        string          `json:"iam_arn"`
	Phase         string          `json:"phase"`
	Connection    CacheConnection `json:"connection"`
}

type CacheCreate struct {
	Name          string `json:"name"`
	InstanceClass string `json:"instance_class,omitempty"`
	RedisVersion  string `json:"redis_version,omitempty"`
}

type DatabaseUser struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	Phase    string `json:"phase"`
}

func (c *Client) WaitUntilActive(ctx context.Context, getStatus func() (string, error), timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := getStatus()
		if err != nil {
			return err
		}
		switch status {
		case "active":
			return nil
		case "failed":
			return fmt.Errorf("resource entered failed status")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for active (last status %s)", status)
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) WaitUntilSucceeded(ctx context.Context, getStatus func() (string, error), timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := getStatus()
		if err != nil {
			return err
		}
		switch strings.ToUpper(status) {
		case "SUCCEEDED":
			return nil
		case "FAILED":
			return fmt.Errorf("operation failed")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for succeeded (last status %s)", status)
		}
		timer := time.NewTimer(5 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (c *Client) CreateDatabase(ctx context.Context, accountID string, body DatabaseCreate) (*Database, error) {
	path := "/api/v1/accounts/" + accountID + "/databases"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("mdb_instance", accountID, body.Name), body)
	if err != nil {
		return nil, err
	}
	var out Database
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetDatabase(ctx context.Context, accountID, ref string) (*Database, error) {
	path := "/api/v1/accounts/" + accountID + "/databases/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Database
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteDatabase(ctx context.Context, accountID, ref string) error {
	path := "/api/v1/accounts/" + accountID + "/databases/" + iamPathRef(ref)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) CreateCache(ctx context.Context, accountID string, body CacheCreate) (*Cache, error) {
	path := "/api/v1/accounts/" + accountID + "/caches"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("redis_instance", accountID, body.Name), body)
	if err != nil {
		return nil, err
	}
	var out Cache
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetCache(ctx context.Context, accountID, ref string) (*Cache, error) {
	path := "/api/v1/accounts/" + accountID + "/caches/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Cache
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCache(ctx context.Context, accountID, ref string) error {
	path := "/api/v1/accounts/" + accountID + "/caches/" + iamPathRef(ref)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) CreateDatabaseUser(ctx context.Context, accountID, instanceRef, username, password, role, database string) (*DatabaseUser, error) {
	path := "/api/v1/accounts/" + accountID + "/databases/" + iamPathRef(instanceRef) + "/users"
	body := map[string]string{"username": username, "password": password, "role": role}
	if database != "" {
		body["database"] = database
	}
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, "", body)
	if err != nil {
		return nil, err
	}
	var out DatabaseUser
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetDatabaseUser(ctx context.Context, accountID, instanceRef, username string) (*DatabaseUser, error) {
	path := "/api/v1/accounts/" + accountID + "/databases/" + iamPathRef(instanceRef) + "/users/" + iamPathRef(username)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out DatabaseUser
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RotateDatabaseUser(ctx context.Context, accountID, instanceRef, username, password string) (*DatabaseUser, error) {
	path := "/api/v1/accounts/" + accountID + "/databases/" + iamPathRef(instanceRef) + "/users/" + iamPathRef(username) + "/rotate"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, "", map[string]string{"password": password})
	if err != nil {
		return nil, err
	}
	var out DatabaseUser
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteDatabaseUser(ctx context.Context, accountID, instanceRef, username string) error {
	path := "/api/v1/accounts/" + accountID + "/databases/" + iamPathRef(instanceRef) + "/users/" + iamPathRef(username)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

type Function struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	Status             string            `json:"status"`
	Runtime            string            `json:"runtime"`
	Handler            string            `json:"handler"`
	MemoryLimitMB      int64             `json:"memory_limit_mb"`
	TimeoutSeconds     int64             `json:"timeout_seconds"`
	Environment        map[string]string `json:"environment"`
	IamARN             string            `json:"iam_arn"`
	InvokeURL          string            `json:"invoke_url"`
	FunctionURL        string            `json:"function_url"`
	FunctionURLEnabled bool              `json:"function_url_enabled"`
	PublicURLEnabled   bool              `json:"public_url_enabled"`
}

type FunctionCreate struct {
	Name           string            `json:"name"`
	Runtime        string            `json:"runtime,omitempty"`
	Handler        string            `json:"handler,omitempty"`
	MemoryLimitMB  int64             `json:"memory_limit_mb,omitempty"`
	TimeoutSeconds int64             `json:"timeout_seconds,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
}

type FunctionUpdate struct {
	Runtime        string            `json:"runtime,omitempty"`
	Handler        string            `json:"handler,omitempty"`
	MemoryLimitMB  *int64            `json:"memory_limit_mb,omitempty"`
	TimeoutSeconds *int64            `json:"timeout_seconds,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
}

type FunctionURL struct {
	FunctionName       string `json:"function_name"`
	FunctionURL        string `json:"function_url"`
	FunctionURLEnabled bool   `json:"function_url_enabled"`
	PublicURLEnabled   bool   `json:"public_url_enabled"`
}

type FunctionURLEnable struct {
	PublicURLEnabled   bool  `json:"public_url_enabled"`
	RateLimitPerMinute int64 `json:"rate_limit_per_minute,omitempty"`
	MaxPayloadBytes    int64 `json:"max_payload_bytes,omitempty"`
}

type Repository struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Status         string   `json:"status"`
	ZotNamespace   string   `json:"zot_namespace"`
	KeepLast       *int64   `json:"keep_last"`
	ProtectedTags  []string `json:"protected_tags"`
	IamARN         string   `json:"iam_arn"`
	ImageRefPrefix string   `json:"image_ref_prefix"`
}

type RepositoryCreate struct {
	Name          string   `json:"name"`
	KeepLast      *int64   `json:"keep_last,omitempty"`
	ProtectedTags []string `json:"protected_tags,omitempty"`
}

type Domain struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FQDN     string `json:"fqdn"`
	Status   string `json:"status"`
	DNSMode  string `json:"dns_mode"`
	Verified bool   `json:"verified"`
	IamARN   string `json:"iam_arn"`
}

type DomainCreate struct {
	Hostname string `json:"hostname"`
	DNSMode  string `json:"dns_mode,omitempty"`
}

func (c *Client) CreateFunction(ctx context.Context, accountID string, body FunctionCreate) (*Function, error) {
	path := "/api/v1/accounts/" + accountID + "/functions"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("function", accountID, body.Name), body)
	if err != nil {
		return nil, err
	}
	var out Function
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetFunction(ctx context.Context, accountID, name string) (*Function, error) {
	path := "/api/v1/accounts/" + accountID + "/functions/" + iamPathRef(name)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Function
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateFunction(ctx context.Context, accountID, name string, body FunctionUpdate) (*Function, error) {
	path := "/api/v1/accounts/" + accountID + "/functions/" + iamPathRef(name)
	raw, _, err := c.Do(ctx, http.MethodPatch, path, accountID, "", body)
	if err != nil {
		return nil, err
	}
	var out Function
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteFunction(ctx context.Context, accountID, name string) error {
	path := "/api/v1/accounts/" + accountID + "/functions/" + iamPathRef(name)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) GetFunctionURL(ctx context.Context, accountID, name string) (*FunctionURL, error) {
	path := "/api/v1/accounts/" + accountID + "/functions/" + iamPathRef(name) + "/url"
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out FunctionURL
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) EnableFunctionURL(ctx context.Context, accountID, name string, body FunctionURLEnable) (*FunctionURL, error) {
	path := "/api/v1/accounts/" + accountID + "/functions/" + iamPathRef(name) + "/url/enable"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, "", body)
	if err != nil {
		return nil, err
	}
	var out FunctionURL
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DisableFunctionURL(ctx context.Context, accountID, name string) error {
	path := "/api/v1/accounts/" + accountID + "/functions/" + iamPathRef(name) + "/url/disable"
	_, _, err := c.Do(ctx, http.MethodPost, path, accountID, "", nil)
	return err
}

func (c *Client) CreateRepository(ctx context.Context, accountID string, body RepositoryCreate) (*Repository, error) {
	path := "/api/v1/accounts/" + accountID + "/registry/repositories"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("ir_repository", accountID, body.Name), body)
	if err != nil {
		return nil, err
	}
	var out Repository
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetRepository(ctx context.Context, accountID, name string) (*Repository, error) {
	path := "/api/v1/accounts/" + accountID + "/registry/repositories/" + iamPathRef(name)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Repository
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteRepository(ctx context.Context, accountID, name string) error {
	path := "/api/v1/accounts/" + accountID + "/registry/repositories/" + iamPathRef(name)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) CreateDomain(ctx context.Context, accountID string, body DomainCreate) (*Domain, error) {
	path := "/api/v1/accounts/" + accountID + "/domains"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("domain", accountID, body.Hostname), body)
	if err != nil {
		return nil, err
	}
	var out Domain
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetDomain(ctx context.Context, accountID, ref string) (*Domain, error) {
	path := "/api/v1/accounts/" + accountID + "/domains/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Domain
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteDomain(ctx context.Context, accountID, ref string) error {
	path := "/api/v1/accounts/" + accountID + "/domains/" + iamPathRef(ref)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func idempotencyKey(kind, accountID, name string) string {
	return "terraform-" + kind + "-" + accountID + "-" + name
}

type Nic struct {
	PublicIPv4 string `json:"public_ipv4"`
	PublicIPv6 string `json:"public_ipv6"`
	PrivateIP  string `json:"private_ip"`
}

type Machine struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Class         string `json:"class"`
	ImageID       string `json:"image_id"`
	RegionCode    string `json:"region_code"`
	AzCode        string `json:"az_code"`
	Status        string `json:"status"`
	ProviderState string `json:"provider_state"`
	DesiredState  string `json:"desired_state"`
	IamARN        string `json:"iam_arn"`
	Nic           Nic    `json:"nic"`
}

type MachineCreate struct {
	Name       string   `json:"name"`
	Class      string   `json:"class"`
	ImageID    string   `json:"image_id"`
	RegionCode string   `json:"region_code,omitempty"`
	AzCode     string   `json:"az_code,omitempty"`
	SSHKeyIDs  []string `json:"ssh_key_ids,omitempty"`
}

type MachineCreateResult struct {
	MachineID   string `json:"machine_id"`
	OperationID string `json:"operation_id"`
}

type Operation struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Action string `json:"action"`
	Error  string `json:"error"`
}

type SSHKey struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	KeyType     string `json:"key_type"`
	PublicKey   string `json:"public_key"`
	PrivateKey  string `json:"private_key"`
	IamARN      string `json:"iam_arn"`
}

type SSHKeyCreate struct {
	Name string `json:"name"`
}

type Application struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Template  string `json:"template"`
	Status    string `json:"status"`
	ProjectID string `json:"project_id"`
	IamARN    string `json:"iam_arn"`
}

type ApplicationCreate struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Template  string `json:"template,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

func (c *Client) GetOperation(ctx context.Context, accountID, operationID string) (*Operation, error) {
	path := "/api/v1/accounts/" + accountID + "/operations/" + iamPathRef(operationID)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Operation
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateMachine(ctx context.Context, accountID string, body MachineCreate) (*MachineCreateResult, error) {
	path := "/api/v1/accounts/" + accountID + "/compute/machines"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("compute_machine", accountID, body.Name), body)
	if err != nil {
		return nil, err
	}
	var out MachineCreateResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetMachine(ctx context.Context, accountID, ref string) (*Machine, error) {
	path := "/api/v1/accounts/" + accountID + "/compute/machines/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Machine
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteMachine(ctx context.Context, accountID, ref string) (*MachineCreateResult, error) {
	path := "/api/v1/accounts/" + accountID + "/compute/machines/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return &MachineCreateResult{}, nil
		}
		return nil, err
	}
	var out MachineCreateResult
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateSSHKey(ctx context.Context, accountID string, body SSHKeyCreate) (*SSHKey, error) {
	path := "/api/v1/accounts/" + accountID + "/compute/ssh-keys"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("ssh_key", accountID, body.Name), body)
	if err != nil {
		return nil, err
	}
	var out SSHKey
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSSHKey(ctx context.Context, accountID, ref string) (*SSHKey, error) {
	path := "/api/v1/accounts/" + accountID + "/compute/ssh-keys/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out SSHKey
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteSSHKey(ctx context.Context, accountID, ref string) error {
	path := "/api/v1/accounts/" + accountID + "/compute/ssh-keys/" + iamPathRef(ref)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}

func (c *Client) CreateApplication(ctx context.Context, accountID string, body ApplicationCreate) (*Application, error) {
	path := "/api/v1/accounts/" + accountID + "/applications"
	raw, _, err := c.Do(ctx, http.MethodPost, path, accountID, idempotencyKey("application", accountID, body.Slug), body)
	if err != nil {
		return nil, err
	}
	var out Application
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetApplication(ctx context.Context, accountID, ref string) (*Application, error) {
	path := "/api/v1/accounts/" + accountID + "/applications/" + iamPathRef(ref)
	raw, _, err := c.Do(ctx, http.MethodGet, path, accountID, "", nil)
	if err != nil {
		return nil, err
	}
	var out Application
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteApplication(ctx context.Context, accountID, ref string) error {
	path := "/api/v1/accounts/" + accountID + "/applications/" + iamPathRef(ref)
	_, _, err := c.Do(ctx, http.MethodDelete, path, accountID, "", nil)
	if err != nil {
		if apiErr, ok := err.(*APIError); ok && apiErr.NotFound() {
			return nil
		}
	}
	return err
}
