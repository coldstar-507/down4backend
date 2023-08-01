package getnodes

import (
	"context"
	"fmt"
	"io"
	"strings"

	"encoding/json"
	"log"
	"net/http"

	"firebase.google.com/go/v4/db"
	"github.com/coldstar-507/down4backend/server"
	"github.com/coldstar-507/down4backend/utils"
)

func init() {
	ctx := context.Background()
	server.ServerInit(ctx)
}

func getFullID(ctx context.Context, unique string, ec chan *error, sc chan *string) {
	ref, err := server.Client.Firestore.Collection("users").Doc(unique).Get(ctx)
	if err != nil {
		ec <- &err
		return
	}

	fullID, err := ref.DataAt("id")
	if err != nil {
		ec <- &err
		return
	}

	if s, ok := fullID.(string); ok {
		sc <- &s
	} else {
		err = fmt.Errorf("error: users/%v/id isn't a string, it's a %T\n", unique, fullID)
		ec <- &err
	}
}

func GetNodes(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatalf("error reading request body: %v\n", err)
	}

	usernames := strings.Split(string(raw), " ")
	sc := make(chan *string, len(usernames))
	ec := make(chan *error, len(usernames))
	fullIDs := make([]*string, 0, len(usernames))

	for _, v := range usernames {
		go getFullID(ctx, v, ec, sc)
	}

	for range usernames {
		select {
		case id := <-sc:
			fullIDs = append(fullIDs, id)
		case e := <-ec:
			log.Printf("error getting a fullID: %v\n", *e)
		}
	}

	nc := make(chan *map[string]interface{}, len(fullIDs))
	nodes := make([]*map[string]interface{}, 0, len(fullIDs))
	ec2 := make(chan *error, len(fullIDs))

	for _, id := range fullIDs {
		go getNode(ctx, *id, ec2, nc)
	}

	for range fullIDs {
		select {
		case n := <-nc:
			nodes = append(nodes, n)
		case e := <-ec2:
			log.Printf("error getting a full node: %v\n", *e)
		}
	}

	marsh, err := json.Marshal(nodes)
	if err != nil {
		log.Fatalf("error marshaling nodes: %v\n", err)
	}

	if _, err := w.Write(marsh); err != nil {
		log.Printf("error writing data to w, err: %v\n", err)
	}
}

func getNodeMedia(ctx context.Context, id string) (string, map[string]string, error) {
	_, reg, shrd, err := utils.ParseID(id)
	if err != nil {
		return "", nil, err
	}

	bckt := server.Client.Shards[reg][shrd].StaticBucket
	sUrl, err := bckt.SignedURL(id, server.Client.SignedOpts)

	obj := bckt.Object(id)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return "", nil, err
	}

	return sUrl, attrs.Metadata, nil
}

func getNode(ctx context.Context, id string, ec chan *error, nc chan *map[string]interface{}) {
	var (
		err      error
		db       *db.Client
		full     map[string]interface{} = make(map[string]interface{}, 3)
		node     map[string]interface{}
		link     string
		metadata map[string]string
	)
	_, reg, shrd, err := utils.ParseID(id)
	if err != nil {
		ec <- &err
		return
	}

	db = server.Client.Shards[reg][shrd].RealtimeDB
	if err = db.NewRef("users/"+id).Get(ctx, &node); err != nil {
		ec <- &err
		return
	}

	full["node"] = &node

	mediaID, ok := node["mediaID"].(string)
	if !ok {
		log.Printf("could not get node media and link :: mediaID isn't a string")
		nc <- &full
		return
	}

	link, metadata, err = getNodeMedia(ctx, mediaID)
	if err != nil {
		log.Printf("could not get node media and link :: %v\n", err)
		nc <- &full
		return
	}
	full["metadata"] = &metadata
	full["link"] = &link

	nc <- &full
}
