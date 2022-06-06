package initializationFS

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestUsernameValidityHttp(t *testing.T) {

	var shy, andrew []byte = []byte("shy"), []byte("andrew")
	req1 := httptest.NewRequest("POST", "/", bytes.NewReader(shy))
	req2 := httptest.NewRequest("POST", "/", bytes.NewReader(andrew))

	rr1 := httptest.NewRecorder()
	rr2 := httptest.NewRecorder()

	IsValidUsernameFS(rr1, req1)
	IsValidUsernameFS(rr2, req2)

	fmt.Printf("shy: %v\nandrew: %v\n", rr1.Code, rr2.Code)
	if rr1.Code != 500 || rr2.Code != 200 {
		t.Errorf("shy code should be 200, is: %v\n andrew code should be 500, is: %v\n", rr1.Code, rr2.Code)
	}
}

func TestUserInitializationHttp(t *testing.T) {

	im, err := os.ReadFile("C:/Users/coton/Pictures/Chan/393yogreenlandshark.jpg")
	if err != nil {
		t.Errorf("error reading file image for user init test: %v\n", err)
	}

	jeff := InitUserInfo{
		Secret:     "fkldsajffdfdflsdkajf",
		Name:       "Guy",
		Lastname:   "Big",
		Identifier: "slam",
		Image:      im,
		Token:      "fjdsklafj89dfjs8afjsdokfj",
		Money: PublicMoneyInfo{
			Index:  0,
			Change: 1,
			Neuter: "jfds98243joi43jtjfosdijf98432j19jfdslakfjdlsajf",
		},
	}

	marshalled, err := json.Marshal(jeff)
	if err != nil {
		t.Errorf("error marshalling info: %v\n", err)
	}

	req := httptest.NewRequest("POST", "/", bytes.NewReader(marshalled))
	rr := httptest.NewRecorder()

	InitUserFS(rr, req)
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
	GenerateUserMoneyInfoFS(rr, req)

	var rd OutputMoneyInfo
	json.NewDecoder(rr.Body).Decode(&rd)

	words := strings.Split(rd.Mnemonic, " ")

	if len(words) != 12 {
		t.Errorf("invalide mnemonic, not 12 words: %v\n", words)
	}

	fmt.Printf("Mnemonic: %v\nDown4priv : %v\nMaster: %v\nIndex: %v\nChange: %v\n", rd.Mnemonic, rd.Down4Priv, rd.Master, rd.UpperIndex, rd.UpperChange)

}
