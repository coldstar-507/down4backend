package messages

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
)

type MessageServer struct {
	FS      *firestore.Client
	RTDB    *db.Client
	MSGBCKT *storage.BucketHandle
	NDBCKT  *storage.BucketHandle
	MSGR    *messaging.Client
	URLOPTS *storage.SignedURLOptions
}

var ms MessageServer

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

	ms = MessageServer{
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

func HandlePayment(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req Down4InternetPayment
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding payment body: %v\n", err)
	}

	payWrite := ms.MSGBCKT.Object(req.PaymentID).NewWriter(ctx)
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
	go pushEvent(ctx, req.PaymentID, req.Targets, Pay, ackChan, errChan2)

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

	title := "@" + req.Sender + "payed you"

	if _, err := ms.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
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

	title := "@" + req.Message.SenderID + " pinged!"
	body := req.Message.Text

	if _, err := ms.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
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

func HandleSnipRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req SnipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	mediaWriter := ms.MSGBCKT.Object(req.Message.Media.Identifier).NewWriter(ctx)
	mediaWriter.Write(req.Message.Media.Data)

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

	if err := mediaWriter.Close(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing snip: %v\n", err)
	}

	if err := updateMediaMetadata(ctx, req.Message.Media.Identifier, &req.Message.Media.Metadata); err != nil {
		if err := deleteMedia(ctx, req.Message.Media.Identifier); err != nil {
			log.Printf("error deleting media at %s: %v\n", req.Message.Media.Identifier, err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing snip: %v\n", err)
	}

	title := "@" + req.SenderID + " pinged!"
	body := "&attachment"

	if _, err := ms.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
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

	if req.Message.Media.Identifier != "" && !req.WithUpload {
		if _, err := getMediaMetadata(ctx, req.Message.Media.Identifier); err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Printf("we will need an upload for this message: %v\n", err)
			return
		}
	}

	var groupMediaWriter, mediaWriter *storage.Writer
	groupMediaWriter = ms.MSGBCKT.Object(req.GroupMedia.Identifier).NewWriter(ctx)
	groupMediaWriter.Write(req.GroupMedia.Data)
	if req.WithUpload {
		mediaWriter = ms.MSGBCKT.Object(req.Message.Media.Identifier).NewWriter(ctx)
		mediaWriter.Write(req.Message.Media.Data)

	}

	if err := groupMediaWriter.Close(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing group media: %v\n", err)
		if err := updateMediaMetadata(ctx, req.GroupMedia.Identifier, &req.GroupMedia.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Message.Media.Identifier); err != nil {
				log.Printf("error deleting media at: %s, err = %v\n", req.GroupMedia.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error updating media metadata: %v\n", err)
		}
	}

	if mediaWriter != nil {
		if err := mediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error closing media writer: %v\n", err)
		}
		if err := updateMediaMetadata(ctx, req.Message.Media.Identifier, &req.Message.Media.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Message.Media.Identifier); err != nil {
				log.Printf("error deleting media at: %s, err = %v\n", req.Message.Media.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error updating media metadata: %v\n", err)
		}
	}

	if _, err := ms.FS.Collection("Nodes").Doc(req.GroupID).Set(ctx, map[string]interface{}{
		"id": req.GroupID,
		"nm": req.GroupName,
		"im": req.GroupMedia.Identifier,
		"g":  append(req.Targets, req.Message.SenderID),
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing group info to firestore: %v\n", err)
	}

	tknChan := make(chan *string, len(req.Targets))
	errChan := make(chan *error, len(req.Targets))
	errChan2 := make(chan *error, len(req.Targets))
	ackChan := make(chan bool, len(req.Targets))
	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)
	go pushEvent(ctx, req.Message.MessageID, req.Targets, Chat, ackChan, errChan2)

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

	if _, err := ms.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
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

func HandleHyperchatRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req HyperchatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	if req.Message.Media.Identifier != "" && !req.WithUpload {
		if _, err := getMediaMetadata(ctx, req.Message.Media.Identifier); err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Fatalf("we will need an upload for this message: %v\n", err)
		}
	}

	var hyperchatMediaWriter, mediaWriter *storage.Writer
	hyperchatMediaWriter = ms.MSGBCKT.Object(req.GroupNode.Identifier).NewWriter(ctx)
	hyperchatMediaWriter.Write(req.GroupNode.Image.Data)
	if req.WithUpload {
		mediaWriter = ms.MSGBCKT.Object(req.Message.Media.Identifier).NewWriter(ctx)
		mediaWriter.Write(req.Message.Media.Data)

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

	if err := hyperchatMediaWriter.Close(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing group media: %v\n", err)
		if err := updateMediaMetadata(ctx, req.GroupNode.Image.Identifier, &req.GroupNode.Image.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Message.Media.Identifier); err != nil {
				log.Printf("error deleting media at %s: %v\n", req.GroupNode.Image.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error updating media metadata: %v\n", err)
		}
	}

	if mediaWriter != nil {
		if err := mediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error closing media writer: %v\n", err)
		}
		if err := updateMediaMetadata(ctx, req.Message.Media.Identifier, &req.Message.Media.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Message.Media.Identifier); err != nil {
				log.Printf("error deleting media at %s: %v\n", req.Message.Media.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error updating media metadata: %v\n", err)
		}
	}

	title := "Hyperchat by " + req.Message.SenderName + " " + req.Message.SenderLastName
	body := req.Message.Text

	if _, err := ms.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
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

func HandleChatRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req ChatRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	msgRef := ms.RTDB.NewRef("Messages").Child(req.Message.MessageID)

	if req.Message.Media.Identifier != "" && !req.WithUpload {
		if _, err := getMediaMetadata(ctx, req.Message.Media.Identifier); err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Printf("we will need an upload for this message: %v\n", err)
			return
		}
	}

	var mediaWriter *storage.Writer
	if req.WithUpload {
		mediaWriter = ms.MSGBCKT.Object(req.Message.Media.Identifier).NewWriter(ctx)
		mediaWriter.Write(req.Message.Media.Data)
	}

	if mediaWriter != nil {
		if err := mediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing media: %v\n", err)
		}
		if err := updateMediaMetadata(ctx, req.Message.Media.Identifier, &req.Message.Media.Metadata); err != nil {
			if err := deleteMedia(ctx, req.Message.Media.Identifier); err != nil {
				log.Printf("error deleting media at %s: %v\n", req.Message.Media.Identifier, err)
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error updating media metadata: %v\n", err)
		}
	}

	if err := msgRef.Set(ctx, *req.Message.ToRTDB()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing message to RTDB: %v\n", err)
	}

	tknChan := make(chan *string, len(req.Targets))
	errChan := make(chan *error, len(req.Targets))
	errChan2 := make(chan *error, len(req.Targets))
	ackChan := make(chan bool, len(req.Targets))
	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)
	go pushEvent(ctx, req.Message.MessageID, req.Targets, Chat, ackChan, errChan2)

	tokens := make([]string, 0)

	for range len(req.Targets) * 2 {
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
		if req.Message.Media.Identifier != "" {
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
		if req.Message.Media.Identifier != "" {
			if req.Message.Text != "" {
				body = req.Message.Text + "\n" + "&attachment"
			} else {
				body = "&attachment"
			}
		} else {
			body = req.Message.Text
		}
	}

	if _, err := ms.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
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

func getMessageMedia(ctx context.Context, mediaID string) (*Down4Media, error) {
	obj := ms.MSGBCKT.Object(mediaID)
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
	if err := ms.MSGBCKT.Object(mediaID).Delete(ctx); err != nil {
		return err
	}
	return nil
}

func getMessagingTokens(ctx context.Context, ids []string, ch chan *string, ech chan *error) {
	for _, id := range ids {
		id_ := id
		go func() {
			userTokenRef := ms.RTDB.NewRef("Users/" + id_ + "/tkn/")
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
	metadata, err := ms.MSGBCKT.Object(mediaID).Attrs(ctx)
	if err != nil {
		return nil, err
	}
	return &metadata.Metadata, nil
}

func updateMediaMetadata(ctx context.Context, mediaID string, md *map[string]string) error {
	if _, err := ms.MSGBCKT.Object(mediaID).Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: *md,
	}); err != nil {
		return err
	}
	return nil
}

func updateActivity(ctx context.Context, uid string) error {
	err := ms.RTDB.NewRef("Users/"+uid+"/ac").Set(ctx, time.Now().Unix())
	return err
}

const (
	Chat = "c"
	Snip = "s"
	Pay  = "p"
)

func pushEvent(ctx context.Context, objectID string, targets []string, T string, ch chan bool, ech chan *error) {
	for _, target := range targets {
		target_ := target
		dbRef := ms.RTDB.NewRef("Users").Child(target_).Child(T).Child(objectID)
		if err := dbRef.Set(ctx, ""); err != nil {
			ech <- &err
		} else {
			ch <- true
		}
	}
}

// func HandleMessageRequest(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()

// 	var req MessageRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body: %v\n", err)
// 	}

// 	if req.Message.Media.Identifier != "" && !req.WithUpload {
// 		if _, err := getMediaMetadata(ctx, req.Message.Media.Identifier); err != nil {
// 			w.WriteHeader(http.StatusNoContent)
// 			log.Fatalf("error, we will need an upload for this message: %v\n", err)
// 		}
// 	}

// 	var groupMediaWriter *storage.Writer
// 	if req.GroupNode.Identifier != "" {
// 		groupMediaWriter = ms.MSGBCKT.Object(req.GroupNode.Image.Identifier).NewWriter(ctx)
// 		groupMediaWriter.Write(req.GroupNode.Image.Data)
// 	}

// 	var mediaWriter *storage.Writer
// 	if req.Message.Media.Identifier != "" && req.WithUpload {
// 		mediaWriter = ms.MSGBCKT.Object(req.Message.Media.Identifier).NewWriter(ctx)
// 		mediaWriter.Write(req.Message.Media.Data)
// 	}

// 	tknChan := make(chan *string, len(req.Targets))
// 	errChan := make(chan *error, len(req.Targets))
// 	go getMessagingTokens(ctx, req.Targets, tknChan, errChan)

// 	tokens := make([]string, 0)

// 	for range req.Targets {
// 		select {
// 		case adr := <-tknChan:
// 			tokens = append(tokens, *adr)
// 		case err := <-errChan:
// 			log.Printf("error getting a target: %v\n", *err)
// 		}
// 	}

// 	if groupMediaWriter != nil {
// 		if err := groupMediaWriter.Close(); err != nil {
// 			w.WriteHeader(http.StatusInternalServerError)
// 			log.Fatalf("error writing group media: %v\n", err)
// 		}
// 		if err := updateMediaMetadata(ctx, req.GroupNode.Image.Identifier, &req.GroupNode.Image.Metadata); err != nil {
// 			deleteMedia(ctx, req.GroupNode.Identifier)
// 			w.WriteHeader(http.StatusInternalServerError)
// 			log.Fatalf("error writing group media metadata: %v\n", err)
// 		}
// 	}

// 	if mediaWriter != nil {
// 		if err := mediaWriter.Close(); err != nil {
// 			w.WriteHeader(http.StatusInternalServerError)
// 			log.Fatalf("error writing media: %v\n", err)
// 		}
// 		if err := updateMediaMetadata(ctx, req.Message.Media.Identifier, &req.Message.Media.Metadata); err != nil {
// 			deleteMedia(ctx, req.GroupNode.Identifier)
// 			deleteMedia(ctx, req.Message.Media.Identifier)
// 			w.WriteHeader(http.StatusInternalServerError)
// 			log.Fatalf("error writing group media metadata: %v\n", err)
// 		}
// 	}

// 	var title, body string
// 	if req.IsGroup {
// 		title = req.Message.SenderName + " created a group"
// 	} else if req.IsHyperchat {
// 		title = req.Message.SenderName + " created an hyperchat"
// 	}

// 	if req.Message.Media.Identifier != "" {
// 		if req.Message.Text != "" {
// 			body = req.Message.Text + "\n" + "&attachment"
// 		} else {
// 			body = "&attachment"
// 		}
// 	} else {
// 		body = req.Message.Text
// 	}

// 	if _, err := ms.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
// 		Tokens: tokens,
// 		Data:   *req.ToNotification(),
// 		Notification: &messaging.Notification{
// 			Title: title,
// 			Body:  body,
// 		},
// 	}); err != nil {
// 		deleteMedia(ctx, req.GroupNode.Identifier)
// 		deleteMedia(ctx, req.Message.Media.Identifier)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error mutlicating message: %v\n", err)
// 	}

// 	w.WriteHeader(http.StatusOK)

// 	updateActivity(ctx, req.Message.SenderID)
// }
