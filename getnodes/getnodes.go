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
	}

	fullID, err := ref.DataAt("id")
	if err != nil {
		ec <- &err
	}

	if s, ok := fullID.(string); ok {
		sc <- &s
	} else {
		err = fmt.Errorf("error: users/%v/id isn't a string", unique)
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
			log.Printf("error getting fullID: %v\n", *e)
		}
	}

	nc := make(chan *map[string]interface{}, len(fullIDs))
	nodes := make([]*map[string]interface{}, 0, len(fullIDs))

	for _, id := range fullIDs {
		go getNode(ctx, *id, ec, nc)
	}

	for range fullIDs {
		select {
		case n := <-nc:
			nodes = append(nodes, n)
		case e := <-ec:
			log.Print(*e)
		}
	}

	fmt.Printf("marshalling the nodes: %v\n", nodes)
	marsh, err := json.Marshal(nodes)
	if err != nil {
		log.Fatalf("error marshaling nodes: %v\n", err)
	}

	log.Printf("writing %v bytes of data in response\n", len(marsh))

	var jeff []interface{}
	json.Unmarshal(marsh, &jeff)
	fmt.Printf("unmarshalled: %v\n", jeff)

	if n, err := w.Write(marsh); err != nil {
		log.Printf("error writing data to w, err: %v\n", err)
	} else {
		log.Printf("wrote %v bytes to w\n", n)
	}
}

func getNodeMedia(ctx context.Context, id string) (string, map[string]string, error) {
	_, reg, shrd, err := utils.ParseID(id)
	if err != nil {
		return "", nil, err
	}

	obj := server.Client.Shards[reg][shrd].StaticBucket.Object(id)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return "", nil, err
	}

	return attrs.MediaLink, attrs.Metadata, nil
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
	}

	full["node"] = &node

	mediaID, ok := node["mediaID"].(string)
	if !ok {
		nc <- &full
		return
	}

	link, metadata, err = getNodeMedia(ctx, mediaID)
	if err != nil {
		nc <- &full
		return
	}
	full["metadata"] = &metadata
	full["link"] = &link

	nc <- &full
}
