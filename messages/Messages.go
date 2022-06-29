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

func HandleMessageRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body: %v\n", err)
	}

	if req.Message.Media.Identifier != "" && !req.WithUpload {
		if _, err := getMediaMetadata(ctx, req.Message.Media.Identifier); err != nil {
			w.WriteHeader(http.StatusNoContent)
			log.Fatalf("error, we will need an upload for this message: %v\n", err)
		}
	}

	var groupMediaWriter *storage.Writer
	if req.GroupNode.Identifier != "" {
		groupMediaWriter = ms.MSGBCKT.Object(req.GroupNode.Image.Identifier).NewWriter(ctx)
		groupMediaWriter.Write(req.GroupNode.Image.Data)
	}

	var mediaWriter *storage.Writer
	if req.Message.Media.Identifier != "" && req.WithUpload {
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

	if groupMediaWriter != nil {
		if err := groupMediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing group media: %v\n", err)
		}
		if err := updateMediaMetadata(ctx, req.GroupNode.Image.Identifier, &req.GroupNode.Image.Metadata); err != nil {
			deleteMedia(ctx, req.GroupNode.Identifier)
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing group media metadata: %v\n", err)
		}
	}

	if mediaWriter != nil {
		if err := mediaWriter.Close(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing media: %v\n", err)
		}
		if err := updateMediaMetadata(ctx, req.Message.Media.Identifier, &req.Message.Media.Metadata); err != nil {
			deleteMedia(ctx, req.GroupNode.Identifier)
			deleteMedia(ctx, req.Message.Media.Identifier)
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error writing group media metadata: %v\n", err)
		}
	}

	var title, body string
	if req.IsGroup {
		title = req.Message.SenderName + " created a group"
	} else if req.IsHyperchat {
		title = req.Message.SenderName + " created an hyperchat"
	}

	if req.Message.Media.Identifier != "" {
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
		Data:   *req.ToNotification(),
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
	}); err != nil {
		deleteMedia(ctx, req.GroupNode.Identifier)
		deleteMedia(ctx, req.Message.Media.Identifier)
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error mutlicating message: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)

	updateActivity(ctx, req.Message.SenderID)
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
			userTokenRef := ms.RTDB.NewRef(id_ + "/tkn/")
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
	err := ms.RTDB.NewRef(uid+"/ac").Set(ctx, time.Now().Unix())
	return err
}

// func HandleFriendRequestAccepted(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	var fra FriendRequestAccepted

// 	if err := json.NewDecoder(r.Body).Decode(&fra); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body: %v\n", err)
// 	}

// 	err := ms.FS.RunTransaction(ctx, func(ctx context.Context, t *firestore.Transaction) error {
// 		ref0 := ms.FS.Collection("Nodes").Doc(fra.Sender)
// 		ref1 := ms.FS.Collection("Nodes").Doc(fra.Accepter)
// 		var fsNode0, fsNode1 FireStoreNode
// 		snaps, err := t.GetAll([]*firestore.DocumentRef{ref0, ref1})
// 		if err != nil {
// 			return err
// 		}
// 		if err := snaps[0].DataTo(&fsNode0); err != nil {
// 			return err
// 		}
// 		if err := snaps[1].DataTo(&fsNode1); err != nil {
// 			return err
// 		}
// 		var alreadyIn0, alreadyIn1 bool = false, false
// 		for _, v := range fsNode0.Friends {
// 			if v == fra.Accepter {
// 				alreadyIn0 = true
// 			}
// 		}
// 		for _, v := range fsNode1.Friends {
// 			if v == fra.Sender {
// 				alreadyIn1 = true
// 			}
// 		}
// 		if !alreadyIn0 {
// 			fsNode0.Friends = append(fsNode0.Friends, fra.Accepter)
// 		}
// 		if !alreadyIn1 {
// 			fsNode1.Friends = append(fsNode1.Friends, fra.Sender)
// 		}
// 		if err := t.Set(ref0, fsNode0); err != nil {
// 			return err
// 		}
// 		if err := t.Set(ref1, fsNode1); err != nil {
// 			return err
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error running transaction: %v\n", err)
// 	}

// 	var senderTkn string
// 	if err := ms.RTDB.NewRef(fra.Sender+"/tkn").Get(ctx, &senderTkn); err != nil {}

// 	w.WriteHeader(http.StatusOK)
// }

// func HandleFriendRequest(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()
// 	var fr FriendRequest

// 	if err := json.NewDecoder(r.Body).Decode(&fr); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body: %v\n", err)
// 	}

// 	errChan := make(chan *error, len(fr.Targets))
// 	tknChan := make(chan *string, len(fr.Targets))

// 	go getMessagingTokens(ctx, fr.Targets, tknChan, errChan)

// 	tokens := make([]string, 0)

// 	select {
// 	case tkn := <-tknChan:
// 		tokens = append(tokens, *tkn)
// 	case err := <-errChan:
// 		log.Printf("error getting a token: %v\n", *err)
// 	}

// 	var body string
// 	if fr.RequesterLastName == "" {
// 		body = fr.RequesterName + " wants to be your friend"
// 	} else {
// 		body = fr.RequesterName + " " + fr.RequesterLastName + " wants to be your friend"
// 	}

// 	ms.MSGR.SendMulticast(ctx, &messaging.MulticastMessage{
// 		Tokens: tokens,
// 		Notification: &messaging.Notification{
// 			Body: body,
// 		},
// 		Data: map[string]string{
// 			"t":  "friendRequest",
// 			"id": fr.RequesterID,
// 			"nm": fr.RequesterName,
// 			"ln": fr.RequesterLastName,
// 		},
// 	})
// }
