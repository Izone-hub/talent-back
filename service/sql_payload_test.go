package service

import (
	"encoding/json"
	"testing"
)

func mustJSONEqual(t *testing.T, got, want string) {
	t.Helper()
	var g, w interface{}
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		t.Fatalf("got is not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("want is not valid JSON: %v", err)
	}
	gb, _ := json.Marshal(g)
	wb, _ := json.Marshal(w)
	if string(gb) != string(wb) {
		t.Fatalf("payload mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestBuildSQLPayloadObjectForm(t *testing.T) {
	raw := []byte(`{
		"database": {"schema": "CREATE TABLE t (id INT);", "seed": "INSERT INTO t VALUES (1);"},
		"tests": [
			{"name": "a", "verify": "SELECT * FROM t", "expected_rows": [[1]]},
			{"name": "hidden one", "is_hidden": true, "verify": "x", "expected_rows": []},
			{"query": "SELECT count(*) FROM t", "expected": 1}
		]
	}`)
	payload, files, err := BuildSQLPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if files["schema.sql"] == "" || files["seed.sql"] == "" {
		t.Fatalf("expected schema/seed files, got %v", files)
	}
	mustJSONEqual(t, payload, `[
		{"name":"a","verify":"SELECT * FROM t","expected_rows":[[1]]},
		{"query":"SELECT count(*) FROM t","expected":1}
	]`)
}

func TestBuildSQLPayloadArrayForm(t *testing.T) {
	raw := []byte(`[{"query": "SELECT 1", "expected": [[1]]}, {"is_hidden": true}]`)
	payload, files, err := BuildSQLPayload(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files for array form, got %v", files)
	}
	mustJSONEqual(t, payload, `[{"query":"SELECT 1","expected":[[1]]}]`)
}

func TestBuildSQLPayloadEmpty(t *testing.T) {
	payload, files, err := BuildSQLPayload(nil)
	if err != nil || payload != "[]" || len(files) != 0 {
		t.Fatalf("empty handling failed: %q %v %v", payload, files, err)
	}
}
