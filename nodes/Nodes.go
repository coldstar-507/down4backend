package nodes

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
)

type Down4Media struct {
	Identifier string `json:"id"`
	Data       []byte `json:"d"`
}

type RealTimeNode struct {
	Identifier string            `json:"id"`
	Type       string            `json:"t"`
	Name       string            `json:"nm"`
	Lastname   string            `json:"ln"`
	ImageID    string            `json:"im"`
	Token      string            `json:"tkn"`
	Friends    map[string]string `json:"frd"`
	Messages   map[string]string `json:"msg"`
	Admins     map[string]string `json:"adm"`
	Childs     map[string]string `json:"chl"`
	Parents    map[string]string `json:"prt"`
}

type FullNode struct {
	Identifier string     `json:"id"`
	Type       string     `json:"t"`
	Name       string     `json:"nm"`
	Lastname   string     `json:"ln"`
	Image      Down4Media `json:"im"`
	Messages   []string   `json:"msg"`
	Admins     []string   `json:"adm"`
	Childs     []string   `json:"chl"`
	Parents    []string   `json:"prt"`
}

type NodeServer struct {
	RTDB   *db.Client
	NDBCKT *storage.BucketHandle
}

var ns NodeServer

func init() {

	config := &firebase.Config{
		DatabaseURL: "https://down4-26ee1-default-rtdb.firebaseio.com/",
	}

	ctx := context.Background()

	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	rtdb, err := app.Database(ctx)
	if err != nil {
		log.Fatalf("error initializing db: %v\n", err)
	}

	stor, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("error initializing storage: %v\n", err)
	}

	ndbcket := stor.Bucket("down4-26ee1-nodes")

	ns = NodeServer{
		RTDB:   rtdb,
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
	nodeChan := make(chan *FullNode, len(ids))
	go func() {
		for _, id := range ids {
			id_ := id
			go getNode(ctx, id_, nodeChan, errChan)
		}
	}()

	nodes := make([]FullNode, 0)
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

func getNode(ctx context.Context, id string, nodeChan chan *FullNode, errChan chan *error) {

	var rtn RealTimeNode
	if err := ns.RTDB.NewRef("/Nodes/"+id).Get(ctx, &rtn); err != nil {
		errChan <- &err
		log.Printf("error getting node id: %s from collection 'Nodes': %v\n", id, err)
		return
	}

	imData, err := getNodeMedia(ctx, rtn.ImageID)
	if err != nil {
		errChan <- &err
		log.Printf("error reading image data from bucket reader: %v\n", err)
		return
	}

	node := FullNode{
		Identifier: rtn.Identifier,
		Type:       rtn.Type,
		Name:       rtn.Name,
		Lastname:   rtn.Lastname,
		Image:      Down4Media{Identifier: rtn.ImageID, Data: imData},
		Messages:   mapToSliceString(rtn.Messages),
		Admins:     mapToSliceString(rtn.Admins),
		Childs:     mapToSliceString(rtn.Childs),
		Parents:    mapToSliceString(rtn.Parents),
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

func mapToSliceString(m map[string]string) []string {
	s := make([]string, 0)
	for _, v := range m {
		s = append(s, v)
	}
	return s
}
