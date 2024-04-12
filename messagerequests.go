package backend

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"encoding/json"
	"log"
	"net/http"

	rtdb "firebase.google.com/go/v4/db"
	"firebase.google.com/go/v4/messaging"
)

type MessageRequest struct {
	Msg     map[string]string `json:"msg"`
	Sender  string            `json:"s"`
	Root    string            `json:"r"`
	Header  string            `json:"h"`
	Body    string            `json:"b"`
	Push    string            `json:"p"`
	Targets []*MessageTarget  `json:"trgts"`
}

type MessageTarget struct {
	UserId    string `json:"uid"`
	DeviceId  string `json:"dev"`
	Token     string `json:"tkn"`
	ShowNotif bool   `json:"ntf"`
	DoPush    bool   `json:"psh"`
}

type PushRes struct {
	Push    string    `json:"psh"`
	Replays []*Replay `json:"rps"`
}

type Replay struct {
	UserId   string `json:"uid"`
	DeviceId string `json:"dev"`
}

func init() {
	ctx := context.Background()
	ServerInit(ctx)
}

func PushData(ctx context.Context, userId, deviceId, payload string) *Replay {
	id := ParseRoot(userId)[0]
	db := id.ServerShard().RealtimeDB
	ref := db.NewRef("roots/" + id.Unik + "/queues/" + deviceId)
	if _, err := ref.Push(ctx, payload); err != nil {
		return &Replay{UserId: userId, DeviceId: deviceId}
	}
	return nil
}

func (mr *MessageRequest) makeNotifications(rps []*Replay) []*messaging.Message {
	msgs := make([]*messaging.Message, 0, len(mr.Targets))
	for _, t := range mr.Targets {
		skip := ContainsWhere(t, rps, func(a *MessageTarget, b *Replay) bool {
			return a.UserId == b.UserId && a.DeviceId == b.DeviceId
		})

		if !skip && t.ShowNotif {
			m := &messaging.Message{
				Token: t.Token,
				Data: map[string]string{
					"b": mr.Body,
					"h": mr.Header,
					"r": mr.Root,
					"s": mr.Sender,
				},
			}
			msgs = append(msgs, m)
		}
	}
	return msgs
}

func PushNotification(ctx context.Context, token, body, header, rootId, senderId string) error {
	m := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"b": body,
			"h": header,
			"r": rootId,
			"s": senderId,
		},
	}
	_, err := Client.Messager.Send(ctx, m)
	return err
}

func prettyPrint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}

func pushRequest(ctx context.Context, targets []*MessageTarget, push string) []*Replay {
	nt := len(targets)
	rpChan, replays := make(chan *Replay, nt), make([]*Replay, 0, nt)
	handlePush_ := func(p string, mt *MessageTarget, ch chan *Replay) {
		if mt.DoPush {
			ch <- PushData(ctx, mt.UserId, mt.DeviceId, p)
		}
	}

	for i := 0; i < nt; i++ {
		go handlePush_(push, targets[i], rpChan)
	}

	for i := 0; i < nt; i++ {
		if rp := <-rpChan; rp != nil {
			replays = append(replays, rp)
		}
	}

	return replays
}

var errorMessageAlreadyExists = errors.New("message already exists")

func snipTransaction(ctx context.Context, mr *MessageRequest) (int, string, error) {
	msgId := mr.Msg["id"]
	_, rootStr, unikRoot, composedIds := ParseMessageId(msgId)
	db := composedIds[0].ServerShard().RealtimeDB
	rootRef := db.NewRef("roots/" + unikRoot)
	txRef := rootRef.Child("connection/upperSnip")
	var k int
	var upperSnip interface{}
	var newMsgId string

	snipTx := func(tn rtdb.TransactionNode) (interface{}, error) {
		if err := tn.Unmarshal(&upperSnip); err != nil {
			return nil, err
		}

		if value, ok := upperSnip.(float64); ok {
			k = int(value) + 1
		} else {
			k = 0
		}

		snipId := makeChatNumUnik(k)
		newMsgId = snipId + "-" + rootStr + "-s"
		mr.Msg["id"] = newMsgId
		txRef_ := rootRef.Child("snips/" + snipId)
		snipTx_ := func(tn rtdb.TransactionNode) (interface{}, error) {
			var m map[string]interface{}
			tn.Unmarshal(&m)
			if len(m) == 0 {
				return mr.Msg, nil
			} else {
				return nil, errorMessageAlreadyExists
			}
		}
		if err := txRef_.Transaction(ctx, snipTx_); err != nil {
			return nil, err
		}
		return k, nil
	}

	err := txRef.Transaction(ctx, snipTx)
	return k, newMsgId, err
}

func messageTransaction(ctx context.Context, mr *MessageRequest) (int, string, error) {
	msgId := mr.Msg["id"]
	_, rootStr, unikRoot, composedIds := ParseMessageId(msgId)
	db := composedIds[0].ServerShard().RealtimeDB
	rootRef := db.NewRef("roots/" + unikRoot)
	txRef := rootRef.Child("connection/upperChat")
	var k int
	var upperChat interface{}
	var newMsgIdStr string
	chatTxFunc := func(tn rtdb.TransactionNode) (interface{}, error) {
		if err := tn.Unmarshal(&upperChat); err != nil {
			return nil, err
		}

		if value, ok := upperChat.(float64); ok {
			k = int(value) + 1
		} else {
			k = 0
		}

		chatNumStr := makeChatNumUnik(k)
		newMsgIdStr = chatNumStr + "-" + rootStr + "-c"
		msg_ := CopyMap_(mr.Msg)
		msg_["id"] = newMsgIdStr
		chatTxRef_ := rootRef.Child("chats/" + chatNumStr)
		chatTxFunc_ := func(tn rtdb.TransactionNode) (interface{}, error) {
			var m map[string]interface{}
			tn.Unmarshal(&m)
			if len(m) == 0 {
				return msg_, nil
			} else {
				return nil, errorMessageAlreadyExists
			}
		}

		if err := chatTxRef_.Transaction(ctx, chatTxFunc_); err != nil {
			return nil, err
		}

		return k, nil
	}

	err := txRef.Transaction(ctx, chatTxFunc)
	return k, newMsgIdStr, err
}

var pushError error = errors.New("all push failed")

func handlePushErrors(errs []error, msg string) error {
	if Every(errs, func(err error) bool { return err != nil }) {
		return errors.New("All push failed")
		// Fatal(pushError, msg)
	} else {
		for _, err := range errs {
			NonFatal(err, "Error making a push")
		}
	}
	return nil
}

func makeChatNumUnik(chatNum int) string {
	// a 13 max length, we have max num of 9,999,999,999,999
	// which is 9 trillion+ max chats for a single root
	u := strconv.FormatUint(uint64(chatNum), 10) // base16 (hex)
	padLen := 13 - len(u)
	return strings.Repeat("0", padLen) + u
}

var chatUpdateError error = errors.New("current chat update is more recent")

func reactionTransaction(ctx context.Context, mr *MessageRequest) error {
	userPushKey := mr.Msg["id"]
	chatNum, _, unikRoot, composedIds := ParseMessageId(mr.Msg["messageId"])
	db := composedIds[0].ServerShard().RealtimeDB
	rootRef := db.NewRef("roots/" + unikRoot)

	msg := CopyMap__(mr.Msg)
	msg["reactors"] = map[string]string{mr.Msg["senderId"]: ""}
	chatRef := rootRef.Child("chats/" + chatNum + "/reactions/" + userPushKey)
	Fatal(chatRef.Set(ctx, msg), "Error setting reaction")

	genKey := MakePushKey()
	cuRef := rootRef.Child("chatUpdates/" + genKey)
	Fatal(cuRef.Set(ctx, "r "+chatNum+" "+userPushKey), "Error pushing reaction chat update")

	txFun := func(tn rtdb.TransactionNode) (interface{}, error) {
		var curChatUpdate string
		tn.Unmarshal(&curChatUpdate) // can ignore error here
		if genKey > curChatUpdate {
			return genKey, nil
		} else {
			return nil, chatUpdateError
		}
	}
	txRef := rootRef.Child("connection/chatUpdate")
	return txRef.Transaction(ctx, txFun)
}

func reactionIncrement(ctx context.Context, mr *MessageRequest) {
	reactionId := mr.Msg["reactionId"]
	reactorId := mr.Msg["senderId"]
	chatNum, _, unikRoot, composedIds := ParseMessageId(mr.Msg["messageId"])
	db := composedIds[0].ServerShard().RealtimeDB
	rootRef := db.NewRef("roots/" + unikRoot)
	txRef := rootRef.Child("connection/chatUpdate")
	chatRef := rootRef.Child("chats/" + chatNum + "/reactions/" + reactionId + "/reactors")
	pushKey := MakePushKey()
	cuRef := rootRef.Child("chatUpdates/" + pushKey)
	Fatal(chatRef.Child(reactorId).Set(ctx, ""), "Error adding reactor to chat")
	pushUpdate := "i " + reactorId + " " + chatNum + " " + reactionId
	Fatal(cuRef.Set(ctx, pushUpdate), "Error pushing increment chat udpate")
	txChatUpdate := func(tn rtdb.TransactionNode) (interface{}, error) {
		var curPush string
		tn.Unmarshal(&curPush)
		pushKey := MakePushKey()
		if pushKey > curPush {
			return pushKey, nil
		} else {
			return nil, chatUpdateError
		}
	}
	txRef.Transaction(ctx, txChatUpdate)
}

// I think we can have one function for all
func ProcessMessage(w http.ResponseWriter, r *http.Request) {
	const retry = 4
	var rtrErr error = fmt.Errorf("Exhausted %v retries\n", retry)

	ctx := context.Background()
	var mrs []*MessageRequest

	Fatal(json.NewDecoder(r.Body).Decode(&mrs), "Error decoding requests")

	var prs []*PushRes
	onDone := func(err error) {
		if len(prs) > 0 {
			b, _ := json.Marshal(prs)
			w.Write(b)
		}
		if err != nil {
			log.Fatalf("Fatal error: %v\n", err)
		}
	}

	for _, mr := range mrs {
		var (
			k       int
			msgid   string
			err     error
			replays []*Replay
			psh     string
		)

		if len(mr.Push) > 0 {
			psh = mr.Push
			replays = pushRequest(ctx, mr.Targets, mr.Push)
			// handlePushErrors(errs, "Error pushing push")
		} else if len(mr.Msg) > 0 {
			switch mr.Msg["type"] {
			case "chat":
				for i := 0; i < retry; i++ {
					log.Printf("Attempt #%v for %v\n", i, mr.Msg["id"])
					k, msgid, err = messageTransaction(ctx, mr)
					if err == errorMessageAlreadyExists {
						if i == retry-1 {
							onDone(fmt.Errorf("Chat error: %v\n", rtrErr))
						} else {
							// set it to null and try again via looping
							err = nil
							continue
						}
					} else if err != nil {
						// this is a fatal error
						onDone(fmt.Errorf("Message error: %v\n", err))
					} else {
						// success
						if k == 0 {
							psh = "m" + msgid
							replays = pushRequest(ctx, mr.Targets, psh)
							break
						}
					}
				}
				break
			case "snip":
				for i := 0; i < retry; i++ {
					log.Printf("Attempt #%v for %v\n", i, mr.Msg["id"])
					k, msgid, err = messageTransaction(ctx, mr)
					if err == errorMessageAlreadyExists {
						if i == retry-1 {
							onDone(fmt.Errorf("Snip error: %v\n", rtrErr))
						} else {
							// set it to null and try again via looping
							err = nil
							continue
						}
					} else if err != nil {
						// this is a fatal error
						onDone(fmt.Errorf("Snip error: %v\n", err))

					} else {
						// success
						if k == 0 {
							psh = "m" + msgid
							replays = pushRequest(ctx, mr.Targets, psh)
							break
						}
					}
				}
				break
			case "reaction":
				for i := 0; i < retry; i++ {
					err = reactionTransaction(ctx, mr)
					if err == chatUpdateError {
						if i == retry-1 {
							onDone(fmt.Errorf("React error: %v\n", rtrErr))
						} else {
							err = nil
							continue
						}
					} else if err != nil {
						onDone(fmt.Errorf("Reaction error: %v\n", err))
						return
					} else {
						// success
						break
					}
				}
				break
			case "increment":
				reactionIncrement(ctx, mr)
				break
			}
		}

		if len(mr.Header) > 0 {
			ntfs := mr.makeNotifications(replays)
			br, err := Client.Messager.SendEach(ctx, ntfs)
			NonFatal(err, "Error sending notifications")
			for _, x := range br.Responses {
				NonFatal(x.Error, "Error sending a notification")
			}
		}

		if len(psh) > 1 && len(replays) > 1 {
			prs = append(prs, &PushRes{Push: psh, Replays: replays})
		}
	}
	
	onDone(nil)
}

// func handlePush(ctx context.Context, doPush bool, q *MessageRequest, mt MessageTarget, ec chan *error, ac chan bool, mc chan *messaging.Message, rootID, senderID, push string) {
// 	if doPush && mt.DoPush && len(push) > 0 && len(mt.DeviceId) > 0 {
// 		uni, reg, shrd, err := utils.ParseID(mt.UserId)
// 		if err != nil {
// 			ec <- &err
// 			return
// 		}
// 		srv := server.Client.Shards[reg][shrd].RealtimeDB
// 		ref := srv.NewRef("roots/" + uni + "/queues/" + mt.DeviceId)
// 		if _, err = ref.Push(ctx, push); err != nil {
// 			ec <- &err
// 			return
// 		}
// 	}

// 	if mt.ShowNotif && len(mt.Token) > 0 {
// 		m := &messaging.Message{
// 			Token: mt.Token,
// 			Data: map[string]string{
// 				"b": q.Body,
// 				"h": q.Header,
// 				"r": rootID,
// 				"s": senderID,
// 			},
// 		}
// 		mc <- m
// 	} else {
// 		ac <- true
// 	}
// }

// type mt struct {
// 	UserId    string `json:"uid"`
// 	DeviceId  string `json:"dev"`
// 	Token     string `json:"tkn"`
// 	ShowNotif bool   `json:"ntf"`
// 	DoPush    bool   `json:"psh"`
// }

// type mq struct {
// 	Mts      []mt   `json:"m"`
// 	Push     string `json:"p"`
// 	Header   string `json:"h"`
// 	Body     string `json:"b"`
// 	SenderId string `json:"s"`
// 	RootId   string `json:"r"`
// }

// func handlePush(ctx context.Context, q *mq, mt mt, ec chan *error, ac chan bool, mc chan *messaging.Message) {
// 	if mt.DoPush && len(q.Push) > 0 {
// 		uni, reg, shrd, err := utils.ParseID(mt.UserId)
// 		if err != nil {
// 			ec <- &err
// 			return
// 		}
// 		srv := server.Client.Shards[reg][shrd].RealtimeDB
// 		ref := srv.NewRef("nodes/" + uni + "/queues/" + mt.DeviceId)
// 		if _, err = ref.Push(ctx, q.Push); err != nil {
// 			ec <- &err
// 			return
// 		}
// 	}

// 	if mt.ShowNotif {
// 		m := &messaging.Message{
// 			Token: mt.Token,
// 			Data: map[string]string{
// 				"b": q.Body,
// 				"h": q.Header,
// 				"r": q.RootId,
// 				"s": q.SenderId,
// 			},
// 		}
// 		mc <- m
// 	} else {
// 		ac <- true
// 	}
// }

// func HandleMessageRequest(w http.ResponseWriter, r *http.Request) {
// 	ctx := context.Background()

// 	onFatalErrf := func(errMsg string, a ...any) {
// 		w.WriteHeader(500)
// 		log.Fatalf(errMsg, a)
// 	}

// 	var req mq
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		onFatalErrf("error decoding request json: %v", err)
// 	}

// 	n := len(req.Mts)
// 	ec := make(chan *error, n)
// 	ac := make(chan bool, n)
// 	mc := make(chan *messaging.Message, n)

// 	for _, v := range req.Mts {
// 		go handlePush(ctx, &req, v, ec, ac, mc)
// 	}

// 	msgs := make([]*messaging.Message, 0)
// 	for i := 0; i < n; i++ {
// 		select {
// 		case m := <-mc:
// 			msgs = append(msgs, m)
// 			break
// 		case <-ac:
// 			break
// 		case e := <-ec:
// 			log.Printf("error handleing push: %v\n", *e)
// 			break
// 		}
// 	}

// 	if _, err := server.Client.Messager.SendEach(ctx, msgs); err != nil {
// 		log.Printf("error sending notifs: %v\n", err)
// 	}
// }
