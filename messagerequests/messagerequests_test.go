package messagerequests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestMessageRequests(t *testing.T) {

	tokens := []string{"c6ClRz1sTAWJol-qlvLejz:APA91bG08Z93oJgMoPl2Q09plChSOy6U110OkGPGw3HSlG2vsMs-IBJkx_U6p4DPUj_JTqGOHlLF_sSEj_Fw6xTKCXLAtdkYnboQHCKE_Er8wQj3keMg5gxTXYKK3XU36PTC146u8Ixk"}

	mr := &messageRequest{
		Tokens: tokens,
		Header: "Jeff",
		Body:   "Sup my beautiful big brother",
	}

	js, _ := json.Marshal(mr)

	r := httptest.NewRequest("POST", "/", bytes.NewReader(js))
	w := httptest.NewRecorder()

	HandleMessageRequest(w, r)

}
