package backend

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math/rand"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func randomBuffer(len int) []byte {
	buf := make([]byte, len)
	rand.Seed(time.Now().UnixNano())
	rand.Read(buf)
	return buf
}

func sha256Hex(data []byte) string {
	h := sha256.New()
	h.Write(data)
	hashed := h.Sum(make([]byte, 0))
	return hex.EncodeToString(hashed)
}

func sha1Hex(data []byte) string {
	h := sha1.New()
	h.Write(data)
	hashed := h.Sum(make([]byte, 0))
	return hex.EncodeToString(hashed)
}

func TestTimestamp(t *testing.T) {
	ts := unixMilliseconds()

	t.Logf("timestamp unix milliseconds: %v\n", ts)
}

func TestHandlePaymentRequest(t *testing.T) {

	pr := PaymentRequest{
		Sender:     "scott",
		Targets:    []string{"alice"},
		Payment:    []byte("eyJpZCI6IjhlNTliYjI2Nzg4OGZlYjIyYzEyOGRjM2M3YzRhM2U4NzgwODhhOGM4ZmJlYmM2MmUyMWQwODVkMjM2ZGZlZmIiLCJ0eCI6W3sibWsiOm51bGwsInZuIjoxLCJubCI6MCwiaWQiOiJkMTFhZmRmODg5OTdmOTUxZjRmNmZkYjIxOGE3YTEzNWU5Zjk5YTlkNGQzYjcxMzQ4NTdiOGI3ODg5OTY1MTc5IiwiY2YiOjgyLCJ0aSI6W3sidXR4byI6eyJyYyI6bnVsbCwic3QiOm51bGwsIm9pIjoxLCJpZCI6IjI0MjhlYjE1OTM3N2JmYzY0MWIyYzhkZjA2MzI0OGJkMGI1NTI2YjQyM2U2YzFiZWI0M2UwYjc4NTc0NjZkZGQiLCJzIjo1MDAwMCwic2MiOiI3NmE5MTRjMDhmOTgwYTFhNjZmMWQ1ZDZjOWIwOWU3NThmYWM1NDI1MWQ5MDRhODhhYyJ9LCJzcCI6bnVsbCwic24iOjQyOTQ5NjcyOTUsInNsIjoxMDcsInNjIjoiNDgzMDQ1MDIyMTAwZjljYmRlOGY3ZGNlN2Y3NTlmM2QxZjhhZDJkNGEyN2NhMjUyMTllOWU3NjRkY2MxYTdlNDBmODQzOWNjM2VlMjAyMjAzNjM1NGMzZmQzYjQ2ZDYwMjQzYWUxM2EwYjhlNzM4ZDhmOGQ0MjBmYTc5NGYwZDk2OTlmMmMzYzU5NmVmODQ1NDEyMTAyYWNlMDZiMWUwMmVkNjg2ZjlmMzEyMTk4YWVhODEyNTQ3OTllOTkxYjJkZGNlYTE2NzZhYWE0M2FlOWZjYWM1MCIsImRwIjpudWxsfV0sInRvIjpbeyJyYyI6bnVsbCwic3QiOlswLDAsMCwwLDk3LDEwOCwxMDUsOTksMTAxXSwib2kiOjAsImlkIjoiZDExYWZkZjg4OTk3Zjk1MWY0ZjZmZGIyMThhN2ExMzVlOWY5OWE5ZDRkM2I3MTM0ODU3YjhiNzg4OTk2NTE3OSIsInMiOjU0LCJzYyI6Ijc2YTkxNDVkOWRjOTVmYTNjNWQ2ZGEzMjMzMThlMzlkODU1MjNjNjQ3Y2RiOWU4OGFjIn0seyJyYyI6ImFsaWNlIiwic3QiOlswLDAsMCwwXSwib2kiOjEsImlkIjoiZDExYWZkZjg4OTk3Zjk1MWY0ZjZmZGIyMThhN2ExMzVlOWY5OWE5ZDRkM2I3MTM0ODU3YjhiNzg4OTk2NTE3OSIsInMiOjQ5OTM0LCJzYyI6Ijc2YTkxNDVkOWRjOTVmYTNjNWQ2ZGEzMjMzMThlMzlkODU1MjNjNjQ3Y2RiOWU4OGFjIn1dfSx7Im1rIjpudWxsLCJ2biI6MSwibmwiOjAsImlkIjoiOTNkZTcyYmIxM2VkMWRjODBlZGRjZjAxYWM2MjFkNzdjZTNiNDQ4NWFiY2M1YzRmYWYzYmFiMDZkMDhjODRmYiIsImNmIjowLCJ0aSI6W3sidXR4byI6eyJyYyI6ImFsaWNlIiwic3QiOlswLDAsMCwwXSwib2kiOjEsImlkIjoiZDExYWZkZjg4OTk3Zjk1MWY0ZjZmZGIyMThhN2ExMzVlOWY5OWE5ZDRkM2I3MTM0ODU3YjhiNzg4OTk2NTE3OSIsInMiOjQ5OTM0LCJzYyI6Ijc2YTkxNDVkOWRjOTVmYTNjNWQ2ZGEzMjMzMThlMzlkODU1MjNjNjQ3Y2RiOWU4OGFjIn0sInNwIjoiYWxpY2UiLCJzbiI6MCwic2wiOjEwNywic2MiOiI0ODMwNDUwMjIxMDBhNWIxNzViNjA2NjlhOTI1YzcxY2FlYzQ3ZWZjNmJkZDJmZjQxNDgyZTNiZWU3ZWRkZjc4ZGRkOWZkMTUzMWIwMDIyMDA3MDFlYTJhYWE0YzA4MGUzYWUzZmY4ZTFkMzFlNDlkODNjZDBmZTJlODhlMmI4Zjg2MWEwMDc3NWVlMmMxNTY0MTIxMDJiMjM1ZTQ4MjRlY2E3NjU3ZWI2NmJkNmQxMjM0NmQ2NGJkM2E5ZTE1ODgzOWViN2E4OWU4YjJhYmUyMTRmOTQ3IiwiZHAiOiJkMTFhZmRmODg5OTdmOTUxZjRmNmZkYjIxOGE3YTEzNWU5Zjk5YTlkNGQzYjcxMzQ4NTdiOGI3ODg5OTY1MTc5In1dLCJ0byI6W3sicmMiOm51bGwsInN0IjpbMCwwLDAsMSw5NywxMDgsMTA1LDk5LDEwMV0sIm9pIjowLCJpZCI6IjkzZGU3MmJiMTNlZDFkYzgwZWRkY2YwMWFjNjIxZDc3Y2UzYjQ0ODVhYmNjNWM0ZmFmM2JhYjA2ZDA4Yzg0ZmIiLCJzIjo4LCJzYyI6Ijc2YTkxNGUzOWQxOTgzZWNiYjcxMjA4OWExMjU2NzRmMDVlYTJjODc2YzY5OTQ4OGFjIn0seyJyYyI6ImhlbGVuZSIsInN0IjpbMCwwLDAsMSwxMDQsMTAxLDEwOCwxMDEsMTEwLDEwMV0sIm9pIjoxLCJpZCI6IjkzZGU3MmJiMTNlZDFkYzgwZWRkY2YwMWFjNjIxZDc3Y2UzYjQ0ODVhYmNjNWM0ZmFmM2JhYjA2ZDA4Yzg0ZmIiLCJzIjo3MDAsInNjIjoiNzZhOTE0NDZlZThlZjY4N2I3MzdiZWJlY2M5NGRlMGJkM2MwN2E0ODM5NmE1NDg4YWMifSx7InJjIjoiYWxpY2UiLCJzdCI6WzAsMCwwLDEsOTcsMTA4LDEwNSw5OSwxMDFdLCJvaSI6MiwiaWQiOiI5M2RlNzJiYjEzZWQxZGM4MGVkZGNmMDFhYzYyMWQ3N2NlM2I0NDg1YWJjYzVjNGZhZjNiYWIwNmQwOGM4NGZiIiwicyI6NzAwLCJzYyI6Ijc2YTkxNDU0MDk2Y2ExZjYzOTZkODFlZWRkZThhN2M0M2UyMDFlOTUzYTE5NmY4OGFjIn0seyJyYyI6ImFsaWNlIiwic3QiOlswLDAsMCwxXSwib2kiOjMsImlkIjoiOTNkZTcyYmIxM2VkMWRjODBlZGRjZjAxYWM2MjFkNzdjZTNiNDQ4NWFiY2M1YzRmYWYzYmFiMDZkMDhjODRmYiIsInMiOjQ4NTExLCJzYyI6Ijc2YTkxNDc4NTFjYjEwZjIyMWJhMmVmYzk5NmQzNDA3YmJlNGU1ZGNhNTI5NzQ4OGFjIn1dfSx7Im1rIjpudWxsLCJ2biI6MSwibmwiOjAsImlkIjoiOGVkMzJlMWM2NjI1MzVmMWM1NDM3ZWM5MmFlYzQ5MjIxNjY2NzM1ZmY5MmQ0Njg4NDJlODZmMDE1ZjI0OGU5OSIsImNmIjowLCJ0aSI6W3sidXR4byI6eyJyYyI6ImFsaWNlIiwic3QiOlswLDAsMCwxLDk3LDEwOCwxMDUsOTksMTAxXSwib2kiOjIsImlkIjoiOTNkZTcyYmIxM2VkMWRjODBlZGRjZjAxYWM2MjFkNzdjZTNiNDQ4NWFiY2M1YzRmYWYzYmFiMDZkMDhjODRmYiIsInMiOjcwMCwic2MiOiI3NmE5MTQ1NDA5NmNhMWY2Mzk2ZDgxZWVkZGU4YTdjNDNlMjAxZTk1M2ExOTZmODhhYyJ9LCJzcCI6ImFsaWNlIiwic24iOjAsInNsIjoxMDYsInNjIjoiNDczMDQ0MDIyMDM2NDYxODBhOTE0MDNmMzlhOWZjNGRhMzY1NTQyODFhZDc3MzFmODFjMmEzYzdmMDc4ZDZhNWRiNGQ0ODYwNWYwMjIwMThkMjAxODJjNDYwOTk2YWE2NWQzYTEzNDdhYzg5NzhhYzQwMmIwZmRjODU2ZWM1MTE1MzM1YTAxZjRlMTZjNjQxMjEwMjkzOTUzZTdiYzYzYzg5ZTI0MzQ1NWQxNzVkYWE0ZDMwOGMzNzVmNTBjNTk0NDg0MzUyYzg4YzU5MmIyYmJjZTAiLCJkcCI6IjkzZGU3MmJiMTNlZDFkYzgwZWRkY2YwMWFjNjIxZDc3Y2UzYjQ0ODVhYmNjNWM0ZmFmM2JhYjA2ZDA4Yzg0ZmIifSx7InV0eG8iOnsicmMiOiJhbGljZSIsInN0IjpbMCwwLDAsMV0sIm9pIjozLCJpZCI6IjkzZGU3MmJiMTNlZDFkYzgwZWRkY2YwMWFjNjIxZDc3Y2UzYjQ0ODVhYmNjNWM0ZmFmM2JhYjA2ZDA4Yzg0ZmIiLCJzIjo0ODUxMSwic2MiOiI3NmE5MTQ3ODUxY2IxMGYyMjFiYTJlZmM5OTZkMzQwN2JiZTRlNWRjYTUyOTc0ODhhYyJ9LCJzcCI6ImFsaWNlIiwic24iOjAsInNsIjoxMDcsInNjIjoiNDgzMDQ1MDIyMTAwODU2ZjI0YTQwN2UxZmI3Mzk4ZmY4ZmJiYzJhNzc1ODg5YWIxN2FhZDM2YTVjODFiZjkwZTkzN2UxNWRlMzJjYzAyMjA1OTAxMDE3YTM0NWE5Mzg0ZDhhNWRiMjFmOTIwMGViYjZhMTZhNDc3MDkzNDJiNmQ1MDY0MDlkMTlmM2RiNGYyNDEyMTAzMjE3OGYzY2RiZWUzNmNlYTQ3ZDBlOGU4YjVhYjM4NWU2MDM2OTVmYTgzOGQzYWE5MDU4NTBkYzUxMjE2ZTAyZiIsImRwIjoiOTNkZTcyYmIxM2VkMWRjODBlZGRjZjAxYWM2MjFkNzdjZTNiNDQ4NWFiY2M1YzRmYWYzYmFiMDZkMDhjODRmYiJ9XSwidG8iOlt7InJjIjpudWxsLCJzdCI6WzAsMCwwLDEsOTcsMTA4LDEwNSw5OSwxMDFdLCJvaSI6MCwiaWQiOiI4ZWQzMmUxYzY2MjUzNWYxYzU0MzdlYzkyYWVjNDkyMjE2NjY3MzVmZjkyZDQ2ODg0MmU4NmYwMTVmMjQ4ZTk5IiwicyI6MTIsInNjIjoiNzZhOTE0ZTM5ZDE5ODNlY2JiNzEyMDg5YTEyNTY3NGYwNWVhMmM4NzZjNjk5NDg4YWMifSx7InJjIjoiaGVsZW5lIiwic3QiOlswLDAsMCwxLDEwNCwxMDEsMTA4LDEwMSwxMTAsMTAxXSwib2kiOjEsImlkIjoiOGVkMzJlMWM2NjI1MzVmMWM1NDM3ZWM5MmFlYzQ5MjIxNjY2NzM1ZmY5MmQ0Njg4NDJlODZmMDE1ZjI0OGU5OSIsInMiOjgwMCwic2MiOiI3NmE5MTQ0NmVlOGVmNjg3YjczN2JlYmVjYzk0ZGUwYmQzYzA3YTQ4Mzk2YTU0ODhhYyJ9LHsicmMiOiJhbGljZSIsInN0IjpbMCwwLDAsMSw5NywxMDgsMTA1LDk5LDEwMV0sIm9pIjoyLCJpZCI6IjhlZDMyZTFjNjYyNTM1ZjFjNTQzN2VjOTJhZWM0OTIyMTY2NjczNWZmOTJkNDY4ODQyZTg2ZjAxNWYyNDhlOTkiLCJzIjo4MDAsInNjIjoiNzZhOTE0NTQwOTZjYTFmNjM5NmQ4MWVlZGRlOGE3YzQzZTIwMWU5NTNhMTk2Zjg4YWMifSx7InJjIjoiYWxpY2UiLCJzdCI6WzAsMCwwLDFdLCJvaSI6MywiaWQiOiI4ZWQzMmUxYzY2MjUzNWYxYzU0MzdlYzkyYWVjNDkyMjE2NjY3MzVmZjkyZDQ2ODg0MmU4NmYwMTVmMjQ4ZTk5IiwicyI6NDc1NzYsInNjIjoiNzZhOTE0Nzg1MWNiMTBmMjIxYmEyZWZjOTk2ZDM0MDdiYmU0ZTVkY2E1Mjk3NDg4YWMifV19XSwibGVuIjozLCJzYWZlIjp0cnVlfQ=="),
		Identifier: "deaa020c11e8aab1e4f5a8cc0a6982896b2db0dd588db1e53ca0e1f4a348f026",
	}

	marsh, err := json.Marshal(pr)
	if err != nil {
		t.Errorf("error marshalling chat request: %v\n", err)
	}
	reader := bytes.NewReader(marsh)

	req := httptest.NewRequest("POST", "/", reader)
	rsp := httptest.NewRecorder()

	HandlePaymentRequest(rsp, req)

	if rsp.Code != 200 {
		t.Errorf("error handleing chat request")
	}
}

func TestHandleChatRequest(t *testing.T) {

	wim, err := os.ReadFile("C:\\Users\\coton\\Pictures\\Chan\\SigridUndset.jpg")
	// im, err := os.ReadFile("/home/scott/Pictures/basedretard.png")
	if err != nil {
		t.Errorf("error reading file image for user init test: %v\n", err)
	}

	mediaID := sha256Hex(wim)

	media := Down4Media{
		Identifier: mediaID,
		Data:       wim,
		Metadata: map[string]string{
			"o":   "scott",
			"ts":  strconv.FormatInt(time.Now().Unix(), 10),
			"vid": "false",
			"shr": "true",
			"trv": "false",
			"ptv": "false",
			"pto": "false",
			"ar":  "1.0",
		},
	}

	rb := randomBuffer(16)
	randomID := hex.EncodeToString(rb)

	t.Logf("randomID: %v\n", randomID)

	cr := ChatRequest{
		Targets: []string{"scott"},
		Media:   media,
		Message: Down4Message{
			Root:      "a29f13efcc4b9cab1d5b7e8a5d785534c7a4ca202d1a657c74f4a75dc0e6da4b",
			Type:      "chat",
			MessageID: randomID,
			MediaID:   media.Identifier,
			SenderID:  "beast",
			Text:      "She was a wonderful writer.",
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
		Text:     "Hello scott, this is the god of down4",
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

func TestHandleSnipRequest(t *testing.T) {

	wim, err := os.ReadFile("C:\\Users\\coton\\Pictures\\Chan\\Capture.PNG")
	// im, err := os.ReadFile("/home/scott/Pictures/basedretard.png")
	if err != nil {
		t.Errorf("error reading file image for user init test: %v\n", err)
	}

	rb := randomBuffer(16)
	rb2 := randomBuffer(16)
	randomMediaID := hex.EncodeToString(rb)
	randomMessageID := hex.EncodeToString(rb2)

	sr := SnipRequest{
		Targets: []string{"scott"},
		Message: Down4Message{
			MessageID: randomMessageID,
			SenderID:  "itachi",
			Type:      "snip",
			Timestamp: unixMilliseconds(),
			MediaID:   randomMediaID,
		},
		Media: Down4Media{
			Data:       wim,
			Identifier: randomMediaID,
			Metadata: map[string]string{
				"o":   "helene",
				"ts":  strconv.FormatInt(unixMilliseconds(), 10),
				"vid": "false",
				"shr": "false",
				"txt": "This guy is extremely powerful",
				"trv": "false",
				"ptv": "false",
				"pto": "false",
				"ar":  "1.0",
			},
		},
	}

	marsh, err := json.Marshal(sr)
	if err != nil {
		log.Printf("error marshalling ping: %v", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(marsh))
	rsp := httptest.NewRecorder()

	HandleSnipRequest(rsp, req)

	if rsp.Code != 200 {
		t.Errorf("error handling ping request")
	}
}

func TestHandleHyperchatRequest(t *testing.T) {

	rb := randomBuffer(16)
	randomID := hex.EncodeToString(rb)

	root := sha256Hex(rb)

	hr := HyperchatRequest{
		Targets:   []string{"scott", "scorpion", "helene"},
		WordPairs: []string{"hell fee", "hip boot", "chill herb", "speard salt", "paint dough"},
		Message: Down4Message{
			Type:      "chat",
			Root:      root,
			MessageID: randomID,
			SenderID:  "itachi",
			Text:      "Another day, another random word. Hallelujiah",
			Timestamp: unixMilliseconds(),
		},
	}

	marsh, err := json.Marshal(hr)
	if err != nil {
		t.Errorf("error marshalling hyperchat request: %v\n", err)
	}
	reader := bytes.NewReader(marsh)

	req := httptest.NewRequest("POST", "/", reader)
	rsp := httptest.NewRecorder()

	HandleHyperchatRequest(rsp, req)

	if rsp.Code != 200 {
		t.Errorf("error handleing hyperchat request")
	}
}

func TestHandleGroupRequest(t *testing.T) {

	gm, err := os.ReadFile("C:\\Users\\coton\\Pictures\\Chan\\1630440242847.jpg")
	if err != nil {
		t.Errorf("error reading pictures for group image: %v\n", err)
	}

	mediaMD := map[string]string{
		"o":   "jeff",
		"pto": "false",
		"ptv": "false",
		"ar":  "1.0",
		"shr": "true",
		"trv": "false",
		"ts":  strconv.FormatInt(unixMilliseconds(), 10),
	}

	mediaID := sha1Hex(gm)

	rb := randomBuffer(16)
	randomID := hex.EncodeToString(rb)

	root := sha256Hex(rb)

	t.Logf("Group root: %v\n", root)

	gr := GroupRequest{
		Targets:   []string{"beast", "scorpion", "caal", "scott", "wolf"},
		GroupID:   root,
		GroupName: "The Rats",
		Private:   true,
		GroupMedia: Down4Media{
			Identifier: mediaID,
			Data:       gm,
			Metadata:   mediaMD,
		},
		Message: Down4Message{
			Root:      root,
			MessageID: randomID,
			Type:      "chat",
			Text:      "The rats are in town motherfuckers.",
			SenderID:  "kurt",
			Timestamp: unixMilliseconds(),
		},
	}

	marsh, err := json.Marshal(gr)
	if err != nil {
		t.Errorf("error marshalling groupRequest request: %v\n", err)
	}
	reader := bytes.NewReader(marsh)

	req := httptest.NewRequest("POST", "/", reader)
	rsp := httptest.NewRecorder()

	HandleGroupRequest(rsp, req)

	if rsp.Code != 200 {
		t.Errorf("error handleing group request")
	}
}

func TestUsernameValidityHttp(t *testing.T) {

	var validUser, invalidUser []byte = []byte("fuckalluserid"), []byte("magnus")
	req1 := httptest.NewRequest("POST", "/", bytes.NewReader(validUser))
	req2 := httptest.NewRequest("POST", "/", bytes.NewReader(invalidUser))

	rr1 := httptest.NewRecorder()
	rr2 := httptest.NewRecorder()

	IsValidUsername(rr1, req1)
	IsValidUsername(rr2, req2)

	fmt.Printf("validUser: %v\ninvalidUser: %v\n", rr1.Code, rr2.Code)
	if rr1.Code != 200 || rr2.Code != 500 {
		t.Errorf("validUser code should be 200, is: %v\n invalidUser code should be 500, is: %v\n", rr1.Code, rr2.Code)
	}
}

func TestUserInitializationHttp(t *testing.T) {
	wim, err := os.ReadFile("C:\\Users\\coton\\Pictures\\Chan\\1631267891994.jpg")
	// im, err := os.ReadFile("/home/scott/Pictures/basedretard.png")
	if err != nil {
		t.Errorf("error reading file image for user init test: %v\n", err)
	}

	jeff := InitUserInfo{
		Neuter:     "IAmANeuteMotherfuckers",
		Secret:     "IAmASecret",
		Token:      "IAmAToken",
		Name:       "Magnus",
		Lastname:   "Carlsen",
		Identifier: "cat",
		Image: Down4Media{
			Identifier: "catMagnusCarlsen",
			Data:       wim,
			Metadata: map[string]string{
				"o":   "cat",
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

func TestGetNodes(t *testing.T) {
	nodesToGet := []byte("helene")
	rdr := bytes.NewReader(nodesToGet)

	req := httptest.NewRequest("POST", "/", rdr)
	rsp := httptest.NewRecorder()

	GetNodes(rsp, req)

	var nodes []FullNode
	if err := json.NewDecoder(rsp.Body).Decode(&nodes); err != nil {
		t.Errorf("error decoding response: %v\n", err)
	}

	for i, node := range nodes {
		t.Logf("\nnode #%v\nid: %v\nname: %v\ntype: %v\nimID: %v\nneuteur: %v\n", i, node.Identifier, node.Name, node.Type, node.Image.Identifier, node.Neuter)
	}
}

func TestGetMessageMedia(t *testing.T) {
	mediaID := "ab1893d1f5128ee462f63759294d9aeb7377207851336aeacfc673bede71d62c"
	rdr := bytes.NewReader([]byte(mediaID))

	rsp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", rdr)

	GetMessageMedia(rsp, req)

	var d4media Down4Media
	if err := json.NewDecoder(rsp.Body).Decode(&d4media); err != nil {
		t.Errorf("error decoding message media: %v\n", err)
	}

	if rsp.Code == 200 {
		t.Logf("\ngot media\nid: %v\nmd: %v\n", d4media.Identifier, d4media.Metadata)
		os.WriteFile("C:\\Users\\coton\\Desktop\\cat.png", d4media.Data, fs.ModeDevice)
	} else {
		t.Errorf("error getting message media")
	}
}

func TestGetMessageMediaMetadata(t *testing.T) {
	mediaID := "6bfac783ae08397f449e8f04e96af648fd710aae0bcd9b16541da8538e7991db"
	rdr := bytes.NewReader([]byte(mediaID))

	rsp := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/", rdr)

	GetMediaMetadata(rsp, req)
	var md map[string]string
	if err := json.NewDecoder(rsp.Body).Decode(&md); err != nil {
		t.Errorf("error decoding message medatada: %v\n", err)
	}

	if rsp.Code == 200 {
		t.Logf("\nmedia metadata: %v\n", md)
	} else {
		t.Errorf("error getting message metadata\n")
	}
}
