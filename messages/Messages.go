package messages

import (
	"context"
	"crypto/sha1"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
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
	Media     []byte   `json:"m"`
}

type Down4Media struct {
	Identifier string `json:"id"`
	Data       []byte `json:"d"`
}

type MessageServer struct {
	RTDB    *db.Client
	MSGBCKT *storage.BucketHandle
	MSGR    *messaging.Client
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

	ms = MessageServer{
		RTDB:    rtdb,
		MSGBCKT: msgbckt,
		MSGR:    msgr,
	}

}

func (m *MessageRequest) ToNotification(messageID, mediaID string) map[string]string {

	mp := make(map[string]string)

	mp["id"] = mediaID
	mp["sd"] = (*m).Sender
	mp["tg"] = strings.Join((*m).Targets, " ")
	mp["tn"] = (*m).Thumbnail
	mp["nm"] = (*m).Name
	mp["txt"] = (*m).Text
	mp["ts"] = strconv.FormatInt((*m).Timestamp, 10)
	mp["r"] = strings.Join((*m).Reactions, " ")
	mp["n"] = strings.Join((*m).Nodes, " ")
	mp["ch"] = strconv.FormatBool((*m).IsChat)
	mp["m"] = mediaID

	return mp
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

	ms.MSGR.SendMulticast(
		ctx,
		&messaging.MulticastMessage{
			Tokens: tokens,
			Data:   msgReq.ToNotification(messageID, mediaID),
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
			userTokenRef := ms.RTDB.NewRef("/Users/" + id_ + "/tkn/")
			var token string
			if err := userTokenRef.Get(ctx, &token); err != nil {
				ech <- &err
			} else {
				ch <- &token
			}
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
