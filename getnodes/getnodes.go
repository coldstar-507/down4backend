package getnodes

import (
	"context"
	"fmt"
	"io"
	"strings"

	"encoding/json"
	"log"
	"net/http"

	"github.com/coldstar-507/down4backend/server"
	"github.com/coldstar-507/down4backend/utils"
)

func init() {
	ctx := context.Background()
	server.ServerInit(ctx)
}

func getFullId(ctx context.Context, unique string, sc chan *string) {
	ref, err := server.Client.Firestore.Collection("users").Doc(unique).Get(ctx)
	if err != nil {
		msg := fmt.Sprintf("error getting doc at user/%s", unique)
		utils.NonFatal(err, msg)
		sc <- nil
		return
	}

	fullId, err := ref.DataAt("id")
	if err != nil {
		msg := fmt.Sprintf("error decoding userData a user/%s", unique)
		utils.NonFatal(err, msg)
		sc <- nil
		return
	}

	if s, ok := fullId.(string); ok {
		sc <- &s
	} else {
		err := fmt.Errorf("id at user/%s isn't a string", unique)
		utils.NonFatal(err, "error getting string id")
		sc <- nil
	}
}

func GetNodes(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	raw, err := io.ReadAll(r.Body)
	utils.Fatal(err, "error reading request body")

	usernames := strings.Split(string(raw), " ")
	sc := make(chan *string, len(usernames))
	fullIds := make([]*string, 0, len(usernames))

	for _, v := range usernames {
		go getFullId(ctx, v, sc)
	}

	for range usernames {
		select {
		case id := <-sc:
			if id != nil {
				fullIds = append(fullIds, id)
			}
		}
	}

	nc := make(chan *map[string]interface{}, len(fullIds))
	nodes := make([]*map[string]interface{}, 0, len(fullIds))
	for _, id := range fullIds {
		fmt.Printf("getting node for id=%s\n", *id)
		go getNode(ctx, *id, nc)
	}

	for range fullIds {
		select {
		case n := <-nc:
			if n != nil {
				nodes = append(nodes, n)
			}
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

func getNodeMedia(ctx context.Context, id *utils.ComposedId) (string, map[string]string, error) {
	bckt := id.ServerShard().StaticBucket
	sUrl, err := bckt.SignedURL(id.Unik, server.Client.SignedOpts)
	utils.NonFatal(err, fmt.Sprintf("could not get signed url for mediaId=%s", id.ToString()))

	obj := bckt.Object(id.Unik)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return "", nil, err
	}

	return sUrl, attrs.Metadata, nil
}

func getNode(ctx context.Context, idStr string, nc chan *map[string]interface{}) {
	var full map[string]interface{} = make(map[string]interface{}, 3)

	cps, err := utils.ParseRoot(idStr)
	if err != nil {
		utils.NonFatal(err, "could not parse node idStr")
		nc <- nil
		return
	}
	if len(cps) != 1 {
		log.Println("dual root is invalid for getNode")
		nc <- nil
		return
	}

	id := cps[0]
	db := id.ServerShard().RealtimeDB
	var node map[string]interface{}
	if err := db.NewRef("roots/"+id.Unik+"/node").Get(ctx, &node); err != nil {
		nc <- nil
		return
	}
	full["node"] = &node

	mediaIdStr, ok := node["mediaId"].(string)
	if !ok {
		log.Printf("could not get node media and link: mediaId isn't a string")
		nc <- &full
		return
	}

	mediaId, err := utils.ParseMediaId(mediaIdStr)
	if err != nil {
		utils.NonFatal(err, "could not parse mediaId in getNode")
		nc <- &full
	}

	link, metadata, err := getNodeMedia(ctx, mediaId)
	if err != nil {
		log.Printf("could not get node media and link: %v\n", err)
		nc <- &full
		return
	}

	full["metadata"] = &metadata
	full["link"] = &link
	nc <- &full
}
