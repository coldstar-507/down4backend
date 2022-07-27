package messages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
)

func TestHandlePingRequest(t *testing.T) {
	ping := PingRequest{
		Targets: []string{"name"},
		Message: Down4Message{
			SenderID:       "niggler",
			SenderName:     "Niggler",
			SenderLastName: "Bato",
			Text:           "Hello my beautiful name!",
		},
	}

	marsh, _ := json.Marshal(ping)
	data := bytes.NewReader(marsh)

	req := httptest.NewRequest("POST", "/", data)
	rsp := httptest.NewRecorder()

	HandlePingRequest(rsp, req)

	if rsp.Code != 200 {
		t.Errorf("error in ping request test")
	}

}

// func TestHandleSnipRequest(t *testing.T) {
// 	snip := SnipRequest{
// 		Targets: []string{"name"},
// 		Message: Down4Message{
// 			SenderID:       "niggler",
// 			SenderName:     "Niggler",
// 			SenderLastName: "Bato",

// 		},
// 	}
// }

func TestGetMessageMediaHTTP(t *testing.T) {
	mediaName := "1628475211371.jpg"
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(mediaName))
	rsp := httptest.NewRecorder()

	GetMessageMedia(rsp, req)

	var d4media Down4Media
	if rsp.Code != 200 {
		t.Errorf("error, could not get this media: %s\n", mediaName)
	} else {
		json.NewDecoder(rsp.Body).Decode(&d4media)
		fmt.Println(d4media)
	}
}

func TestMediaExistsHTTP(t *testing.T) {
	existingMediaID, nonExistingMediaID := []byte("07bf2e3d03c139e7de63db49d8084929f1c5c646"), []byte("fds32")
	req1 := httptest.NewRequest("POST", "/", bytes.NewReader(existingMediaID))
	req2 := httptest.NewRequest("POST", "/", bytes.NewReader(nonExistingMediaID))
	rsp1 := httptest.NewRecorder()
	rsp2 := httptest.NewRecorder()

	GetMediaMetadata(rsp1, req1)
	GetMediaMetadata(rsp2, req2)

	if rsp1.Code != 200 {
		t.Errorf("error, media 07bf2e3d03c139e7de63db49d8084929f1c5c646 does exist!")
	}
	if rsp2.Code != 500 {
		t.Errorf("error, media fds32 does not exist!")
	}
}
