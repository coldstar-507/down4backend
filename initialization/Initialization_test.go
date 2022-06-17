package initialization

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

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

	im, err := os.ReadFile("C:/Users/coton/Pictures/Chan/393yogreenlandshark.jpg")
	if err != nil {
		t.Errorf("error reading file image for user init test: %v\n", err)
	}

	jeff := InitUserInfo{
		Secret:     "fkldsajffdfdflsdkajf",
		Name:       "Baleine",
		Lastname:   "Big",
		Identifier: "jig",
		Image: Down4Media{
			Identifier: "wlallallala",
			Data:       im,
			Metadata: map[string]string{
				"o":   "big",
				"ts":  "32910390",
				"trv": "false",
				"ptv": "false",
				"shr": "true",
				"vid": "false",
				"pto": "false",
			},
		},
		Token:  "fjdsklafj89dfjs8afjsdokfj",
		Neuter: "jfdkls213123jfldksjfl3213kdsjflkdsjf321lkds",
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

func TestMoneyInitializationHttp(t *testing.T) {

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
