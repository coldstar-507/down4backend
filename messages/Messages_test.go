package messages

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestChatWithMediaUpload(t *testing.T) {

	im, err := os.ReadFile("C:/Users/coton/Desktop/hashirama.png")
	if err != nil {
		t.Errorf("error reading image file for thumbnail: %v\n", err)
	}
	tn := base64.StdEncoding.EncodeToString(im)
	if im, err = os.ReadFile("C:/Users/coton/Pictures/Chan/1638406306586.jpg"); err != nil {
		t.Errorf("error reading image file for attachement: %v\n", err)

	}
	d4media := Down4Media{
		Identifier: "litte-kid",
		Data:       im,
		Metadata: map[string]string{
			"o":   "helene",
			"ts":  "312321443",
			"pto": "false",
			"shr": "true",
			"ptv": "false",
			"vid": "false",
			"trv": "false",
		},
	}

	chatWithMediaUpload := MessageRequest{
		WithUpload:  true,
		IsGroup:     false,
		IsHyperchat: false,
		Targets:     []string{"bobo"},
		Message: Down4Message{
			MessageID:       "fdslkj4l32kj",
			Root:            "bobo",
			SenderID:        "bobo",
			SenderName:      "Chicken",
			SenderLastName:  "BBQ",
			SenderThumbnail: tn,
			Text:            "Satoshi",
			Media:           d4media,
			Timestamp:       time.Now().Unix(),
			IsChat:          true,
		},
	}

	data, err := json.Marshal(chatWithMediaUpload)
	if err != nil {
		t.Errorf("error marshaling data: %v\n", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(data))
	rsp := httptest.NewRecorder()

	HandleMessageRequest(rsp, req)

	fmt.Printf("response status: %v\n", rsp.Code)
}

func TestChatWithoutMediaUpload(t *testing.T) {

	im, err := os.ReadFile("C:/Users/coton/Desktop/hashirama.png")
	if err != nil {
		t.Errorf("error reading image file for thumbnail: %v\n", err)
	}
	tn := base64.StdEncoding.EncodeToString(im)

	d4mediaNoUpload := Down4Media{
		Identifier: "nigga",
	}

	chatWithoutMediaUpload := MessageRequest{
		WithUpload:  false,
		IsGroup:     false,
		IsHyperchat: false,
		Targets:     []string{"bobo"},
		Message: Down4Message{
			MessageID:       "fdsfgdsf22323",
			Root:            "traphouse",
			SenderID:        "bobo",
			SenderName:      "Bobo",
			SenderLastName:  "The bear",
			SenderThumbnail: tn,
			Text:            "What's up dude!",
			Timestamp:       time.Now().Unix(),
			IsChat:          true,
			Media:           d4mediaNoUpload,
		},
	}

	data, err := json.Marshal(chatWithoutMediaUpload)
	if err != nil {
		t.Errorf("error marshaling data: %v\n", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(data))
	rsp := httptest.NewRecorder()

	HandleMessageRequest(rsp, req)

	fmt.Printf("response status: %v\n", rsp.Code)
}

func TestGroupchatWithoutMedia(t *testing.T) {

	im, err := os.ReadFile("C:/Users/coton/Desktop/hashirama.png")
	if err != nil {
		t.Errorf("error reading image file for thumbnail: %v\n", err)
	}
	tn := base64.StdEncoding.EncodeToString(im)
	if im, err = os.ReadFile("C:/Users/coton/Pictures/Chan/howmuch.jpg"); err != nil {
		t.Errorf("error reading image file for attachement: %v\n", err)

	}

	pseudoNode := PseudoNode{
		Identifier: "traphouse",
		Name:       "TrapHouse",
		Image: Down4Media{
			Identifier: "fjdkslfnmfndsmafjkdlsaf",
			Data:       im,
			Metadata: map[string]string{
				"o":   "jeff",
				"ts":  "3123231443",
				"pto": "false",
				"shr": "true",
				"ptv": "false",
				"vid": "false",
				"trv": "false",
			},
		},
	}

	normalChat := MessageRequest{
		WithUpload:  false,
		IsGroup:     true,
		IsHyperchat: false,
		Targets:     []string{"bobo"},
		GroupNode:   pseudoNode,
		Message: Down4Message{
			MessageID:       "j2k13lk231kj321jksklad",
			Root:            "traphouse",
			SenderID:        "helene",
			SenderName:      "Helene",
			SenderLastName:  "Dufour",
			SenderThumbnail: tn,
			Text:            "This is a cloudy day which is perfect for coding end finishing the messaging part of down4",
			Timestamp:       time.Now().Unix(),
			IsChat:          true,
		},
	}

	data, err := json.Marshal(normalChat)
	if err != nil {
		t.Errorf("error marshaling data: %v\n", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(data))
	rsp := httptest.NewRecorder()

	HandleMessageRequest(rsp, req)

	fmt.Printf("response status: %v\n", rsp.Code)
}

func TestHyperchatWithoutMedia(t *testing.T) {

	im, err := os.ReadFile("C:/Users/coton/Desktop/hashirama.png")
	if err != nil {
		t.Errorf("error reading image file for thumbnail: %v\n", err)
	}
	tn := base64.StdEncoding.EncodeToString(im)
	if im, err = os.ReadFile("C:/Users/coton/Pictures/Chan/1630109700356.png"); err != nil {
		t.Errorf("error reading image file for attachement: %v\n", err)

	}
	var im2 []byte
	if im2, err = os.ReadFile("C:/Users/coton/Pictures/Chan/2d.png"); err != nil {
		t.Errorf("error reading image file for attachement: %v\n", err)

	}
	d4media := Down4Media{
		Identifier: "lol",
		Data:       im2,
		Metadata: map[string]string{
			"o":   "scott",
			"ts":  "312321443",
			"pto": "false",
			"shr": "true",
			"ptv": "false",
			"vid": "false",
			"trv": "false",
		},
	}

	pseudoNode := PseudoNode{
		Identifier: "hyper",
		Name:       "FirstHyperchat",
		Image: Down4Media{
			Identifier: "fjdkslfnmfndsmafjkdlsaf",
			Data:       im,
			Metadata: map[string]string{
				"o":   "jeff",
				"ts":  "3123231443",
				"pto": "false",
				"shr": "true",
				"ptv": "false",
				"vid": "false",
				"trv": "false",
			},
		},
	}

	normalChat := MessageRequest{
		WithUpload:  true,
		IsGroup:     false,
		IsHyperchat: true,
		Targets:     []string{"bobo"},
		GroupNode:   pseudoNode,
		Message: Down4Message{
			MessageID:       "j2k13lkff231kj321jksklad",
			Root:            "hyperchat",
			SenderID:        "what",
			SenderName:      "Everactually",
			SenderLastName:  "lol",
			SenderThumbnail: tn,
			Media:           d4media,
			Text:            "This is a cloudy day which is perfect for coding end finishing the messaging part of down4",
			Timestamp:       time.Now().Unix(),
			IsChat:          true,
		},
	}

	data, err := json.Marshal(normalChat)
	if err != nil {
		t.Errorf("error marshaling data: %v\n", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(data))
	rsp := httptest.NewRecorder()

	HandleMessageRequest(rsp, req)

	fmt.Printf("response status: %v\n", rsp.Code)
}

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
