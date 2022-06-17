package messages

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
)

type MessageServer struct {
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

	stor, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("error initializing storage: %v\n", err)
	}

	msgbckt := stor.Bucket("down4-26ee1-messages")
	ndbckt := stor.Bucket("down4-26ee1-nodes")

	ms = MessageServer{
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

	var groupMediaWriter *storage.Writer
	if req.GroupNode.Identifier != "" {
		groupMediaWriter = ms.MSGBCKT.Object(req.GroupNode.Image.Identifier).NewWriter(ctx)
		groupMediaWriter.Write(req.GroupNode.Image.Data)
	}

	var mediaWriter *storage.Writer
	if req.Message.Media.Identifier != "" {
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

func GetMedia(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading body bytes: %v\n", err)
	}
	mediaID := string(bodyBytes)
	obj := ms.MSGBCKT.Object(mediaID)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error getting metadata of media: %v\n", err)
	}
	rdr, err := obj.NewReader(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error creating reader for media: %v\n", err)
	}
	mediaData, err := io.ReadAll(rdr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading media: %v\n", err)
	}
	down4Media := Down4Media{
		Identifier: mediaID,
		Data:       mediaData,
		Metadata:   attrs.Metadata,
	}
	if err := json.NewEncoder(w).Encode(down4Media); err != nil {
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
	metadata := getMediaMetadata(ctx, mediaID)
	if metadata == nil {
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

func GetMessageMedia(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	byteID, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading body of request in GetMessageMedia: %v\n", err)
	}

	messageID := string(byteID)

	rd, err := ms.MSGBCKT.Object(messageID).NewReader(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error creating reading for bucket in GetMessageMedia: %v\n", err)
	}

	mediaData, err := io.ReadAll(rd)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading media data from reader in GetMessageMedia: %v\n", err)
	}

	if _, err = w.Write(mediaData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing media data on response in GetMessageMedia: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
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

func getMediaMetadata(ctx context.Context, mediaID string) *map[string]string {
	metadata, err := ms.MSGBCKT.Object(mediaID).Attrs(ctx)
	if err != nil {
		return nil
	}
	return &metadata.Metadata
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

// func HandleChatRequestWithMediaUpload(w http.ResponseWriter, r *http.Request) {

// 	ctx := context.Background()

// 	var chatReq ChatRequestWithMediaUpload
// 	err := json.NewDecoder(r.Body).Decode(&chatReq)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body message request: %v\n", err)
// 	}

// 	mediaWriter := ms.MSGBCKT.Object(chatReq.Message.Media.Identifier).NewWriter(ctx)
// 	mediaWriter.Write(chatReq.Message.Media.Data)

// 	errChan := make(chan *error, len(chatReq.Targets))
// 	tknChan := make(chan *string, len(chatReq.Targets))

// 	go getMessagingTokens(ctx, chatReq.Targets, tknChan, errChan)
// 	tokens := make([]string, 0)

// 	for range chatReq.Targets {
// 		select {
// 		case adr := <-tknChan:
// 			tokens = append(tokens, *adr)
// 		case err := <-errChan:
// 			log.Printf("error getting a target: %v\n", *err)
// 		}
// 	}

// 	if err := mediaWriter.Close(); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error writing data to bucket on writer close: %v\n", err)
// 	}
// 	if err := updateMediaMetadata(ctx, chatReq.Message.Media.Identifier, &chatReq.Message.Media.Metadata); err != nil {
// 		deleteMedia(ctx, chatReq.Message.Media.Identifier)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error writing metadata to bucket: %v\n", err)
// 	}

// 	if _, err := ms.MSGR.SendMulticast(
// 		ctx,
// 		&messaging.MulticastMessage{
// 			Tokens: tokens,
// 			Data:   *chatReq.ToNotification(),
// 			Notification: &messaging.Notification{
// 				Title: chatReq.Message.SenderName,
// 				Body:  chatReq.Message.Text},
// 		},
// 	); err != nil {
// 		deleteMedia(ctx, chatReq.Message.Media.Identifier)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error multicasting message: %v\n", err)
// 	}

// 	w.WriteHeader(http.StatusOK)

// 	updateActivity(ctx, chatReq.Message.SenderID)
// }

// func HandleHyperchatRequestWithMediaUpload(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()

// 	var req HyperchatRequestWithMediaUpload
// 	err := json.NewDecoder(r.Body).Decode(&req)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body in message request: %v\n", err)
// 	}
// 	mediaData, err := base64.StdEncoding.DecodeString(msgReq.Media)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding base64 media in message request: %v\n", err)
// 	}

// 	mediaWriter := ms.MSGBCKT.Object(req.Message.Media.Identifier).NewWriter(ctx)
// 	mediaWriter.Write(req.Message.Media.Data)

// 	hcImageWriter := ms.MSGBCKT.Object(req.Hyperchat.Identifier).NewWriter(ctx)
// 	hcImageWriter.Write(req.Hyperchat.Image.Data)

// 	errChan := make(chan *error, len(req.Targets))
// 	tknChan := make(chan *string, len(req.Targets))

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

// 	if err := mediaWriter.Close(); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error writing message media data to bucket: %v\n", err)
// 	}
// 	if err := updateMediaMetadata(ctx, req.Message.Media.Identifier, &req.Message.Media.Metadata); err != nil {
// 		deleteMedia(ctx, req.Message.MessageID)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error updating message media metadata: %v\n", err)
// 	}

// 	if err := hcImageWriter.Close(); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error writing hyperchat image data to bucket: %v\n", err)
// 	}
// 	if err := updateMediaMetadata(ctx, req.Hyperchat.Image.Identifier, &req.Hyperchat.Image.Metadata); err != nil {
// 		deleteMedia(ctx, req.Message.MessageID)
// 		deleteMedia(ctx, req.Hyperchat.Image.Identifier)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error updating message media metadata: %v\n", err)
// 	}

// 	var title string
// 	if req.Message.SenderLastName == "" {
// 		title = req.Message.SenderName + " created an hyperchat"
// 	} else {
// 		title = req.Message.SenderName + " " + req.Message.SenderLastName + " created an hyperchat"
// 	}
// 	if _, err := ms.MSGR.SendMulticast(
// 		ctx,
// 		&messaging.MulticastMessage{
// 			Tokens: tokens,
// 			Data:   *req.ToNotification(),
// 			Notification: &messaging.Notification{
// 				Title: title,
// 				Body:  req.Message.Text},
// 		},
// 	); err != nil {
// 		deleteMedia(ctx, req.Message.MessageID)
// 		deleteMedia(ctx, req.Hyperchat.Image.Identifier)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error multicasting message: %v\n", err)
// 	}

// 	w.WriteHeader(http.StatusOK)

// 	updateActivity(ctx, req.Message.SenderID)
// }

// func HandleHyperchatRequestWithNotifOnly(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()

// 	var req HyperchatRequestWithNotifOnly
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body message request: %v\n", err)
// 	}

// 	hcMediaWriter := ms.MSGBCKT.Object(req.Hyperchat.Image.Identifier).NewWriter(ctx)
// 	hcMediaWriter.Write(req.Hyperchat.Image.Data)

// 	errChan := make(chan *error, len(req.Targets))
// 	tknChan := make(chan *string, len(req.Targets))

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

// 	if err := hcMediaWriter.Close(); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error writing hyperchat media to storage: %v\n", err)
// 	}

// 	if err := updateMediaMetadata(ctx, req.Hyperchat.Image.Identifier, &req.Hyperchat.Image.Metadata); err != nil {
// 		deleteMedia(ctx, req.Hyperchat.Image.Identifier)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error error writing hyperchat media metadata to storage: %v\n", err)
// 	}

// 	var title string
// 	if req.Notif["ln"] != "" {
// 		title = req.Notif["sdrnm"] + " created an hyperchat:"
// 	} else {
// 		title = req.Notif["sdrnm"] + " " + req.Notif["sdrln"] + " created an hyperchat:"
// 	}

// 	req.Notif["t"] = "hc"
// 	req.Notif["hcnm"] = req.Hyperchat.Name
// 	req.Notif["hcln"] = req.Hyperchat.LastName
// 	req.Notif["hcid"] = req.Hyperchat.Identifier
// 	req.Notif["hcmid"] = req.Hyperchat.Image.Identifier

// 	if _, err := ms.MSGR.SendMulticast(
// 		ctx,
// 		&messaging.MulticastMessage{
// 			Tokens: tokens,
// 			Data:   req.Notif,
// 			Notification: &messaging.Notification{
// 				Title: title,
// 				Body:  req.Notif["txt"]},
// 		},
// 	); err != nil {
// 		deleteMedia(ctx, req.Hyperchat.Image.Identifier)
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error multicasting message: %v\n", err)
// 	}

// 	w.WriteHeader(http.StatusOK)

// 	updateActivity(ctx, req.Notif["sdrid"])
// }

// func HandleChatRequestWithNotifOnly(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()

// 	var req ChatRequestWithNotifOnly
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body message request: %v\n", err)
// 	}

// 	errChan := make(chan *error, len(req.Targets))
// 	tknChan := make(chan *string, len(req.Targets))

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

// 	var title string
// 	if req.Notif["ln"] != "" {
// 		title = req.Notif["sdrnm"] + " messaged you"
// 	} else {
// 		title = req.Notif["sdrnm"] + " " + req.Notif["sdrln"] + " messaged you"
// 	}
// 	req.Notif["t"] = "cht"
// 	if _, err := ms.MSGR.SendMulticast(
// 		ctx,
// 		&messaging.MulticastMessage{
// 			Tokens: tokens,
// 			Data:   req.Notif,
// 			Notification: &messaging.Notification{
// 				Title: title,
// 				Body:  req.Notif["txt"]},
// 		},
// 	); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error multicasting message: %v\n", err)
// 	}

// 	w.WriteHeader(http.StatusOK)

// 	updateActivity(ctx, req.Notif["sdrid"])
// }

// func HandlePingRequest(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()

// 	var ping PingRequest
// 	err := json.NewDecoder(r.Body).Decode(&ping)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body message request: %v\n", err)
// 	}

// 	errChan := make(chan *error, len(ping.Targets))
// 	tknChan := make(chan *string, len(ping.Targets))

// 	go getMessagingTokens(ctx, ping.Targets, tknChan, errChan)
// 	tokens := make([]string, 0)

// 	for range ping.Targets {
// 		select {
// 		case adr := <-tknChan:
// 			tokens = append(tokens, *adr)
// 		case err := <-errChan:
// 			log.Printf("error getting a target: %v\n", *err)
// 		}
// 	}

// 	var title string
// 	if ping.Notif["ln"] != "" {
// 		title = ping.Notif["sdrnm"] + " pinged you"
// 	} else {
// 		title = ping.Notif["sdrnm"] + " " + ping.Notif["sdrln"] + " pinged you"
// 	}
// 	if _, err := ms.MSGR.SendMulticast(
// 		ctx,
// 		&messaging.MulticastMessage{
// 			Tokens: tokens,
// 			Notification: &messaging.Notification{
// 				Title: title,
// 				Body:  ping.Notif["txt"]},
// 		},
// 	); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error multicasting message: %v\n", err)
// 	}

// 	w.WriteHeader(http.StatusOK)

// 	updateActivity(ctx, ping.Notif["sdrid"])
// }
