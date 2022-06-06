package initializationFS

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/libsv/go-bk/base58"
	"github.com/libsv/go-bk/bip32"
	"github.com/libsv/go-bk/bip39"
	"github.com/libsv/go-bk/chaincfg"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrUsernameAlreadyExists = errors.New("username already exists")

type InitializationServer struct {
	FS     *firestore.Client
	NDBCKT *storage.BucketHandle
}

var is InitializationServer

func init() {

	ctx := context.Background()

	opt := option.WithCredentialsFile("C:/Users/coton/Documents/project-down4/service-accounts/down4-26ee1-firebase-adminsdk-im27t-392380b354.json")

	fs, err := firestore.NewClient(ctx, "down4-26ee1", opt)
	if err != nil {
		log.Fatalf("error initializing db: %v\n", err)
	}

	stor, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("error initializing storage: %v\n", err)
	}

	ndbcket := stor.Bucket("down4-26ee1-nodes")

	is = InitializationServer{
		FS:     fs,
		NDBCKT: ndbcket,
	}

}

func IsValidUsernameFS(w http.ResponseWriter, r *http.Request) {

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

func InitUserFS(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	var info InitUserInfo
	if err := json.NewDecoder(r.Body).Decode(&info); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding userinfo: %v\n", err)
	}

	mediaID, err := generateB64MediaID(info.Image, info.Identifier)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating b64Media ID: %v\n", err)
	}

	d4Media := Down4Media{Identifier: mediaID, Data: info.Image}
	if err = uploadNodeMedia(ctx, d4Media); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error uploading media in user initialization: %v\n", err)
	}

	nodeRef := is.FS.Collection("Nodes").Doc(info.Identifier)
	moneyRef := is.FS.Collection("Moneys").Doc(info.Identifier)
	userRef := is.FS.Collection("Users").Doc(info.Identifier)
	createUser := func(ctx context.Context, tx *firestore.Transaction) error {

		// nodeSnap, err := nodeRef.Get(ctx)
		nodeSnap, err := tx.Get(nodeRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if nodeSnap.Exists() {
			return ErrUsernameAlreadyExists
		}
		moneySnap, err := tx.Get(moneyRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if moneySnap.Exists() {
			return ErrUsernameAlreadyExists
		}
		userSnap, err := tx.Get(userRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if userSnap.Exists() {
			return ErrUsernameAlreadyExists
		}

		nodeInfo := FireStoreNode{
			Identifier: info.Identifier,
			Name:       info.Name,
			Lastname:   info.Lastname,
			ImageID:    mediaID,
			Type:       "usr",
			Longitude:  0,
			Latitude:   0,
			Friends:    make([]string, 0),
			Admins:     make([]string, 0),
			Messages:   make([]string, 0),
			Childs:     make([]string, 0),
			Parents:    make([]string, 0),
		}
		if err := tx.Set(nodeRef, nodeInfo); err != nil {
			return err
		}

		moneyInfo := PublicMoneyInfo{
			Neuter: info.Money.Neuter,
			Index:  info.Money.Index,
			Change: info.Money.Change,
		}
		if err := tx.Set(moneyRef, moneyInfo); err != nil {
			return err
		}

		userInfo := UserInfo{
			Secret:   info.Secret,
			Activity: time.Now().Unix(),
			Token:    info.Token,
		}
		if err := tx.Set(userRef, userInfo); err != nil {
			return err
		}

		return nil
	}

	if err := is.FS.RunTransaction(ctx, createUser); err != nil {
		is.NDBCKT.Object(mediaID).Delete(ctx) // try deleting object, doesn't really matter if it fails
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing user info transaction: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func GenerateUserMoneyInfoFS(w http.ResponseWriter, r *http.Request) {

	const (
		passKey = ""
		childNo = "0"
	)

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

	outputInfo := OutputMoneyInfo{
		Master:      master.String(),
		Down4Priv:   down4neuter.String(),
		LowerIndex:  0,
		UpperIndex:  0,
		LowerChange: 1,
		UpperChange: 1,
		Mnemonic:    mnemonic,
	}

	if err = json.NewEncoder(w).Encode(outputInfo); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing data to response: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func isValidUsername(ctx context.Context, username string) (bool, error) {

	ref := is.FS.Collection("Users").Doc(username)

	snap, err := ref.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return false, err
	}

	return !snap.Exists(), nil

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

func generateB64MediaID(media []byte, uid string) (string, error) {
	h := sha1.New()
	data := append([]byte(uid), media...)
	if _, err := h.Write(data); err != nil {
		return "", nil
	}
	hash := h.Sum(nil)
	return base58.Encode(hash), nil
}
