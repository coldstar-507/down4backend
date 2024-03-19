package backend

import (
	"context"
	"os"
	"time"

	"log"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type ServerShard struct {
	RealtimeDB   *db.Client
	TempBucket   *storage.BucketHandle
	StaticBucket *storage.BucketHandle
}

const (
	N_SHARD  = 2
	N_REGION = 3
)

type server struct {
	Shards     map[string]([N_SHARD]ServerShard)
	Messager   *messaging.Client
	Firestore  *firestore.Client
	SignedOpts *storage.SignedURLOptions
}

var Client *server


func ServerInit(ctx context.Context) {
	servAcc := os.Getenv("FIREBASE_CONFIG")
	opt := option.WithCredentialsFile(servAcc)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	msgr, err := app.Messaging(ctx)
	if err != nil {
		log.Fatalf("error initializing messager: %v\n", err)
	}

	fs, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("error initializing db: %v\n", err)
	}

	stor, err := app.Storage(ctx)
	if err != nil {
		log.Fatalf("error initializing storage: %v\n", err)
	}

	sUrls := &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(time.Hour * 24 * 4),
	}

	db_am_1, _ := app.DatabaseWithURL(ctx, "https://down4-26ee1-fd90e-us1.firebaseio.com/")
	db_am_2, _ := app.DatabaseWithURL(ctx, "https://down4-26ee1-c65d2-us2.firebaseio.com/")
	db_eu_1, _ := app.DatabaseWithURL(ctx, "https://down4-26ee1-30b1c-eu1.europe-west1.firebasedatabase.app/")
	db_eu_2, _ := app.DatabaseWithURL(ctx, "https://down4-26ee1-e487b-eu2.europe-west1.firebasedatabase.app/")
	db_as_1, _ := app.DatabaseWithURL(ctx, "https://down4-26ee1-8511f-sea1.asia-southeast1.firebasedatabase.app/")
	db_as_2, _ := app.DatabaseWithURL(ctx, "https://down4-26ee1-d98a8-sea2.asia-southeast1.firebasedatabase.app/")

	tmp_am_1, _ := stor.Bucket("down4-26ee1-us1-tmp")
	tmp_am_2, _ := stor.Bucket("down4-26ee1-us2-tmp")
	tmp_eu_1, _ := stor.Bucket("down4-26ee1-eu1-tmp")
	tmp_eu_2, _ := stor.Bucket("down4-26ee1-eu2-tmp")
	tmp_as_1, _ := stor.Bucket("down4-26ee1-sea1-tmp")
	tmp_as_2, _ := stor.Bucket("down4-26ee1-sea2-tmp")

	st_am_1, _ := stor.Bucket("down4-26ee1-us1")
	st_am_2, _ := stor.Bucket("down4-26ee1-us2")
	st_eu_1, _ := stor.Bucket("down4-26ee1-eu1")
	st_eu_2, _ := stor.Bucket("down4-26ee1-eu2")
	st_as_1, _ := stor.Bucket("down4-26ee1-sea1")
	st_as_2, _ := stor.Bucket("down4-26ee1-sea2")

	Client = &server{
		SignedOpts: sUrls,
		Firestore:  fs,
		Messager:   msgr,
		Shards: map[string][N_SHARD]ServerShard{
			"america": {
				ServerShard{
					RealtimeDB:   db_am_1,
					TempBucket:   tmp_am_1,
					StaticBucket: st_am_1,
				},
				ServerShard{
					RealtimeDB:   db_am_2,
					TempBucket:   tmp_am_2,
					StaticBucket: st_am_2,
				},
			},
			"asia": {
				ServerShard{
					RealtimeDB:   db_as_1,
					TempBucket:   tmp_as_1,
					StaticBucket: st_as_1,
				},
				ServerShard{
					RealtimeDB:   db_as_2,
					TempBucket:   tmp_as_2,
					StaticBucket: st_as_2,
				},
			},
			"europe": {
				ServerShard{
					RealtimeDB:   db_eu_1,
					TempBucket:   tmp_eu_1,
					StaticBucket: st_eu_1,
				},
				ServerShard{
					RealtimeDB:   db_eu_2,
					TempBucket:   tmp_eu_2,
					StaticBucket: st_eu_2,
				},
			},
		},
	}
}
