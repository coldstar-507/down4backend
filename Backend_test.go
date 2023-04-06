package backend

import (
	"bytes"
	// "encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	// "io/fs"

	// "log"

	"net/http/httptest"
	// "os"
	"strings"
	"testing"
)

// func randomBuffer(len int) []byte {
// 	buf := make([]byte, len)
// 	rand.Seed(time.Now().UnixNano())
// 	rand.Read(buf)
// 	return buf
// }

// func sha256Hex(data []byte) string {
// 	h := sha256.New()
// 	h.Write(data)
// 	hashed := h.Sum(make([]byte, 0))
// 	return hex.EncodeToString(hashed)
// }

// func sha1Hex(data []byte) string {
// 	h := sha1.New()
// 	h.Write(data)
// 	hashed := h.Sum(make([]byte, 0))
// 	return hex.EncodeToString(hashed)
// }

func TestTimestamp(t *testing.T) {
	ts := unixMilliseconds()

	t.Logf("timestamp unix milliseconds: %v\n", ts)
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
	// wim, err := os.ReadFile("C:\\Users\\coton\\Pictures\\Chan\\1631267891994.jpg")
	// im, err := os.ReadFile("/home/scott/Pictures/basedretard.png")
	// if err != nil {
	// 	t.Errorf("error reading file image for user init test: %v\n", err)
	// }

	// randomID := nByteBase64ID(16)

	jeff := InitUserInfo{
		Neuter:     "IAmANeuteMotherfuckers",
		Secret:     "IAmASecret",
		Token:      "IAmAToken",
		Name:       "Magnus",
		Lastname:   "Carlsen",
		Identifier: "cat",
		Image:      "IAmMediaID",
		// Image: Down4Media{
		// 	Identifier: randomID,
		// 	Data:       base64.StdEncoding.EncodeToString(wim),
		// 	Metadata: map[string]string{
		// 		"o":   "cat",
		// 		"ts":  strconv.FormatInt(unixMilliseconds(), 10),
		// 		"trv": "false",
		// 		"ptv": "false",
		// 		"shr": "true",
		// 		"vid": "false",
		// 		"pto": "false",
		// 	},
		// },
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
	nodesToGet := []byte("madara one csw")
	rdr := bytes.NewReader(nodesToGet)

	req := httptest.NewRequest("POST", "/", rdr)
	rsp := httptest.NewRecorder()

	GetNodes(rsp, req)

	var nodes []FullNode
	if err := json.NewDecoder(rsp.Body).Decode(&nodes); err != nil {
		t.Errorf("error decoding response: %v\n", err)
	}

	t.Logf("There are %v nodes\n", len(nodes))

	for _, node := range nodes {
		t.Logf("\nNodeID=%s\nName=%s\nMediaID=%s\nMediaOwner=%s\n=========\n", node.Node["id"], node.Node["name"], node.Node["mediaID"], node.Metadata["ownerID"])
	}
}

// func TestGetMessageMedia(t *testing.T) {
// 	mediaID := "ab1893d1f5128ee462f63759294d9aeb7377207851336aeacfc673bede71d62c"
// 	rdr := bytes.NewReader([]byte(mediaID))

// 	rsp := httptest.NewRecorder()
// 	req := httptest.NewRequest("POST", "/", rdr)

// 	GetMessageMedia(rsp, req)

// 	var d4media Down4Media
// 	if err := json.NewDecoder(rsp.Body).Decode(&d4media); err != nil {
// 		t.Errorf("error decoding message media: %v\n", err)
// 	}

// 	mediaData, err := base64.StdEncoding.DecodeString(d4media.Data)
// 	if err != nil {
// 		t.Errorf("error decoding media data from base64: %v\n", mediaData)
// 	}

// 	if rsp.Code == 200 {
// 		t.Logf("\ngot media\nid: %v\nmd: %v\n", d4media.Identifier, d4media.Metadata)
// 		os.WriteFile("C:\\Users\\coton\\Desktop\\cat.png", mediaData, fs.ModeDevice)
// 	} else {
// 		t.Errorf("error getting message media")
// 	}
// }

// func TestGetMessageMediaMetadata(t *testing.T) {
// 	mediaID := "6bfac783ae08397f449e8f04e96af648fd710aae0bcd9b16541da8538e7991db"
// 	rdr := bytes.NewReader([]byte(mediaID))

// 	rsp := httptest.NewRecorder()
// 	req := httptest.NewRequest("POST", "/", rdr)

// 	GetMediaMetadata(rsp, req)
// 	var md map[string]string
// 	if err := json.NewDecoder(rsp.Body).Decode(&md); err != nil {
// 		t.Errorf("error decoding message medatada: %v\n", err)
// 	}

// 	if rsp.Code == 200 {
// 		t.Logf("\nmedia metadata: %v\n", md)
// 	} else {
// 		t.Errorf("error getting message metadata\n")
// 	}
// }
