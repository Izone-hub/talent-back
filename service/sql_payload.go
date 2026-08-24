package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

type sqlDatabaseSpec struct {
	Schema string `json:"schema"`
	Seed   string `json:"seed"`
}

type sqlTestCasesDoc struct {
	Database *sqlDatabaseSpec        `json:"database,omitempty"`
	Tests    []map[string]interface{} `json:"tests,omitempty"`
}

// BuildSQLPayload normalizes SQL question test cases into the flat array
// consumed by the SQL test harness, plus work-dir files that carry the
// imported database (schema.sql / seed.sql).
//
// Accepted test_cases shapes:
//
//  1. Object with an imported database:
//     {"database": {"schema": "CREATE TABLE ...", "seed": "INSERT ..."},
//      "tests": [{"name": "...", "verify": "SELECT ...", "expected_rows": [...]}]}
//
//  2. Plain array (no imported database; tests may carry per-test "setup"):
//     [{"query": "SELECT 1", "expected": [[1]]}]
//
// Hidden tests (is_hidden / hidden) are filtered out.
func BuildSQLPayload(raw json.RawMessage) (string, map[string]string, error) {
	files := map[string]string{}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" {
		return "[]", files, nil
	}

	var out []byte

	if trimmed[0] == '{' {
		var doc sqlTestCasesDoc
		if err := json.Unmarshal(raw, &doc); err != nil {
			return "", nil, fmt.Errorf("invalid SQL test cases document: %w", err)
		}
		if doc.Database != nil {
			if s := strings.TrimSpace(doc.Database.Schema); s != "" {
				files["schema.sql"] = s + "\n"
			}
			if s := strings.TrimSpace(doc.Database.Seed); s != "" {
				files["seed.sql"] = s + "\n"
			}
		}
		tests := make([]map[string]interface{}, 0, len(doc.Tests))
		for _, t := range doc.Tests {
			if hidden, _ := t["is_hidden"].(bool); hidden {
				continue
			}
			if hidden, _ := t["hidden"].(bool); hidden {
				continue
			}
			tests = append(tests, t)
		}
		encoded, err := json.Marshal(tests)
		if err != nil {
			return "", nil, fmt.Errorf("failed to encode SQL tests: %w", err)
		}
		out = encoded
	} else {
		var tests []map[string]interface{}
		if err := json.Unmarshal(raw, &tests); err != nil {
			return "", nil, fmt.Errorf("invalid SQL test cases: %w", err)
		}
		filtered := make([]map[string]interface{}, 0, len(tests))
		for _, t := range tests {
			if hidden, _ := t["is_hidden"].(bool); hidden {
				continue
			}
			if hidden, _ := t["hidden"].(bool); hidden {
				continue
			}
			filtered = append(filtered, t)
		}
		encoded, err := json.Marshal(filtered)
		if err != nil {
			return "", nil, fmt.Errorf("failed to encode SQL tests: %w", err)
		}
		out = encoded
	}

	return string(out), files, nil
}
