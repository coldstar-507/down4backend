package backend

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
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

	if err := uploadNodeMedia(ctx, info.Image); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error uploading media in user initialization: %v\n", err)
	}

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
			ImageID:    info.Image.Identifier,
			Private:    false,
			Type:       "user",
			Friends:    make([]string, 0),
			Childs:     make([]string, 0),
			Parents:    make([]string, 0),
			Words:      make([]string, 0),
		}
		if err := tx.Set(nodeRef, nodeInfo); err != nil {
			return err
		}

		return nil
	}

	if err := s.FS.RunTransaction(ctx, createFirestoreNode); err != nil {
		s.NDBCKT.Object(info.Image.Identifier).Delete(ctx) // try deleting object, doesn't really matter if it fails
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
			Snips:    make(map[string]string),
			Messages: make(map[string]string),
			Payments: make(map[string]string),
		}
		return realtimeUserInfo, nil
	}

	if err := s.RTDB.NewRef("Users/"+info.Identifier).Transaction(ctx, createRealtimeUser); err != nil {
		nodeRef.Delete(ctx)
		s.NDBCKT.Object(info.Image.Identifier).Delete(ctx)
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

func HandlePaymentRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding payment body: %v\n", err)
	}

	payWrite := s.MSGBCKT.Object(req.Identifier).NewWriter(ctx)
	payWrite.Write(req.Payment)

	if err := payWrite.Close(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing payment to storage: %v\n", err)
	}

	tknChan := make(chan *string, len(req.Targets))
	errChan := make(chan *error, len(req.Targets))
	errChan2 := make(chan *error, len(req.Targets))
	ackChan := make(chan bool, len(req.Targets))
	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)
	go pushPay(ctx, req.Identifier, req.Targets, ackChan, errChan2)

	tokens := make([]string, 0)

	for i := 0; i < len(req.Targets)*2; i++ {
		select {
		case adr := <-tknChan:
			tokens = append(tokens, *adr)
		case err := <-errChan:
			log.Printf("error getting a target: %v\n", *err)
		case err2 := <-errChan2:
			log.Printf("error sending payment to a target: %v\n", *err2)
		case <-ackChan:
			continue
		}
	}

	title := "@" + req.Sender + " payed you!"

	if _, err := s.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
		},
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error mutlicating message: %v\n", err)
	}

	if err := updateActivity(ctx, req.Sender); err != nil {
		log.Printf("error updating activity for %s: %v\n", req.Sender, err)
	}
}

func HandlePingRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req PingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	tknChan := make(chan *string, len(req.Targets))
	errChan := make(chan *error, len(req.Targets))
	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)

	tokens := make([]string, 0)

	for range req.Targets {
		select {
		case adr := <-tknChan:
			tokens = append(tokens, *adr)
		case err := <-errChan:
			log.Printf("error getting a target: %v\n", *err)
		}
	}

	title := "@" + req.SenderID + " pinged!"
	body := req.Text

	log.Print(tokens)

	if _, err := s.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error mutlicating message: %v\n", err)
	}

	if err := updateActivity(ctx, req.SenderID); err != nil {
		log.Printf("error updating activity for %s: %v\n", req.SenderID, err)
	}
}

func HandleSnipRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req SnipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	var mediaWriter *storage.Writer
	if req.Media.Identifier != "" {
		mediaWriter = s.MSGBCKT.Object(req.Media.Identifier).NewWriter(ctx)
		if _, err := mediaWriter.Write(req.Media.Data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing snip media to message bucket: %v\n", err)
		}
		log.Printf("uploaded %v bytes of date no problem!\n", len(req.Media.Data))
	} else {
		log.Printf("We are not uploading any media")
	}

	err := s.RTDB.NewRef("Messages").Child(req.Message.MessageID).Set(ctx, req.Message)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing snip to rtdb: %v\n", err)
	}

	if mediaWriter != nil {
		if err := mediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing snip: %v\n", err)
		}
		if err := updateMediaMetadata(ctx, req.Media.Identifier, &req.Media.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Media.Identifier); err != nil {
				log.Printf("error deleting media at %s: %v\n", req.Media.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing snip: %v\n", err)
		}
	}

	tknChan := make(chan *string, len(req.Targets))
	errChan := make(chan *error, len(req.Targets))
	errChan2 := make(chan *error, len(req.Targets))
	ackChan := make(chan bool, len(req.Targets))

	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)
	go pushEvent(ctx, req.Message.MessageID, req.Targets, ackChan, errChan2)

	tokens := make([]string, 0)

	for i := 0; i < len(req.Targets)*2; i++ {
		select {
		case adr := <-tknChan:
			tokens = append(tokens, *adr)
		case err := <-errChan:
			log.Printf("error getting a target: %v\n", *err)
		case err2 := <-errChan2:
			log.Printf("error pushing message to a target: %v\n", *err2)
		case <-ackChan:
			continue
		}
	}

	title := "@" + req.Message.SenderID + " pinged you!"
	body := "&attachment"

	if _, err := s.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error mutlicating message: %v\n", err)
	}

	if err := updateActivity(ctx, req.Message.SenderID); err != nil {
		log.Printf("error updating activity for %s: %v\n", req.Message.SenderID, err)
	}
}

func HandleGroupRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req GroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	// this means, we are sending a media, without the upload
	if req.Message.MediaID != "" && req.Media.Identifier == "" {
		if _, err := getMediaMetadata(ctx, req.Media.Identifier); err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Printf("we will need an upload for this message: %v\n", err)
			return
		}
	}

	groupMediaObject := s.NDBCKT.Object(req.GroupMedia.Identifier)
	groupMediaWriter := groupMediaObject.NewWriter(ctx)
	groupMediaWriter.Write(req.GroupMedia.Data)

	var mediaWriter *storage.Writer
	if req.Media.Identifier != "" {
		mediaWriter = s.MSGBCKT.Object(req.Media.Identifier).NewWriter(ctx)
		mediaWriter.Write(req.Media.Data)

	}

	if err := groupMediaWriter.Close(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing group media: %v\n", err)
	}

	if _, err := groupMediaObject.Update(ctx, storage.ObjectAttrsToUpdate{Metadata: req.GroupMedia.Metadata}); err != nil {
		if err := groupMediaObject.Delete(ctx); err != nil {
			log.Printf("error deleting media at: %s, err = %v\n", req.GroupMedia.Identifier, err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error updating media metadata: %v\n", err)
	}

	if mediaWriter != nil {
		if err := mediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error closing media writer: %v\n", err)
		}
		if err := updateMediaMetadata(ctx, req.Media.Identifier, &req.Media.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Media.Identifier); err != nil {
				log.Printf("error deleting media at: %s, err = %v\n", req.Media.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error updating media metadata: %v\n", err)
		}
	}

	fullNode := FullNode{
		Identifier: req.GroupID,
		Name:       req.GroupName,
		Image:      req.GroupMedia,
		Group:      append(req.Targets, req.Message.SenderID),
		Private:    req.Private,
		Type:       "group",
	}

	fsNodesRef := s.FS.Collection("Nodes").Doc(req.GroupID)
	if _, err := fsNodesRef.Set(ctx, *(fullNode.ToFireStoreNode())); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing group info to firestore: %v\n", err)
	}

	msgRef := s.RTDB.NewRef("Messages").Child(req.Message.MessageID)
	if err := msgRef.Set(ctx, req.Message); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing message to RTDB: %v\n", err)
	}

	tknChan := make(chan *string, len(req.Targets))
	errChan := make(chan *error, len(req.Targets))
	errChan2 := make(chan *error, len(req.Targets))
	ackChan := make(chan bool, len(req.Targets))
	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)
	go pushEvent(ctx, req.Message.MessageID, req.Targets, ackChan, errChan2)

	tokens := make([]string, 0)

	for i := 0; i < len(req.Targets)*2; i++ {
		select {
		case adr := <-tknChan:
			tokens = append(tokens, *adr)
		case err := <-errChan:
			log.Printf("error getting a target: %v\n", *err)
		case err2 := <-errChan2:
			log.Printf("error pushing message to a target: %v\n", *err2)
		case <-ackChan:
			continue
		}
	}

	title := "@" + req.Message.SenderID + " formed " + req.GroupName
	var body string
	if req.Message.MessageID != "" {
		if req.Message.Text != "" {
			body = req.Message.Text + "\n" + "&attachment"
		} else {
			body = "&attachment"
		}
	} else {
		body = req.Message.Text
	}

	if _, err := s.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error mutlicating message: %v\n", err)
	}

	if err := updateActivity(ctx, req.Message.SenderID); err != nil {
		log.Printf("error updating activity for %s: %v\n", req.Message.SenderID, err)
	}

	marshalledFullNode, err := json.Marshal(fullNode)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error marshalling groupNode: %v\n", err)
	}

	if _, err := w.Write(marshalledFullNode); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing groupNode to response: %v\n", err)
	}
}

func HandleHyperchatRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req HyperchatRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	msgRef := s.RTDB.NewRef("Messages").Child(req.Message.MessageID)

	if req.Message.MediaID != "" && req.Media.Identifier == "" {
		if _, err := getMediaMetadata(ctx, req.Media.Identifier); err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Printf("we will need an upload for this message: %v\n", err)
			return
		}
	}

	reqBody, err := json.Marshal(req.WordPairs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error encoding wordpairs for http request: %v\n", err)
	}

	imageGenRes, err := http.Post("https://us-east1-down4-26ee1.cloudfunctions.net/imageGenerationRequest", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error making imageGen request: %v\n", err)
	}

	jsonImGen := make(map[string]string)
	if err := json.NewDecoder(imageGenRes.Body).Decode(&jsonImGen); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading hyperchat image body to Json: %v\n", err)
	}
	wp := strings.Split(jsonImGen["prompt"], " ")

	decodedImBuf := make([]byte, base64.StdEncoding.DecodedLen(len(jsonImGen["image"])))
	if _, err := base64.StdEncoding.Decode(decodedImBuf, []byte(jsonImGen["image"])); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading hyperchat image body to Json: %v\n", err)
	}

	var mediaWriter, mediaWriter2 *storage.Writer
	if req.Media.Identifier != "" {
		mediaWriter = s.MSGBCKT.Object(req.Media.Identifier).NewWriter(ctx)
		mediaWriter.Write(req.Media.Data)
	}

	hyperchatImageID, err := hexSha256(ctx, decodedImBuf)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error generating hyperchatID from hyperchatImage: %v\n", err)
	}

	mediaWriter2 = s.NDBCKT.Object(hyperchatImageID).NewWriter(ctx)
	mediaWriter2.Write(decodedImBuf)

	hcImageMD := map[string]string{
		"o":   req.Message.SenderID,
		"vid": "false",
		"shr": "true",
		"pto": "true",
		"trv": "false",
		"ar":  "1.0",
		"ts":  strconv.FormatInt(unixMilliseconds(), 10),
	}

	if err := mediaWriter2.Close(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing hyperchatMedia to storage: %v\n", err)
	} else {
		_, err = s.NDBCKT.Object(hyperchatImageID).Update(ctx, storage.ObjectAttrsToUpdate{Metadata: hcImageMD})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing hyperchatMedia metadata to storage: %v\n", err)
		}
	}

	if mediaWriter != nil {
		if err := mediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing media: %v\n", err)
		}

		if err := updateMediaMetadata(ctx, req.Media.Identifier, &req.Media.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Media.Identifier); err != nil {
				log.Printf("error deleting media at %s: %v\n", req.Media.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error updating media metadata: %v\n", err)
		}
	}

	if err := msgRef.Set(ctx, req.Message); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing message to RTDB: %v\n", err)
	}

	tknChan := make(chan *string, len(req.Targets))
	errChan := make(chan *error, len(req.Targets))
	errChan2 := make(chan *error, len(req.Targets))
	ackChan := make(chan bool, len(req.Targets))
	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)
	go pushEvent(ctx, req.Message.MessageID, req.Targets, ackChan, errChan2)

	tokens := make([]string, 0)

	for i := 0; i < len(req.Targets)*2; i++ {
		select {
		case adr := <-tknChan:
			tokens = append(tokens, *adr)
		case err := <-errChan:
			log.Printf("error getting a target: %v\n", *err)
		case err2 := <-errChan2:
			log.Printf("error pushing message to a target: %v\n", *err2)
		case <-ackChan:
			continue
		}
	}

	hyperchatNode := FullNode{
		Type:       "hyperchat",
		Name:       wp[0],
		Lastname:   wp[1],
		Identifier: req.Message.Root,
		Group:      append(req.Targets, req.Message.SenderID),
		Private:    true,
		Image: Down4Media{
			Data:       decodedImBuf,
			Identifier: hyperchatImageID,
			Metadata:   hcImageMD,
		},
	}

	fsNodeRef := s.FS.Collection("Nodes").Doc(hyperchatNode.Identifier)
	if _, err := fsNodeRef.Set(ctx, *(hyperchatNode.ToFireStoreNode())); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing hyperchat to firestore: %v\n", err)
	}

	marshalledHyperchatNode, err := json.Marshal(hyperchatNode)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error marshalling hyperchatNode: %v\n", err)
	}

	var body, title string
	title = wp[0] + " " + wp[1]
	if req.Media.Identifier != "" {
		if req.Message.Text != "" {
			body = "@" + req.Message.SenderID + "\n" + req.Message.Text + "\n" + "&attachment"
		} else {
			body = "@" + req.Message.SenderID + "\n" + "&attachment"
		}
	} else {
		body = "@" + req.Message.SenderID + "\n" + req.Message.Text
	}

	if _, err := s.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error mutlicating message: %v\n", err)
	}

	if err := updateActivity(ctx, req.Message.SenderID); err != nil {
		log.Printf("error updating activity for %s: %v\n", req.Message.SenderID, err)
	}

	if _, err := w.Write(marshalledHyperchatNode); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing marshalledHyperchatNode to response: %v\n", err)
	}
}

func HandleChatRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req ChatRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	msgRef := s.RTDB.NewRef("Messages").Child(req.Message.MessageID)

	if req.Message.MediaID != "" && req.Media.Identifier == "" {
		if _, err := getMediaMetadata(ctx, req.Media.Identifier); err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Printf("we will need an upload for this message: %v\n", err)
			return
		}
	}

	var mediaWriter *storage.Writer
	if req.Media.Identifier != "" {
		mediaWriter = s.MSGBCKT.Object(req.Media.Identifier).NewWriter(ctx)
		mediaWriter.Write(req.Media.Data)
	}

	if mediaWriter != nil {
		if err := mediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing media: %v\n", err)
		}
		if err := updateMediaMetadata(ctx, req.Media.Identifier, &req.Media.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Media.Identifier); err != nil {
				log.Printf("error deleting media at %s: %v\n", req.Media.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error updating media metadata: %v\n", err)
		}
	}

	if err := msgRef.Set(ctx, req.Message); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing message to RTDB: %v\n", err)
	}

	tknChan := make(chan *string, len(req.Targets))
	errChan := make(chan *error, len(req.Targets))
	errChan2 := make(chan *error, len(req.Targets))
	ackChan := make(chan bool, len(req.Targets))
	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)
	go pushEvent(ctx, req.Message.MessageID, req.Targets, ackChan, errChan2)

	tokens := make([]string, 0)

	for i := 0; i < len(req.Targets)*2; i++ {
		select {
		case adr := <-tknChan:
			tokens = append(tokens, *adr)
		case err := <-errChan:
			log.Printf("error getting a target: %v\n", *err)
		case err2 := <-errChan2:
			log.Printf("error pushing message to a target: %v\n", *err2)
		case <-ackChan:
			continue
		}
	}

	var body, title string
	if req.GroupName != "" {
		title = req.GroupName
		if req.Media.Identifier != "" {
			if req.Message.Text != "" {
				body = "@" + req.Message.SenderID + "\n" + req.Message.Text + "\n" + "&attachment"
			} else {
				body = "@" + req.Message.SenderID + "\n" + "&attachment"
			}
		} else {
			body = "@" + req.Message.SenderID + "\n" + req.Message.Text
		}

	} else {
		title = "@" + req.Message.SenderID
		if req.Media.Identifier != "" {
			if req.Message.Text != "" {
				body = req.Message.Text + "\n" + "&attachment"
			} else {
				body = "&attachment"
			}
		} else {
			body = req.Message.Text
		}
	}

	if _, err := s.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error mutlicating message: %v\n", err)
	}

	if err := updateActivity(ctx, req.Message.SenderID); err != nil {
		log.Printf("error updating activity for %s: %v\n", req.Message.SenderID, err)
	}
}

func GetMessageMedia(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading body bytes: %v\n", err)
	}
	mediaID := string(bodyBytes)
	d4Media, err := getMessageMedia(ctx, mediaID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error getting message media: %v\n", err)
	}
	if err := json.NewEncoder(w).Encode(*d4Media); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error encoding media to response: %v\n", err)
	}
	w.WriteHeader(http.StatusOK)
}

func GetMediaMetadata(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading body bytes: %v\n", err)
	}
	mediaID := string(bodyBytes)
	metadata, err := getMediaMetadata(ctx, mediaID)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
	} else {
		marshalled, err := json.Marshal(*metadata)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Write(marshalled)
			w.WriteHeader(http.StatusOK)
		}
	}
}

func GetPayment(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("error reading bodyBytes: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	paymentID := string(bodyBytes)

	rdr, err := s.MSGBCKT.Object(paymentID).NewReader(ctx)
	if err != nil {
		log.Fatalf("error creating reader for payment: %v\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	paymentData, err := io.ReadAll(rdr)
	if err != nil {
		log.Fatalf("error reading payment id:%v, err:%v\n", paymentID, err)
		w.WriteHeader(http.StatusInternalServerError)
	}

	if _, err = w.Write(paymentData); err != nil {
		log.Fatalf("error writing payment id:%v, err:%v\n", paymentID, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func getMessageMedia(ctx context.Context, mediaID string) (*Down4Media, error) {
	obj := s.MSGBCKT.Object(mediaID)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		log.Fatalf("error getting metadata of media: %v\n", err)
		return nil, err
	}
	rdr, err := obj.NewReader(ctx)
	if err != nil {
		log.Fatalf("error creating reader for media: %v\n", err)
		return nil, err
	}
	mediaData, err := io.ReadAll(rdr)
	if err != nil {
		log.Fatalf("error reading media: %v\n", err)
		return nil, err
	}
	down4Media := Down4Media{
		Identifier: mediaID,
		Data:       mediaData,
		Metadata:   attrs.Metadata,
	}
	return &down4Media, nil
}

func deleteMedia(ctx context.Context, mediaID string) error {
	if err := s.MSGBCKT.Object(mediaID).Delete(ctx); err != nil {
		return err
	}
	return nil
}

func getMessagingTokens(ctx context.Context, ids []string, ch chan *string, ech chan *error) {
	for _, id := range ids {
		id_ := id
		go func() {
			userTokenRef := s.RTDB.NewRef("Users/" + id_ + "/tkn/")
			var token string
			if err := userTokenRef.Get(ctx, &token); err != nil {
				ech <- &err
			} else {
				ch <- &token
			}
		}()
	}
}

func getMediaMetadata(ctx context.Context, mediaID string) (*map[string]string, error) {
	metadata, err := s.MSGBCKT.Object(mediaID).Attrs(ctx)
	if err != nil {
		return nil, err
	}
	return &metadata.Metadata, nil
}

func updateMediaMetadata(ctx context.Context, mediaID string, md *map[string]string) error {
	if _, err := s.MSGBCKT.Object(mediaID).Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: *md,
	}); err != nil {
		return err
	}
	return nil
}

func updateActivity(ctx context.Context, uid string) error {
	err := s.RTDB.NewRef("Users/"+uid+"/ac").Set(ctx, unixMilliseconds())
	return err
}

func pushEvent(ctx context.Context, objectID string, targets []string, ch chan bool, ech chan *error) {
	for _, target := range targets {
		target_ := target
		dbRef := s.RTDB.NewRef("Users").Child(target_).Child("M").Child(objectID)
		if err := dbRef.Set(ctx, ""); err != nil {
			ech <- &err
		} else {
			ch <- true
		}
	}
}

func pushPay(ctx context.Context, objectID string, targets []string, ch chan bool, ech chan *error) {
	for _, target := range targets {
		target_ := target
		dbRef := s.RTDB.NewRef("Users").Child(target_).Child("M").Child(objectID)
		if err := dbRef.Set(ctx, "p"); err != nil {
			ech <- &err
		} else {
			ch <- true
		}
	}
}

func hexSha256(ctx context.Context, data []byte) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write(data); err != nil {
		return "", err
	}

	hashed := hash.Sum(make([]byte, 0))

	return hex.EncodeToString(hashed), nil
}

func uploadNodeMedia(ctx context.Context, media Down4Media) error {
	obj := s.NDBCKT.Object(media.Identifier)
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

func getNode(ctx context.Context, id string, nodeChan chan *FullNode, errChan chan *error) {
	var fsn FireStoreNode
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

	d4media, err := getNodeMedia(ctx, fsn.ImageID)
	if err != nil {
		errChan <- &err
		log.Printf("error reading image data from bucket reader: %v\n", err)
		return
	}

	node := FullNode{
		Identifier: fsn.Identifier,
		Neuter:     fsn.Neuter,
		Type:       fsn.Type,
		Name:       fsn.Name,
		Lastname:   fsn.Lastname,
		Image:      *d4media,
		Messages:   fsn.Messages,
		Group:      fsn.Group,
		Words:      fsn.Words,
		Friends:    fsn.Friends,
		Admins:     fsn.Admins,
		Childs:     fsn.Childs,
		Parents:    fsn.Parents,
	}

	nodeChan <- &node
}

func getNodeMedia(ctx context.Context, id string) (*Down4Media, error) {
	obj := s.NDBCKT.Object(id)
	rdr, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}

	mediaData, err := io.ReadAll(rdr)
	if err != nil {
		return nil, err
	}

	mediaAttrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, err
	}

	down4Media := Down4Media{
		Identifier: id,
		Data:       mediaData,
		Metadata:   mediaAttrs.Metadata,
	}
	return &down4Media, nil
}

func unixMilliseconds() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
