package getnodes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestGetNodes(t *testing.T) {

	uniques := "golden golden golden golden golden"

	body := bytes.NewReader([]byte(uniques))

	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	GetNodes(w, r)

	var rsp []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &rsp); err != nil {
		fmt.Printf("error unmarshalling res: %v\n", err)
	}

	for i, v := range rsp {
		fmt.Printf("\nnode#%d: %v\n", i, v)
	}
}
