package sigv1

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type vector struct {
	Name          string `json:"name"`
	Method        string `json:"method"`
	Path          string `json:"path"`
	Timestamp     string `json:"timestamp"`
	AccountID     string `json:"account_id"`
	Secret        string `json:"secret"`
	StringToSign  string `json:"string_to_sign"`
	Signature     string `json:"signature"`
	Whoami        bool   `json:"whoami"`
}

func TestFrozenVectorsMatchPython(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "sigv1_vectors.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read vectors: %v", err)
	}
	var cases []vector
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse vectors: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("empty vector file")
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			got := BuildStringToSign(c.Method, c.Path, c.Timestamp, c.AccountID)
			if got != c.StringToSign {
				t.Fatalf("string to sign\n got: %q\nwant: %q", got, c.StringToSign)
			}
			sig := ComputeSignature(c.Secret, got)
			if sig != c.Signature {
				t.Fatalf("signature\n got: %s\nwant: %s", sig, c.Signature)
			}
			if c.Whoami && c.AccountID != WhoamiAccountID {
				t.Fatalf("whoami account id want %q got %q", WhoamiAccountID, c.AccountID)
			}
		})
	}
}

func TestFormatTimestampUTCSeconds(t *testing.T) {
	ts := time.Date(2026, 8, 17, 12, 0, 0, 123456789, time.UTC)
	if got := FormatTimestamp(ts); got != "2026-08-17T12:00:00Z" {
		t.Fatalf("got %s", got)
	}
}
