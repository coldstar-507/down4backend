package nodesFS

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
)

type NodeServer struct {
	FS     *firestore.Client
	NDBCKT *storage.BucketHandle
}

var ns NodeServer

func init() {

	ctx := context.Background()

	fs, err := firestore.NewClient(ctx, "down4-26ee1")
	if err != nil {
		log.Fatalf("error initializing db: %v\n", err)
	}

	stor, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("error initializing storage: %v\n", err)
	}

	ndbcket := stor.Bucket("down4-26ee1-nodes")

	ns = NodeServer{
		FS:     fs,
		NDBCKT: ndbcket,
	}

}

func GetNodes(w http.ResponseWriter, r *http.Request) {

	ctx := context.Background()

	bytesIDs, err := io.ReadAll(r.Body) // expect a string of IDs linked by a space " "
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error reading data from request in GetNodes: %v\n", err)
	}

	ids := strings.Split(string(bytesIDs), " ")

	// buffered is faster and espicially more consistent
	errChan := make(chan *error, len(ids))
	nodeChan := make(chan *OutputNode, len(ids))
	go func() {
		for _, id := range ids {
			id_ := id
			go getNode(ctx, id_, nodeChan, errChan)
		}
	}()

	nodes := make([]OutputNode, 0)
	for range ids {
		select {
		case n := <-nodeChan:
			nodes = append(nodes, *n)
		case <-errChan:
			continue
		}
	}

	encodedData, err := json.Marshal(nodes)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error marshalling nodes %v\n", err)
	}

	if _, err = w.Write(encodedData); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("error writing mashaled nodes in response: %v\n", err)
	}

	w.WriteHeader(http.StatusOK)
}

func getNode(ctx context.Context, id string, nodeChan chan *OutputNode, errChan chan *error) {

	var fsn FireStoreNode
	snap, err := ns.FS.Collection("Nodes").Doc(id).Get(ctx)
	if err != nil {
		errChan <- &err
		log.Printf("error getting node id: %s from collection 'Nodes': %v\n", id, err)
		return
	}

	if err := snap.DataTo(&fsn); err != nil {
		errChan <- &err
		log.Printf("error reading snapshot data into FireStoreNode for user: %s: %v\n", id, err)
		return
	}

	imData, err := getNodeMedia(ctx, fsn.ImageID)
	if err != nil {
		errChan <- &err
		log.Printf("error reading image data from bucket reader: %v\n", err)
		return
	}

	node := OutputNode{
		Identifier: fsn.Identifier,
		Type:       fsn.Type,
		Name:       fsn.Name,
		Lastname:   fsn.Lastname,
		Image:      Down4Media{Identifier: fsn.ImageID, Data: imData},
		Messages:   fsn.Messages,
		Admins:     fsn.Admins,
		Childs:     fsn.Childs,
		Parents:    fsn.Parents,
	}

	nodeChan <- &node

}

func getNodeMedia(ctx context.Context, id string) ([]byte, error) {

	rc, err := ns.NDBCKT.Object(id).NewReader(ctx)
	if err != nil {
		return nil, err
	}

	mediaData, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	return mediaData, nil
}
