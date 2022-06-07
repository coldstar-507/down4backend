package messages

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"github.com/libsv/go-bk/base58"
)

type MessageRequest struct {
	Sender    string   `json:"sd"`
	Targets   []string `json:"tg"`
	Thumbnail string   `json:"tn"`
	Name      string   `json:"nm"`
	Text      string   `json:"txt"`
	Timestamp int64    `json:"ts"`
	Reactions []string `json:"r"`
	Nodes     []string `json:"n"`
	IsChat    bool     `json:"ch"`
	IsVideo   bool     `json:"vid"`
	Media     []byte   `json:"m"`
}

type Down4Media struct {
	Identifier string `json:"id"`
	Data       []byte `json:"d"`
}

type MessageServer struct {
	FS      *firestore.Client
	MSGBCKT *storage.BucketHandle
	MSGR    *messaging.Client
}

type UserInfo struct {
	Secret   string `json:"secret"`
	Activity int64  `json:"activity"`
	Token    string `json:"token"`
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

	fs, err := firestore.NewClient(ctx, "down4-26ee1")
	if err != nil {
		log.Fatalf("error initializing firestore: %v\n", err)
	}

	stor, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("error initializing storage: %v\n", err)
	}

	msgbckt := stor.Bucket("down4-26ee1-messages")

	ms = MessageServer{
		FS:      fs,
		MSGBCKT: msgbckt,
		MSGR:    msgr,
	}

}

func (m *MessageRequest) ToNotification(messageID, mediaID string) (map[string]string, error) {

	var url string
	var err error
	if (*m).IsVideo {
		if url, err = ms.MSGBCKT.SignedURL(mediaID, nil); err != nil {
			return nil, err
		}
	}

	mp := make(map[string]string)

	mp["id"] = mediaID
	mp["sd"] = (*m).Sender
	mp["tn"] = (*m).Thumbnail
	mp["nm"] = (*m).Name
	mp["txt"] = (*m).Text
	mp["ts"] = strconv.FormatInt((*m).Timestamp, 10)
	mp["r"] = strings.Join((*m).Reactions, " ")
	mp["n"] = strings.Join((*m).Nodes, " ")
	mp["ch"] = strconv.FormatBool((*m).IsChat)
	mp["m"] = mediaID
	mp["url"] = url

	return mp, nil
}

func HandleMessageRequest(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	var msgReq MessageRequest
	err := json.NewDecoder(r.Body).Decode(&msgReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error decoding body message request: %v\n", err)
	}

	h, bSender, bText := sha1.New(), []byte(msgReq.Sender), []byte(msgReq.Text)
	hashingData := append(bSender, append(bText, msgReq.Media...)...)
	if _, err = h.Write(hashingData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error hashing message data to create message ID: %v\n", err)
	}
	messageID := base58.Encode(h.Sum(nil))

	errChan := make(chan *error, len(msgReq.Targets))
	tknChan := make(chan *string, len(msgReq.Targets))

	go getMessagingTokens(ctx, msgReq.Targets, tknChan, errChan)
	tokens := make([]string, 0)

	for range msgReq.Targets {
		select {
		case adr := <-tknChan:
			tokens = append(tokens, *adr)
		case err := <-errChan:
			log.Printf("error getting a target: %v\n", *err)
		}
	}

	var mediaID string
	if len(msgReq.Media) != 0 {
		h2 := sha1.New()
		if _, err = h2.Write(append([]byte(msgReq.Sender), msgReq.Media...)); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error hashing and ID for message media: %v\n", err)
		}
		mediaID = string(h2.Sum(nil))
		err = uploadMessageMedia(ctx, Down4Media{Identifier: mediaID, Data: msgReq.Media})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Fatalf("error uploading media in message request: %v\n", err)
		}
	}

	notifData, err := msgReq.ToNotification(messageID, mediaID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error transforming message request to notification in message request: %v\n", err)
	}
	ms.MSGR.SendMulticast(
		ctx,
		&messaging.MulticastMessage{
			Tokens: tokens,
			Data:   notifData,
			Notification: &messaging.Notification{
				Title: msgReq.Name,
				Body:  msgReq.Text},
		},
	)

	w.WriteHeader(http.StatusOK)
}

func getMessagingTokens(ctx context.Context, ids []string, ch chan *string, ech chan *error) {
	for _, id := range ids {
		id_ := id
		go func() {
			var userInfo UserInfo
			userRef := ms.FS.Collection("Users").Doc(id_)
			snap, err := userRef.Get(ctx)
			if err != nil {
				ech <- &err
			}
			if err = snap.DataTo(&userInfo); err != nil {
				ech <- &err
			}
			token := userInfo.Token
			ch <- &token
		}()
	}
}

func uploadMessageMedia(ctx context.Context, media Down4Media) error {

	w := ms.MSGBCKT.Object(media.Identifier).NewWriter(ctx)
	if _, err := w.Write(media.Data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}
