package messages

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMessageRequestHTTP(t *testing.T) {

	jeff := MessageRequest{
		WithUpload:  false,
		IsGroup:     false,
		IsHyperchat: false,
		Targets:     []string{"joerogan"},
		Message: Down4Message{
			SenderID:        "fksdl4",
			SenderName:      "Scott",
			SenderLastName:  "Harrisson",
			SenderThumbnail: "lol",
			Text:            "Hello my good friend joe rogan",
			Timestamp:       time.Now().Unix(),
			IsChat:          true,
		},
	}

	data, err := json.Marshal(jeff)
	if err != nil {
		t.Errorf("error marshaling data: %v\n", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(data))
	rsp := httptest.NewRecorder()

	HandleMessageRequest(rsp, req)

	fmt.Printf("response status: %v\n", rsp.Code)
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
