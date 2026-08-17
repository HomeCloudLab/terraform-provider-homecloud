package provider

import (
	"os"
	"testing"
)

func TestAccSkipUnlessConfigured(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 with HC_ACCESS_KEY_ID / HC_SECRET_ACCESS_KEY / HC_APEX to run acceptance tests")
	}
	if os.Getenv("HC_ACCESS_KEY_ID") == "" || os.Getenv("HC_SECRET_ACCESS_KEY") == "" {
		t.Fatal("TF_ACC=1 requires HC_ACCESS_KEY_ID and HC_SECRET_ACCESS_KEY")
	}
}
