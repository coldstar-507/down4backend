package initialization

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"github.com/libsv/go-bk/base58"
	"github.com/libsv/go-bk/bip32"
	"github.com/libsv/go-bk/bip39"
	"github.com/libsv/go-bk/chaincfg"
)

type InitMoneyData struct {
	Mnemonic    string `json:"mnemonic"`
	Down4Priv   string `json:"down4priv"`
	Master      string `json:"master"`
	LowerIndex  int    `json:"lowerindex"`
	UpperIndex  int    `json:"upperindex"`
	LowerChange int    `json:"lowerchange"`
	UpperChange int    `json:"upperchange"`
}

type Down4Media struct {
	Identifier string `json:"id"`
	Data       []byte `json:"d"`
}

type InitUserInfo struct {
	Username string `json:"id"`
	Name     string `json:"nm"`
	Lastname string `json:"ln"`
	Image    []byte `json:"im"`
	Token    string `json:"tkn"`
}

type RealTimeNode struct {
	Identifier string            `json:"id"`
	Type       string            `json:"t"`
	Name       string            `json:"nm"`
	Lastname   string            `json:"ln"`
	ImageID    string            `json:"im"`
	Token      string            `json:"tkn"`
	Friends    map[string]string `json:"frd"`
	Messages   map[string]string `json:"msg"`
	Admins     map[string]string `json:"adm"`
	Childs     map[string]string `json:"chl"`
	Parents    map[string]string `json:"prt"`
}

var ErrUsernameAlreadyExists = errors.New("username already exists")

type InitializationServer struct {
	RTDB   *db.Client
	NDBCKT *storage.BucketHandle
}

var is InitializationServer

func init() {

	config := &firebase.Config{
		DatabaseURL: "https://down4-26ee1-default-rtdb.firebaseio.com/",
	}

	ctx := context.Background()

	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	rtdb, err := app.Database(ctx)
	if err != nil {
		log.Fatalf("error initializing db: %v\n", err)
	}

	stor, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("error initializing storage: %v\n", err)
	}

	ndbcket := stor.Bucket("down4-26ee1-nodes")

	is = InitializationServer{
		RTDB:   rtdb,
		NDBCKT: ndbcket,
	}

}

func IsValidUsername(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading username from request: %v\n", err)
	}

	username := string(buf)

	var isValid bool
	if isValid, err = isValidUsername(ctx, username); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error checking if username is valid: %v\n", err)
	}

	if isValid {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func InitUser(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	var info InitUserInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding userinfo: %v\n", err)
		return
	}

	mediaID, err := generateB64MediaID(info.Image, info.Username)

	createUser := func(tn db.TransactionNode) (interface{}, error) {
		var currentInfos map[string]interface{}
		if err := tn.Unmarshal(&currentInfos); err != nil {
			return nil, err
		}
		if len(currentInfos) == 0 {
			return RealTimeNode{
				Identifier: info.Username,
				Name:       info.Name,
				Lastname:   info.Lastname,
				ImageID:    mediaID,
			}, nil
		} else {
			return nil, ErrUsernameAlreadyExists
		}
	}

	userRef := is.RTDB.NewRef("/Nodes/" + info.Username)
	if err := userRef.Transaction(ctx, createUser); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing user info transaction: %v\n", err)
	}

	if err != nil {
		userRef.Delete(ctx) // can't confirm this delete, if this fails, user will be written but unusable
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating media ID in user initializaion: %v\n", err)
	}

	d4Media := Down4Media{Identifier: mediaID, Data: info.Image}
	if err = uploadNodeMedia(ctx, d4Media); err != nil {
		userRef.Delete(ctx) // can't confirm this delete, if this fails, user will be written but unusable
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error uploading media in user initialization: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func InitUserMoney(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	const (
		passKey = ""
		childNo = "0"
	)

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading username: %v\n", err)
	}

	username := string(buf)

	entropy, err := bip39.GenerateEntropy(bip39.EntWords12)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating entropy: %v\n", err)
	}

	mnemonic, seed, err := bip39.Mnemonic(entropy, passKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating mnemonic: %v\n", err)
	}

	master, err := bip32.NewMaster(seed, &chaincfg.MainNet)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating master: %v\n", err)
	}

	// x'/y'/z'/i
	// x' -> application 4' for down4
	// y' -> markets | 0' -> user | 1' -> submarket #1 | 2' -> submarket #2 ...
	// z' -> items   | always 0' if y' = 0' | 0' -> item 0 | 1' -> item 1 ...
	// i % 2 = 0 => target adresse | i % 2 = 1 => change adresse
	down4priv, err := master.DeriveChildFromPath("4'/0'/0'")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error deriving down4priv: %v\n", err)
	}

	down4neuter, err := down4priv.Neuter()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error making neuter: %v\n", err)
	}

	moneyinfo := map[string]interface{}{"ix": 0, "cg": 1, "nt": down4neuter.String()}
	userMoneyRef := is.RTDB.NewRef("/Money/" + username)
	if err = userMoneyRef.Set(ctx, moneyinfo); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing info on database: %v\n", err)
	}

	outputInfo := InitMoneyData{
		Master:      master.String(),
		Down4Priv:   down4neuter.String(),
		LowerIndex:  0,
		UpperIndex:  0,
		LowerChange: 1,
		UpperChange: 1,
		Mnemonic:    mnemonic,
	}

	if err = json.NewEncoder(w).Encode(outputInfo); err != nil {
		userMoneyRef.Delete(ctx)
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing data to response: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func uploadNodeMedia(ctx context.Context, media Down4Media) error {

	w := is.NDBCKT.Object(media.Identifier).NewWriter(ctx)
	if _, err := w.Write(media.Data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

func isValidUsername(ctx context.Context, username string) (bool, error) {

	var m map[string]string
	ref := is.RTDB.NewRef("/Nodes/" + username)
	if err := ref.Get(ctx, &m); err != nil {
		return false, err
	}

	if len(m) == 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func generateB64MediaID(media []byte, uid string) (string, error) {
	h := sha1.New()
	data := append([]byte(uid), media...)
	if _, err := h.Write(data); err != nil {
		return "", nil
	}
	hash := h.Sum(nil)
	return base58.Encode(hash), nil
}
