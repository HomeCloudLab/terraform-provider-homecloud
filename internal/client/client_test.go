package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/sigv1"
)

func TestDoSignsCanonicalPath(t *testing.T) {
	accountID := "11111111-1111-1111-1111-111111111111"
	secret := "test-secret-for-vectors"
	fixed := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	var gotMethod, gotPath, gotAccount, gotSig, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAccount = r.Header.Get(sigv1.HeaderDate)
		gotSig = r.Header.Get(sigv1.HeaderSignature)
		gotKey = r.Header.Get(sigv1.HeaderAccessKeyID)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "res-1",
			"name":    "jobs",
			"status":  "active",
			"iam_arn": "arn:homecloud:mq::123456789012:queue/jobs",
		})
	}))
	defer srv.Close()

	c := &Client{
		Endpoint:    srv.URL,
		AccessKeyID: "HCAKTEST",
		Secret:      secret,
		Now:         func() time.Time { return fixed },
	}
	path := "/api/v1/accounts/" + accountID + "/queues"
	q, err := c.CreateQueue(context.Background(), accountID, QueueCreate{Name: "jobs"})
	if err != nil {
		t.Fatal(err)
	}
	if q.Name != "jobs" {
		t.Fatalf("name %s", q.Name)
	}
	if gotMethod != http.MethodPost || gotPath != path {
		t.Fatalf("request %s %s", gotMethod, gotPath)
	}
	if gotKey != "HCAKTEST" {
		t.Fatalf("access key header %s", gotKey)
	}
	want := sigv1.ComputeSignature(secret, sigv1.BuildStringToSign(http.MethodPost, path, gotAccount, accountID))
	if gotSig != want {
		t.Fatalf("signature\n got %s\nwant %s", gotSig, want)
	}
}

func TestWhoamiUsesSentinelAccount(t *testing.T) {
	fixed := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/access-key/whoami" {
			t.Errorf("path %s", r.URL.Path)
		}
		want := sigv1.ComputeSignature("s", sigv1.BuildStringToSign("GET", r.URL.Path, r.Header.Get(sigv1.HeaderDate), sigv1.WhoamiAccountID))
		if r.Header.Get(sigv1.HeaderSignature) != want {
			t.Errorf("whoami signature mismatch")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "acc-1", "account_number": "123"})
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s", Now: func() time.Time { return fixed }}
	who, err := c.Whoami(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if who.AccountID != "acc-1" {
		t.Fatalf("account %s", who.AccountID)
	}
}

func TestAPIErrorEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":{"code":"iam.management_sa_not_enabled","message":"not yet"}}`)
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s"}
	_, err := c.CreateQueue(context.Background(), "11111111-1111-1111-1111-111111111111", QueueCreate{Name: "x"})
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want APIError got %T %v", err, err)
	}
	if apiErr.Code != "iam.management_sa_not_enabled" || apiErr.Status != 403 {
		t.Fatalf("%+v", apiErr)
	}
}
