package getnodes

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestGetNodes(t *testing.T) {
	uniques := "hashirama mafia scammer"

	body := bytes.NewReader([]byte(uniques))

	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	GetNodes(w, r)

	var rsp []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("error unmarshalling res: %v\n", err)
	}

	t.Logf("TestGetNodes: found %d nodes\n", len(rsp))

	for i, v := range rsp {
		t.Logf("#%d -- node: %v\n", i, v)
	}
}
