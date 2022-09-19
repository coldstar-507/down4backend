package transaction

import (
	"context"
	"log"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
)

type TransactionServer struct {
	RTDB *db.Client
	MSGR *messaging.Client
}

var ts TransactionServer

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

	ts = TransactionServer{
		RTDB: rtdb,
		MSGR: msgr,
	}

}

// func GetPayTransactionAdresses(w http.ResponseWriter, r *http.Request) {

// 	ctx := context.Background()

// 	var transaction Transaction
// 	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error decoding body transaction: %v\n", err)
// 	}
// 	targetAdresses := make([]string, len(transaction.Targets))
// 	ach := make(chan *string, len(transaction.Targets))
// 	ech := make(chan *error, len(transaction.Targets))

// 	go func() {
// 		for _, v := range transaction.Targets {
// 			uid := v
// 			go getNewAdress(ctx, uid, ach, ech)
// 		}
// 	}()

// 	for i := range targetAdresses {
// 		select {
// 		case adr := <-ach:
// 			targetAdresses[i] = *adr
// 		case err := <-ech:
// 			w.WriteHeader(http.StatusInternalServerError)
// 			log.Fatalf("error getting a target in ProcessPayTransaction: %v\n", *err)
// 		}
// 	}

// 	var changeAdress string
// 	go getNewChangeAdress(ctx, transaction.From, ach, ech)
// 	select {
// 	case adr := <-ach:
// 		changeAdress = *adr
// 	case err := <-ech:
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error generating change adresse in ProcessPayTransaction: %v\n", *err)
// 	}

// 	neededAdresses := map[string]interface{}{
// 		"targets": targetAdresses,
// 		"change":  changeAdress,
// 	}

// 	data, err := json.Marshal(neededAdresses)
// 	if err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error marshalling pay transaction data in ProcessPayTransaction: %v\n", err)
// 	}

// 	if _, err = w.Write(data); err != nil {
// 		w.WriteHeader(http.StatusInternalServerError)
// 		log.Fatalf("error writing pay transaction data to response in ProcessPayTransaction: %v\n", err)
// 	}

// 	w.WriteHeader(http.StatusOK)
// }

// func getNewAdress(ctx context.Context, username string, ach chan *string, ech chan *error) {

// 	getNewAdresse_ := func(tn db.TransactionNode) (interface{}, error) {
// 		var pmi PublicMoneyInfo
// 		if err := tn.Unmarshal(&pmi); err != nil {
// 			log.Printf("error unmarshalling public money info in GetNewAdresse: %v\n", err)
// 			return nil, err
// 		}
// 		neuter, err := bip32.NewKeyFromString(pmi.Neuter)
// 		if err != nil {
// 			log.Printf("error making key from string in GetNewAdresse: %v\n", err)
// 			return nil, err
// 		}
// 		nextNeuter, err := neuter.Child(pmi.Index)
// 		if err != nil {
// 			log.Printf("error deriving child key in GetNewAdresse: %v\n", err)
// 			return nil, err
// 		}
// 		adress := nextNeuter.Address(&chaincfg.MainNet)
// 		pmi.Index = pmi.Index + 2

// 		ach <- &adress
// 		return pmi, nil
// 	}

// 	if err := ts.RTDB.NewRef(username+"/mny").Transaction(ctx, getNewAdresse_); err != nil {
// 		log.Printf("error getting new adress for user %s: %v\n", username, err)
// 		ech <- &err
// 	}

// }

// func getNewChangeAdress(ctx context.Context, username string, ach chan *string, ech chan *error) {

// 	getNewChangeAdresse_ := func(tn db.TransactionNode) (interface{}, error) {
// 		var pmi PublicMoneyInfo
// 		if err := tn.Unmarshal(&pmi); err != nil {
// 			log.Printf("error unmarshalling public money info in GetChangeAdress: %v\n", err)
// 			return nil, err
// 		}
// 		neuter, err := bip32.NewKeyFromString(pmi.Neuter)
// 		if err != nil {
// 			log.Printf("error loading neuter from string in GetChangeAdress: %v\n", err)
// 			return nil, err
// 		}
// 		nextNeuter, err := neuter.Child(pmi.Change)
// 		if err != nil {
// 			log.Printf("error deriving child key in GetChangeAdress: %v\n", err)
// 			return nil, err
// 		}
// 		adress := nextNeuter.Address(&chaincfg.MainNet)
// 		pmi.Index = pmi.Change + 2

// 		ach <- &adress
// 		return pmi, nil
// 	}

// 	if err := ts.RTDB.NewRef(username+"/mny").Transaction(ctx, getNewChangeAdresse_); err != nil {
// 		ech <- &err
// 		log.Printf("error getting change adress for user %s: %v\n", username, err)
// 	}
// }
