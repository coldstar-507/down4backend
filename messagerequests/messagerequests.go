package messagerequests

import (
	"context"

	"encoding/json"
	"log"
	"net/http"

	"firebase.google.com/go/v4/messaging"
	"github.com/coldstar-507/down4backend/server"
	"github.com/coldstar-507/down4backend/utils"
)

type mt struct {
	UserID    string `json:"uid"`
	DeviceID  string `json:"dev"`
	Token     string `json:"tkn"`
	ShowNotif bool   `json:"ntf"`
	DoPush    bool   `json:"psh"`
}

type mq struct {
	Mts      []mt   `json:"m"`
	Push     string `json:"p"`
	Header   string `json:"h"`
	Body     string `json:"b"`
	SenderID string `json:"s"`
	RootID   string `json:"r"`
}

func init() {
	ctx := context.Background()
	server.ServerInit(ctx)
}

func PushData(ctx context.Context, userID, deviceID, payload string) error {
	uni, reg, shrd, err := utils.ParseID(userID)
	if err != nil {
		return err
	}
	srv := server.Client.Shards[reg][shrd].RealtimeDB
	ref := srv.NewRef("nodes/" + uni + "/queues/" + deviceID)
	if _, err = ref.Push(ctx, payload); err != nil {
		return err
	}
	return nil
}

func PushNotification(ctx context.Context, token, body, header, rootID, senderID string) error {
	m := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"b": body,
			"h": header,
			"r": rootID,
			"s": senderID,
		},
	}
	_, err := server.Client.Messager.Send(ctx, m)
	return err
}

func handlePush(ctx context.Context, q *mq, mt mt, ec chan *error, ac chan bool, mc chan *messaging.Message) {
	if mt.DoPush && len(q.Push) > 0 {
		uni, reg, shrd, err := utils.ParseID(mt.UserID)
		if err != nil {
			ec <- &err
			return
		}
		srv := server.Client.Shards[reg][shrd].RealtimeDB
		ref := srv.NewRef("nodes/" + uni + "/queues/" + mt.DeviceID)
		if _, err = ref.Push(ctx, q.Push); err != nil {
			ec <- &err
			return
		}
	}

	if mt.ShowNotif {
		m := &messaging.Message{
			Token: mt.Token,
			Data: map[string]string{
				"b": q.Body,
				"h": q.Header,
				"r": q.RootID,
				"s": q.SenderID,
			},
		}
		mc <- m
	} else {
		ac <- true
	}
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func HandleMessageRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	onFatalErrf := func(errMsg string, a ...any) {
		w.WriteHeader(500)
		log.Fatalf(errMsg, a)
	}

	var req mq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		onFatalErrf("error decoding request json: %v", err)
	}

	n := len(req.Mts)
	ec := make(chan *error, n)
	ac := make(chan bool, n)
	mc := make(chan *messaging.Message, n)

	for _, v := range req.Mts {
		go handlePush(ctx, &req, v, ec, ac, mc)
	}

	msgs := make([]*messaging.Message, 0)
	for i := 0; i < n; i++ {
		select {
		case m := <-mc:
			msgs = append(msgs, m)
			break
		case <-ac:
			break
		case e := <-ec:
			log.Printf("error handleing push: %v\n", *e)
			break
		}
	}

	if _, err := server.Client.Messager.SendAll(ctx, msgs); err != nil {
		log.Printf("error sending notifs: %v\n", err)
	}
}
