package backend

import (
	// "bytes"

	"context"
	"strings"

	// "crypto/rand"
	// "encoding/base64"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	// "strconv"
	// "strings"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
	"github.com/libsv/go-bk/bip39"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrUsernameAlreadyExists = errors.New("username already exists")

type Server struct {
	FS      *firestore.Client
	RTDB    *db.Client
	MSGBCKT *storage.BucketHandle
	NDBCKT  *storage.BucketHandle
	MSGR    *messaging.Client
	URLOPTS *storage.SignedURLOptions
}

var s Server

func init() {

	// opt := option.WithCredentialsFile("C:/Users/coton/Documents/project-down4/service-accounts/down4-26ee1-8433e5b5e7d2.json")
	config := &firebase.Config{
		DatabaseURL: "https://down4-26ee1-default-rtdb.firebaseio.com/",
	}

	ctx := context.Background()

	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	msgr, err := app.Messaging(ctx)
	if err != nil {
		log.Fatalf("error initializing messager: %v\n", err)
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

	msgbckt := stor.Bucket("down4-26ee1-messages")
	ndbckt := stor.Bucket("down4-26ee1-nodes")

	s = Server{
		FS:      fs,
		RTDB:    rtdb,
		MSGBCKT: msgbckt,
		NDBCKT:  ndbckt,
		MSGR:    msgr,
		URLOPTS: &storage.SignedURLOptions{
			Method:  "GET",
			Expires: time.Now().Add(time.Hour * 96),
		},
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

	ref := s.FS.Collection("Nodes").Doc(username)

	snap, err := ref.Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error checking if username is valid: %v\n", err)
	}

	isValid := !snap.Exists()
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

	// if err := uploadNodeMedia(ctx, info.Image); err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	log.Fatalf("error uploading media in user initialization: %v\n", err)
	// }

	nodeRef := s.FS.Collection("Nodes").Doc(info.Identifier)
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
			ImageID:    info.Image,
			IsPrivate:  false,
			Type:       "user",
			Publics:    []string{},
			Privates:   []string{},
			// Friends:    make([]string, 0),
			// Childs:     make([]string, 0),
			// Parents:    make([]string, 0),
			// Words:      make([]string, 0),
		}
		if err := tx.Set(nodeRef, nodeInfo); err != nil {
			return err
		}

		return nil
	}

	if err := s.FS.RunTransaction(ctx, createFirestoreNode); err != nil {
		s.NDBCKT.Object(info.Image).Delete(ctx) // try deleting object, doesn't really matter if it fails
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
			Activity: unixMilliseconds(),
			Token:    info.Token,
		}
		return realtimeUserInfo, nil
	}

	if err := s.RTDB.NewRef("Users/"+info.Identifier).Transaction(ctx, createRealtimeUser); err != nil {
		nodeRef.Delete(ctx)
		s.NDBCKT.Object(info.Image).Delete(ctx)
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing userinfo on realtimeDB in initUser: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func GenerateMnemonic(w http.ResponseWriter, r *http.Request) {

	entropy, err := bip39.GenerateEntropy(bip39.EntWords12)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating entropy: %v\n", err)
	}

	mnemonic, _, err := bip39.Mnemonic(entropy, "")
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

func GetNodes(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	bytesIDs, err := io.ReadAll(r.Body) // expect a string of IDs linked by a space " "
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading data from request in GetNodes: %v\n", err)
	}

	ids := strings.Split(string(bytesIDs), " ")

	// buffered is faster and espicially more consistent
	errChan := make(chan *error, len(ids))
	nodeChan := make(chan *FullNode, len(ids))
	go func() {
		for _, id := range ids {
			id_ := id
			go getNode(ctx, id_, nodeChan, errChan)
		}
	}()

	nodes := make([]FullNode, 0)
	for range ids {
		select {
		case n := <-nodeChan:
			nodes = append(nodes, *n)
		case <-errChan:
			continue
		}
	}

	encodedData, err := json.Marshal(nodes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error marshalling nodes %v\n", err)
	}

	if _, err = w.Write(encodedData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing mashaled nodes in response: %v\n", err)
	}
}

// func GetMessageMedia(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	bodyBytes, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error reading body bytes: %v\n", err)
// 	}
// 	mediaID := string(bodyBytes)
// 	d4Media, err := getMediaWithData(ctx, s.MSGBCKT, mediaID)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error getting message media: %v\n", err)
// 	}
// 	if err := json.NewEncoder(w).Encode(*d4Media); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error encoding media to response: %v\n", err)
// 	}
// 	w.WriteHeader(http.StatusOK)
// }

// func GetMediaMetadata(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	bodyBytes, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error reading body bytes: %v\n", err)
// 	}
// 	mediaID := string(bodyBytes)
// 	metadata, err := getMediaMetadata(ctx, mediaID)
// 	if err != nil {
// 		w.WriteHeader(http.StatusNoContent)
// 	} else {
// 		marshalled, err := json.Marshal(*metadata)
// 		if err != nil {
// 			w.WriteHeader(http.StatusInternalServerError)
// 		} else {
// 			w.Write(marshalled)
// 			w.WriteHeader(http.StatusOK)
// 		}
// 	}
// }

// func GetPayment(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()

// 	bodyBytes, err := io.ReadAll(r.Body)
// 	if err != nil {
// 		log.Fatalf("error reading bodyBytes: %v\n", err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 	}

// 	paymentID := string(bodyBytes)

// 	rdr, err := s.MSGBCKT.Object(paymentID).NewReader(ctx)
// 	if err != nil {
// 		log.Fatalf("error creating reader for payment: %v\n", err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 	}

// 	paymentData, err := io.ReadAll(rdr)
// 	if err != nil {
// 		log.Fatalf("error reading payment id:%v, err:%v\n", paymentID, err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 	}

// 	if _, err = w.Write(paymentData); err != nil {
// 		log.Fatalf("error writing payment id:%v, err:%v\n", paymentID, err)
// 		w.WriteHeader(http.StatusInternalServerError)
// 	}
// }

// func getMediaWithData(ctx context.Context, hdl *storage.BucketHandle, mediaID string) (*Down4Media, error) {
// 	obj := hdl.Object(mediaID)
// 	attrs, err := obj.Attrs(ctx)
// 	if err != nil {
// 		log.Fatalf("error getting metadata of media: %v\n", err)
// 		return nil, err
// 	}
// 	rdr, err := obj.NewReader(ctx)
// 	if err != nil {
// 		log.Fatalf("error creating reader for media: %v\n", err)
// 		return nil, err
// 	}
// 	mediaData, err := io.ReadAll(rdr)
// 	if err != nil {
// 		log.Fatalf("error reading media: %v\n", err)
// 		return nil, err
// 	}

// 	down4Media := Down4Media{
// 		Identifier: mediaID,
// 		Data:       base64.StdEncoding.EncodeToString(mediaData),
// 		Metadata:   attrs.Metadata,
// 	}
// 	return &down4Media, nil
// }

func HandleMessageRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("Error decoding request json: %v\n", err)
	}

	nTarget := len(req.Targets)
	eCh1, eCh2 := make(chan *error, nTarget), make(chan *error, nTarget)
	tkCh := make(chan *string, nTarget)
	akCh := make(chan bool, nTarget)

	go getMessagingTokens(ctx, req.Targets, tkCh, eCh1)
	go pushEvent(ctx, req.Targets, akCh, eCh2, req.Data)

	tokens := make([]string, 0)
	for i := 0; i < 2*nTarget; i++ {
		select {
		case tk := <-tkCh:
			tokens = append(tokens, *tk)
		case e1 := <-eCh1:
			log.Printf("Error getting token: %v\n", e1)
		case <-akCh:
			continue
		case e2 := <-eCh2:
			log.Printf("Error pushing data: %v\n", e2)
		}
	}

	s.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: req.Header,
			Body:  req.Body,
		},
	})

	if err := updateActivity(ctx, req.Sender); err != nil {
		log.Printf("Error updating activity of %v\n", req.Sender)
	}
}

func getMessagingTokens(ctx context.Context, ids []string, ch chan *string, ech chan *error) {
	for _, id := range ids {
		id_ := id
		go func() {
			userTokenRef := s.RTDB.NewRef("Users/" + id_ + "/token/")
			var token string
			if err := userTokenRef.Get(ctx, &token); err != nil {
				ech <- &err
			} else {
				ch <- &token
			}
		}()
	}
}

// func getMediaMetadata(ctx context.Context, mediaID string) (*map[string]string, error) {
// 	metadata, err := s.MSGBCKT.Object(mediaID).Attrs(ctx)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &metadata.Metadata, nil
// }

func updateActivity(ctx context.Context, uid string) error {
	err := s.RTDB.NewRef("Users/"+uid+"/activity").Set(ctx, unixMilliseconds())
	return err
}

func pushEvent(ctx context.Context, targets []string, ch chan bool, ech chan *error, payload string) {
	for _, target := range targets {
		target_ := target
		go func() {
			ref := s.RTDB.NewRef("Users").Child(target_).Child("M")
			_, err := ref.Push(ctx, payload)
			if err != nil {
				ech <- &err
			} else {
				ch <- true
			}
		}()
	}
}

// func uploadNodeMedia(ctx context.Context, media Down4Media) error {
// 	obj := s.NDBCKT.Object(media.Identifier)
// 	w := obj.NewWriter(ctx)

// 	mediadata, err := base64.StdEncoding.DecodeString(media.Data)
// 	if err != nil {
// 		return err
// 	}

// 	w.Write(mediadata)
// 	if err := w.Close(); err != nil {
// 		return err
// 	}
// 	obj.Update(ctx, storage.ObjectAttrsToUpdate{
// 		Metadata: media.Metadata,
// 	})
// 	return nil
// }

func getNode(ctx context.Context, id string, nodeChan chan *FullNode, errChan chan *error) {
	var fsn map[string]interface{}
	snap, err := s.FS.Collection("Nodes").Doc(id).Get(ctx)
	if err != nil {
		errChan <- &err
		log.Printf("error getting node id: %s from collection 'Nodes': %v\n", id, err)
		return
	}

	if err := snap.DataTo(&fsn); err != nil {
		errChan <- &err
		log.Printf("error reading snapshot data into FireStoreNode for user: %s: %v\n", id, err)
		return
	}

	metadataChan := make(chan *map[string]string, 1)
	dataChan := make(chan *string, 1)
	errorChan := make(chan *error, 2)

	mediaID, isString := fsn["mediaID"].(string)

	var data string
	var metadata map[string]string
	if isString && len(mediaID) != 0 {
		obj := s.NDBCKT.Object(mediaID)
		go func() {
			if att, err := obj.Attrs(ctx); err != nil {
				log.Printf("Sending error")
				errChan <- &err
			} else {
				log.Printf("Sending metadata")
				metadataChan <- &att.Metadata
			}

		}()

		go func() {
			rdr, err := obj.NewReader(ctx)
			if err != nil {
				log.Printf("Sending error")
				errorChan <- &err
			}
			if d, err := io.ReadAll(rdr); err != nil {
				log.Printf("Sending error")
				errChan <- &err
			} else {
				print("Seding data")
				s := base64.StdEncoding.EncodeToString(d)
				dataChan <- &s
			}
		}()

		for i := 0; i < 2; i++ {
			select {
			case e := <-errChan:
				log.Printf("Error getting media info for mediaID: %v, err: %v\n", mediaID, *e)
			case d := <-dataChan:
				data = *d
			case m := <-metadataChan:
				metadata = *m

			}
		}

	}

	node := FullNode{
		Node:     fsn,
		Metadata: metadata,
		Data:     data,
	}

	nodeChan <- &node
}

func unixMilliseconds() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

// func nByteBase64ID(n int) string {
// 	buf := make([]byte, n)
// 	rand.Read(buf)
// 	return base64.StdEncoding.EncodeToString(buf)
// }

// func nByteBase58ID(n int) string {
// 	buf := make([]byte, n)
// 	rand.Read(buf)
// 	return base58.Encode(buf)
// }

// func pushPay(ctx context.Context, objectID string, targets []string, ch chan bool, ech chan *error) {
// 	for _, target := range targets {
// 		target_ := target
// 		dbRef := s.RTDB.NewRef("Users").Child(target_).Child("M").Child(objectID)
// 		if err := dbRef.Set(ctx, "p"); err != nil {
// 			ech <- &err
// 		} else {
// 			ch <- true
// 		}
// 	}
// }
