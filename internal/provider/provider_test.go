package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/homecloudlab/terraform-provider-homecloud/internal/client"
)

func TestProviderConstructs(t *testing.T) {
	_ = providerserver.NewProtocol6(New("test")())
	var _ map[string]func() (tfprotov6.ProviderServer, error)
}

func TestClientCreateQueueRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/access-key/whoami":
			_ = json.NewEncoder(w).Encode(map[string]string{"account_id": "11111111-1111-1111-1111-111111111111"})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/accounts/11111111-1111-1111-1111-111111111111/queues":
			if r.Header.Get("Idempotency-Key") == "" {
				t.Error("missing Idempotency-Key")
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
				"name":      "jobs",
				"status":    "active",
				"iam_arn":   "arn:homecloud:mq::123456789012:queue/jobs",
				"queue_url": "https://mq.example/jobs",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":{"code":"NOT_FOUND","message":"`+r.URL.Path+`"}}`)
		}
	}))
	defer srv.Close()
	c := &client.Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s"}
	id, err := c.ResolveAccountID(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	q, err := c.CreateQueue(context.Background(), id, client.QueueCreate{Name: "jobs"})
	if err != nil {
		t.Fatal(err)
	}
	if q.IamARN != "arn:homecloud:mq::123456789012:queue/jobs" {
		t.Fatalf("iam_arn %s", q.IamARN)
	}
}
