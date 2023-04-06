package backend

type MessageRequest struct {
	Sender    string   `json:"s"`
	Targets   []string `json:"t"`
	Header    string   `json:"h"`
	Body      string   `json:"b"`
	Data      string   `json:"d"`
	Thumbnail string   `json:"n"`
}

// type Down4Media struct {
// 	Identifier string `json:"id"`
// 	Data       string `json:"data,omitempty"`
// 	Thumbnail  string `json:"thumbnail,omitempty"`
// 	// NetworkURL string            `json:"nw,omitempty"`
// 	Metadata map[string]string `json:"metadata"`
// }

// type Down4Message struct {
// 	Root        string `json:"rt,omitempty"`
// 	MessageID   string `json:"id"`
// 	SenderID    string `json:"s"`
// 	ForwarderID string `json:"f,omitempty"`
// 	Text        string `json:"txt,omitempty"`
// 	Timestamp   int64  `json:"ts"`
// 	Replies     string `json:"r,omitempty"`
// 	Nodes       string `json:"n,omitempty"`
// 	MediaID     string `json:"m,omitempty"`
// }

// type PaymentRequest struct {
// 	Sender     string   `json:"s"`
// 	Targets    []string `json:"tr"`
// 	Payment    []byte   `json:"pay"`
// 	Identifier string   `json:"id"`
// 	TextNote   string   `json:"txt"`
// }

// type PingRequest struct {
// 	Targets  []string `json:"tr"`
// 	Text     string   `json:"txt"`
// 	SenderID string   `json:"s"`
// }

// type SnipRequest struct {
// 	Sender    string   `json:"s"`
// 	MediaID   string   `json:"m"`
// 	Targets   []string `json:"tr"`
// 	Root      string   `json:"rt"`
// 	GroupName string   `json:"gn"`
// }

// type ChatRequest struct {
// 	GroupName string       `json:"gn"`
// 	Targets   []string     `json:"tr"`
// 	Message   Down4Message `json:"msg"`
// }

// type HyperchatRequest struct {
// 	Targets   []string     `json:"tr"`
// 	WordPairs []string     `json:"wp"`
// 	Message   Down4Message `json:"msg"`
// }

// type GroupRequest struct {
// 	Targets    []string     `json:"tr"`
// 	Message    Down4Message `json:"msg"`
// 	GroupID    string       `json:"id"`
// 	GroupName  string       `json:"gn"`
// 	GroupMedia Down4Media   `json:"gm"`
// 	Private    bool         `json:"pv"`
// }

type FireStoreNode struct {
	Identifier string `json:"id" firestore:"id"`
	Neuter     string `json:"neuter,omitempty" firestore:"neuter,omitempty"`
	Type       string `json:"type" firestore:"type"`
	Name       string `json:"name" firestore:"name"`
	Lastname   string `json:"lastName,omitempty" firestore:"lastName,omitempty"`
	ImageID    string `json:"mediaID" firestore:"mediaID"`
	IsPrivate  bool   `json:"isPrivate" firestore:"isPrivate"`
	// Latitude   float32 `json:"latitude,omitempty" firestore:"latitude,omitempty"`
	// Longitude  float32 `json:"longitude,omitempty" firestore:"longitude,omitempty"`
	// Friends    []string `json:"frd,omitempty" firestore:"frd,omitempty"`
	Group []string `json:"group,omitempty" firestore:"group,omitempty"`
	// Messages   []string `json:"msg,omitempty" firestore:"msg,omitempty"`
	// Admins     []string `json:"adm,omitempty" firestore:"adm,omitempty"`
	Publics  []string `json:"publics,omitempty" firestore:"public,omitempty"`
	Privates []string `json:"privates,omitempty" firestore:"private,omitempty"`
	// Words      []string `json:"wrd,omitempty" firestore:"wrd,omitempty"`
}

type FullNode struct {
	Node     map[string]interface{} `json:"node"`
	Metadata map[string]string      `json:"media"`
	Data     string                 `json:"data"`
}

// type FullNode struct {
// 	Neuter     string     `json:"nt,omitempty"`
// 	Identifier string     `json:"id"`
// 	Type       string     `json:"t"`
// 	Name       string     `json:"nm"`
// 	Lastname   string     `json:"ln,omitempty"`
// 	Image      Down4Media `json:"im"`
// 	Private    bool       `json:"pv"`
// 	Latitude   float32    `json:"lat,omitempty"`
// 	Longitude  float32    `json:"lng,omitempty"`
// 	Friends    []string   `json:"frd,omitempty"`
// 	Group      []string   `json:"grp,omitempty"`
// 	Messages   []string   `json:"msg,omitempty"`
// 	Admins     []string   `json:"adm,omitempty"`
// 	Childs     []string   `json:"chl,omitempty"`
// 	Parents    []string   `json:"prt,omitempty"`
// 	Words      []string   `json:"wrd,omitempty"`
// }

// func (fn *FullNode) ToFireStoreNode() *FireStoreNode {
// 	return &FireStoreNode{
// 		Type:       fn.Type,
// 		Name:       fn.Name,
// 		Lastname:   fn.Lastname,
// 		Identifier: fn.Identifier,
// 		Private:    fn.Private,
// 		ImageID:    fn.Image.Identifier,
// 		Latitude:   fn.Latitude,
// 		Longitude:  fn.Longitude,
// 		Friends:    fn.Friends,
// 		Group:      fn.Group,
// 		Messages:   fn.Messages,
// 		Admins:     fn.Admins,
// 		Childs:     fn.Childs,
// 		Parents:    fn.Parents,
// 		Words:      fn.Words,
// 	}
// }

type UserInfo struct {
	Secret   string `json:"secret"`
	Activity int64  `json:"activity"`
	Token    string `json:"token"`
}

type InitUserInfo struct {
	Identifier string `json:"id"`
	Name       string `json:"name"`
	Lastname   string `json:"lastName"`
	Secret     string `json:"secret"`
	Token      string `json:"token"`
	Neuter     string `json:"neuter"`
	Image      string `json:"mediaID"`
}
