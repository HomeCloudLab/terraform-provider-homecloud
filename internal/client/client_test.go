package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestWaitUntilActiveImmediate(t *testing.T) {
	c := &Client{}
	err := c.WaitUntilActive(context.Background(), func() (string, error) {
		return "active", nil
	}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
}

func TestWaitUntilActiveFailed(t *testing.T) {
	c := &Client{}
	err := c.WaitUntilActive(context.Background(), func() (string, error) {
		return "failed", nil
	}, time.Second)
	if err == nil {
		t.Fatal("expected failed status error")
	}
}

func TestCreateIAMPolicyMarshalsDocumentObject(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"p1","name":"tf-mq","arn":"arn:homecloud:iam::1:policy/tf-mq","document":{"Version":"2026-07-24"}}`)
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s"}
	doc := json.RawMessage(`{"Version":"2026-07-24","Statement":[{"Effect":"Allow","Action":"mq:*","Resource":"*"}]}`)
	if _, err := c.CreateIAMPolicy(context.Background(), "acc", "tf-mq", "demo", doc); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatal(err)
	}
	if _, ok := parsed["document"].(map[string]any); !ok {
		t.Fatalf("document should be a JSON object, got %T %s", parsed["document"], gotBody)
	}
}

func TestCreateDatabaseSendsIdempotencyKeyAndParsesIamArn(t *testing.T) {
	accountID := "11111111-1111-1111-1111-111111111111"
	var gotPath, gotIdem, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotIdem = r.Header.Get("Idempotency-Key")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, `{"id":"db-1","name":"app-db","status":"provisioning","engine":"postgresql","instance_class":"micro","engine_version":"16","iam_arn":"arn:homecloud:mdb::123456789012:instance/app-db","connection":{"endpoint":"app-db-rw.svc","port":5432,"database":"app","username":"app"}}`)
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s"}
	got, err := c.CreateDatabase(context.Background(), accountID, DatabaseCreate{Name: "app-db", Engine: "postgresql", InstanceClass: "micro"})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/v1/accounts/"+accountID+"/databases" {
		t.Fatalf("request %s %s", gotMethod, gotPath)
	}
	wantKey := "terraform-mdb_instance-" + accountID + "-app-db"
	if gotIdem != wantKey {
		t.Fatalf("idempotency %s want %s", gotIdem, wantKey)
	}
	if got.IamARN != "arn:homecloud:mdb::123456789012:instance/app-db" || got.Name != "app-db" {
		t.Fatalf("%+v", got)
	}
	if got.Connection.Username != "app" {
		t.Fatalf("connection %+v", got.Connection)
	}
}

func TestGetDeleteDatabaseByName(t *testing.T) {
	accountID := "11111111-1111-1111-1111-111111111111"
	var gotGet, gotDel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			gotGet = r.URL.Path
			_, _ = io.WriteString(w, `{"id":"db-1","name":"app-db","status":"active","iam_arn":"arn:homecloud:mdb::1:instance/app-db"}`)
		case http.MethodDelete:
			gotDel = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("method %s", r.Method)
		}
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s"}
	got, err := c.GetDatabase(context.Background(), accountID, "app-db")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "active" {
		t.Fatalf("status %s", got.Status)
	}
	if err := c.DeleteDatabase(context.Background(), accountID, "app-db"); err != nil {
		t.Fatal(err)
	}
	want := "/api/v1/accounts/" + accountID + "/databases/app-db"
	if gotGet != want || gotDel != want {
		t.Fatalf("paths get=%s del=%s", gotGet, gotDel)
	}
}

func TestCreateGetDeleteCache(t *testing.T) {
	accountID := "11111111-1111-1111-1111-111111111111"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/accounts/"+accountID+"/caches":
			if r.Header.Get("Idempotency-Key") != "terraform-redis_instance-"+accountID+"-sessions" {
				t.Errorf("idempotency %s", r.Header.Get("Idempotency-Key"))
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":"c1","name":"sessions","status":"provisioning","iam_arn":"arn:homecloud:redis::1:cache/sessions","connection":{"credentials_secret":"cache-sessions-credentials","port":6379}}`)
		case r.Method == http.MethodGet:
			_, _ = io.WriteString(w, `{"id":"c1","name":"sessions","status":"active","iam_arn":"arn:homecloud:redis::1:cache/sessions"}`)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s"}
	created, err := c.CreateCache(context.Background(), accountID, CacheCreate{Name: "sessions", InstanceClass: "micro"})
	if err != nil {
		t.Fatal(err)
	}
	if created.IamARN != "arn:homecloud:redis::1:cache/sessions" {
		t.Fatalf("%+v", created)
	}
	got, err := c.GetCache(context.Background(), accountID, "sessions")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "active" {
		t.Fatalf("status %s", got.Status)
	}
	if err := c.DeleteCache(context.Background(), accountID, "sessions"); err != nil {
		t.Fatal(err)
	}
}

func TestDatabaseUserCRUD(t *testing.T) {
	accountID := "11111111-1111-1111-1111-111111111111"
	var gotCreateBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/users"):
			gotCreateBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"username":"ci","role":"readwrite","phase":"active"}`)
		case r.Method == http.MethodGet:
			_, _ = io.WriteString(w, `{"username":"ci","role":"readwrite","phase":"active"}`)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/rotate"):
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"username":"ci","role":"readwrite","phase":"active"}`)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s"}
	created, err := c.CreateDatabaseUser(context.Background(), accountID, "app-db", "ci", "hunter22", "readwrite", "")
	if err != nil {
		t.Fatal(err)
	}
	if created.Username != "ci" {
		t.Fatalf("%+v", created)
	}
	var parsed map[string]string
	if err := json.Unmarshal(gotCreateBody, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["password"] != "hunter22" || parsed["username"] != "ci" {
		t.Fatalf("%s", gotCreateBody)
	}
	got, err := c.GetDatabaseUser(context.Background(), accountID, "app-db", "ci")
	if err != nil {
		t.Fatal(err)
	}
	if got.Username != "ci" {
		t.Fatalf("%+v", got)
	}
	if _, err := c.RotateDatabaseUser(context.Background(), accountID, "app-db", "ci", "hunter23"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteDatabaseUser(context.Background(), accountID, "app-db", "ci"); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteDatabaseNotFoundIsOk(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":{"code":"not_found","message":"gone"}}`)
	}))
	defer srv.Close()
	c := &Client{Endpoint: srv.URL, AccessKeyID: "k", Secret: "s"}
	if err := c.DeleteDatabase(context.Background(), "acc", "missing"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteCache(context.Background(), "acc", "missing"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteDatabaseUser(context.Background(), "acc", "db", "missing"); err != nil {
		t.Fatal(err)
	}
}
