package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/sigv1"
)

type Client struct {
	Endpoint    string
	AccessKeyID string
	Secret      string
	AccountID   string
	HTTP        *http.Client
	Now         func() time.Time
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

func idempotencyKey(kind, accountID, name string) string {
	return "terraform-" + kind + "-" + accountID + "-" + name
}
