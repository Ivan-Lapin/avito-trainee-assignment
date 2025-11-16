package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

const base = "http://localhost:8095/api"

func TestPRLifecycle(t *testing.T) {
	// create
	cr := map[string]any{
		"id":        "00000000-0000-0000-0000-0000000000aa",
		"title":     "t",
		"author_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"team_name": "core",
	}
	b, _ := json.Marshal(cr)
	res, err := http.Post(base+"/pullRequest/create", "application/json", bytes.NewReader(b))
	if err != nil || (res.StatusCode != 201 && res.StatusCode != 409) {
		t.Fatalf("create failed: %v code=%d", err, res.StatusCode)
	}

	// merge (идемпотентность)
	m := map[string]any{"id": "00000000-0000-0000-0000-0000000000aa"}
	mb, _ := json.Marshal(m)
	req, _ := http.NewRequest("POST", base+"/pullRequest/merge", bytes.NewReader(mb))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "test-key")
	res, err = http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("merge failed: %v code=%d", err, res.StatusCode)
	}
}
