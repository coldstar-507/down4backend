package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHandleChatRequest(t *testing.T) {
	cr := ChatRequest{
		Targets: []string{"ronaldo"},
		Message: Down4Message{
			Type:      "chat",
			MessageID: "thfdsafsdID",
			SenderID:  "scott",
			Text:      "Hello Satoshi Nakomoto",
			Timestamp: time.Now().Unix(),
		},
	}

	marsh, err := json.Marshal(cr)
	if err != nil {
		t.Errorf("error marshalling chat request: %v\n", err)
	}
	reader := bytes.NewReader(marsh)

	req := httptest.NewRequest("POST", "/", reader)
	rsp := httptest.NewRecorder()

	HandleChatRequest(rsp, req)

	if rsp.Code != 200 {
		t.Errorf("error handleing chat request")
	}
}

func TestHandlePingRequest(t *testing.T) {
	pr := PingRequest{
		Targets:  []string{"scott"},
		Text:     "Hello scott",
		SenderID: "down4",
	}

	marsh, err := json.Marshal(pr)
	if err != nil {
		log.Printf("error marshalling ping: %v", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(marsh))
	rsp := httptest.NewRecorder()

	HandlePingRequest(rsp, req)

	if rsp.Code != 200 {
		t.Errorf("error handling ping request")
	}
}

func TestHandleHyperchatRequest(t *testing.T) {
	cr := HyperchatRequest{
		Targets:    []string{"satoshi"},
		WordPairs:  []string{"pink bow", "disk job"},
		WithUpload: false,
		Message: Down4Message{
			Root:      "gjkdflj423kljfdsakl34j",
			MessageID: "this is a message ID",
			SenderID:  "scott",
			Text:      "Hello satoshi nakomoto",
			Timestamp: time.Now().Unix(),
		},
	}

	marsh, err := json.Marshal(cr)
	if err != nil {
		t.Errorf("error marshalling chat request: %v\n", err)
	}
	reader := bytes.NewReader(marsh)

	req := httptest.NewRequest("POST", "/", reader)
	rsp := httptest.NewRecorder()

	HandleHyperchatRequest(rsp, req)

	if rsp.Code != 200 {
		t.Errorf("error handleing hyperchat request")
	}
}

func TestUsernameValidityHttp(t *testing.T) {

	var scott, andrew []byte = []byte("scott"), []byte("andrew")
	req1 := httptest.NewRequest("POST", "/", bytes.NewReader(scott))
	req2 := httptest.NewRequest("POST", "/", bytes.NewReader(andrew))

	rr1 := httptest.NewRecorder()
	rr2 := httptest.NewRecorder()

	IsValidUsername(rr1, req1)
	IsValidUsername(rr2, req2)

	fmt.Printf("scott: %v\nandrew: %v\n", rr1.Code, rr2.Code)
	if rr1.Code != 500 || rr2.Code != 200 {
		t.Errorf("scott code should be 500, is: %v\n andrew code should be 200, is: %v\n", rr1.Code, rr2.Code)
	}
}

func TestUserInitializationHttp(t *testing.T) {
	im, err := os.ReadFile("/home/scott/Pictures/basedretard.png")
	if err != nil {
		t.Errorf("error reading file image for user init test: %v\n", err)
	}

	jeff := InitUserInfo{
		Neuter:     "IAmANeuteMotherfuckers",
		Secret:     "IAmASecret",
		Token:      "IAmAToken",
		Name:       "Craig",
		Lastname:   "Wright",
		Identifier: "satoshi",
		Image: Down4Media{
			Identifier: "basedretard",
			Data:       im,
			Metadata: map[string]string{
				"o":   "helene",
				"ts":  "90409328",
				"trv": "false",
				"ptv": "false",
				"shr": "true",
				"vid": "false",
				"pto": "false",
			},
		},
	}

	marshalled, err := json.Marshal(jeff)
	if err != nil {
		t.Errorf("error marshalling info: %v\n", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(marshalled))
	rr := httptest.NewRecorder()

	InitUser(rr, req)
	if rr.Code == 200 {
		fmt.Println("sucessfully created user")
	} else {
		fmt.Println("user already exists, can't create user")
	}

	fmt.Println("Must view the results manually, this test is always PASS")
}

func TestMnemonicHttp(t *testing.T) {

	req := httptest.NewRequest("GET", "/", nil)

	rr := httptest.NewRecorder()
	GenerateMnemonic(rr, req)

	byteMnemonic, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Errorf("could not read responsebody: %v\n", err)
	}

	words := strings.Split(string(byteMnemonic), " ")

	if len(words) != 12 {
		t.Errorf("invalide mnemonic, not 12 words: %v\n", words)
	}

	fmt.Printf("Mnemonic: %v\n", string(byteMnemonic))
}
