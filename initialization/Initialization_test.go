package initialization

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

	var scott, andrew []byte = []byte("scott"), []byte("andrew")
	req1 := httptest.NewRequest("POST", "/", bytes.NewReader(scott))
	req2 := httptest.NewRequest("POST", "/", bytes.NewReader(andrew))

	rr1 := httptest.NewRecorder()
	rr2 := httptest.NewRecorder()

	IsValidUsername(rr1, req1)
	IsValidUsername(rr2, req2)

	fmt.Printf("scott: %v\nandrew: %v\n", rr1.Code, rr2.Code)
	if rr1.Code != 500 || rr2.Code != 200 {
		t.Errorf("scott code should be 200, is: %v\n andrew code should be 500, is: %v\n", rr1.Code, rr2.Code)
	}
}

func TestUserInitializationHttp(t *testing.T) {

	im, err := os.ReadFile("/home/scott/Desktop/van.jpg")
	if err != nil {
		t.Errorf("error reading file image for user init test: %v\n", err)
	}

	jeff := InitUserInfo{
		Name:     "Jeff",
		Lastname: "Harrisson",
		Username: "guylaine",
		Image:    im,
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

	var jeff []byte = []byte("guylaine")
	req := httptest.NewRequest("POST", "/", bytes.NewReader(jeff))

	rr := httptest.NewRecorder()
	InitUserMoney(rr, req)

	var rd InitMoneyData
	json.NewDecoder(rr.Body).Decode(&rd)

	words := strings.Split(rd.Mnemonic, " ")

	if len(words) != 12 {
		t.Errorf("invalide mnemonic, not 12 words: %v\n", words)
	}

	fmt.Printf("Mnemonic: %v\nDown4priv : %v\nMaster: %v\nIndex: %v\nChange: %v\n", rd.Mnemonic, rd.Down4Priv, rd.Master, rd.UpperIndex, rd.UpperChange)

}
