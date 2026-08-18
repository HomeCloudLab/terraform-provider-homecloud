package provider

import "testing"

func TestJSONEqualIgnoresKeyOrder(t *testing.T) {
	a := `{"Version":"2026-07-24","Statement":[{"Effect":"Allow"}]}`
	b := `{"Statement":[{"Effect":"Allow"}],"Version":"2026-07-24"}`
	if !jsonEqual(a, b) {
		t.Fatal("expected semantically equal JSON")
	}
	if jsonEqual(a, `{"Version":"nope"}`) {
		t.Fatal("expected different JSON")
	}
}

func TestCompactJSON(t *testing.T) {
	got := compactJSON([]byte(`{ "a" : 1 }`))
	if got != `{"a":1}` {
		t.Fatalf("got %s", got)
	}
}
