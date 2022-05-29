package nodes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
)

func (fn *FullNode) Printable() map[string]interface{} {
	return map[string]interface{}{
		"id":  fn.Identifier,
		"t":   fn.Type,
		"nm":  fn.Name,
		"ln":  fn.Lastname,
		"im":  fn.Image.Identifier,
		"msg": fn.Messages,
		"adm": fn.Admins,
		"chl": fn.Childs,
		"prt": fn.Parents,
	}
}

func TestGetNodesHTTP(t *testing.T) {

	validNodes := []byte("andrew anthony baker david don helene jamie jeff jeremy josh kendie kon ron roul wilson")

	req1 := httptest.NewRequest("POST", "/", bytes.NewReader(validNodes))

	rr1 := httptest.NewRecorder()

	GetNodes(rr1, req1)

	var r1 []FullNode

	json.NewDecoder(rr1.Body).Decode(&r1)

	for _, v := range r1 {
		fmt.Println(v.Printable())
	}

}

func BenchmarkGetNodesHTTP(b *testing.B) {

	validNodes := []byte("andrew anthony baker david don helene jamie jeff jeremy josh kendie kon ron roul wilson")

	req1 := httptest.NewRequest("POST", "/", bytes.NewReader(validNodes))

	rr1 := httptest.NewRecorder()

	GetNodes(rr1, req1)

	var r1 []FullNode

	json.NewDecoder(rr1.Body).Decode(&r1)

}
