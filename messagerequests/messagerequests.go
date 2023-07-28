package messagerequests

import (
	"context"
	"fmt"

	"encoding/json"
	"log"
	"net/http"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
)

type messageRequest struct {
	Tokens         []string          `json:"t"`
	SenderID       string            `json:"s"`
	SenderName     string            `json:"sn"`
	GroupName      string            `json:"gn"`
	Header         string            `json:"h"`
	Data           map[string]string `json:"d"`
	Body           string            `json:"b"`
	GroupMediaURL  string            `json:"gtn"`
	SenderMediaURL string            `json:"stn"`
}

var msgr *messaging.Client

func init() {
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, &firebase.Config{})
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	msgr, err = app.Messaging(ctx)
	if err != nil {
		log.Fatalf("error initializing messager: %v\n", err)
	}

}

func HandleMessageRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	onFatalErrf := func(errMsg string, a ...any) {
		w.WriteHeader(500)
		log.Fatalf(errMsg, a)
	}

	var req messageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		onFatalErrf("error decoding request json: %v", err)
	}

	fmt.Printf("msgReq: %v\n", req)

	br, err := msgr.SendMulticast(ctx, &messaging.MulticastMessage{
		Tokens: req.Tokens,
		Data:   req.Data,
		Notification: &messaging.Notification{
			Title: req.Header,
			Body:  req.Body,
		},
	})

	fmt.Printf("success count: %v\n", br.SuccessCount)

	if err != nil {
		onFatalErrf("error multicasting message: %v", err)
	}

	jsn, err := json.Marshal(br)
	if err != nil {
		onFatalErrf("error marshalling batch response: %v", err)
	}

	if _, err = w.Write(jsn); err != nil {
		onFatalErrf("error writing response: %v", err)
	}

}
