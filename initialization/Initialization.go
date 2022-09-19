package initialization

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/libsv/go-bk/bip39"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrUsernameAlreadyExists = errors.New("username already exists")

type InitializationServer struct {
	FS     *firestore.Client
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

	fs, err := firestore.NewClient(ctx, "down4-26ee1")
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
	}

	if err := uploadNodeMedia(ctx, info.Image); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error uploading media in user initialization: %v\n", err)
	}

	nodeRef := is.FS.Collection("Nodes").Doc(info.Identifier)
	createFirestoreNode := func(ctx context.Context, tx *firestore.Transaction) error {
		nodeSnap, err := tx.Get(nodeRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if nodeSnap.Exists() {
			return ErrUsernameAlreadyExists
		}
		nodeInfo := FireStoreNode{
			Neuter:     info.Neuter,
			Identifier: info.Identifier,
			Name:       info.Name,
			Lastname:   info.Lastname,
			ImageID:    info.Image.Identifier,
			Type:       "user",
			Longitude:  0,
			Latitude:   0,
			Friends:    make([]string, 0),
			Admins:     make([]string, 0),
			Messages:   make([]string, 0),
			Childs:     make([]string, 0),
			Parents:    make([]string, 0),
			Words:      make([]string, 0),
			Group:      make([]string, 0),
		}
		if err := tx.Set(nodeRef, nodeInfo); err != nil {
			return err
		}

		return nil
	}

	if err := is.FS.RunTransaction(ctx, createFirestoreNode); err != nil {
		is.NDBCKT.Object(info.Image.Identifier).Delete(ctx) // try deleting object, doesn't really matter if it fails
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing user info transaction: %v\n", err)
	}

	createRealtimeUser := func(tn db.TransactionNode) (interface{}, error) {
		var currentInfos map[string]interface{}
		if err := tn.Unmarshal(&currentInfos); err != nil {
			return nil, err
		}
		if len(currentInfos) > 0 {
			return nil, ErrUsernameAlreadyExists
		}
		realtimeUserInfo := UserInfo{
			Secret:   info.Secret,
			Activity: time.Now().Unix(),
			Token:    info.Token,
			Snips:    make(map[string]string),
			Messages: make(map[string]string),
			Payments: make(map[string]string),
		}
		return realtimeUserInfo, nil
	}

	if err := is.RTDB.NewRef(info.Identifier).Transaction(ctx, createRealtimeUser); err != nil {
		nodeRef.Delete(ctx)
		is.NDBCKT.Object(info.Image.Identifier).Delete(ctx)
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing userinfo on realtimeDB in initUser: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func GenerateMnemonic(w http.ResponseWriter, r *http.Request) {

	const (
		passKey = ""
		childNo = "0"
	)

	entropy, err := bip39.GenerateEntropy(bip39.EntWords12)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating entropy: %v\n", err)
	}

	mnemonic, _, err := bip39.Mnemonic(entropy, passKey)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating mnemonic: %v\n", err)
	}

	if _, err := w.Write([]byte(mnemonic)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing mnemonic to response: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func isValidUsername(ctx context.Context, username string) (bool, error) {

	ref := is.FS.Collection("Nodes").Doc(username)

	snap, err := ref.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return false, err
	}

	return !snap.Exists(), nil
}

func uploadNodeMedia(ctx context.Context, media Down4Media) error {
	obj := is.NDBCKT.Object(media.Identifier)
	w := obj.NewWriter(ctx)
	w.Write(media.Data)
	if err := w.Close(); err != nil {
		return err
	}
	obj.Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: media.Metadata,
	})
	return nil
}

// func generateHexMediaID(media []byte, uid string) (string, error) {
// 	h := sha1.New()
// 	data := append([]byte(uid), media...)
// 	if _, err := h.Write(data); err != nil {
// 		return "", nil
// 	}
// 	hash := h.Sum(nil)
// 	return hex.EncodeToString(hash), nil
// 	// return base58.Encode(hash), nil
// }
