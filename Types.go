package backend

type Down4Media struct {
	Identifier string            `json:"id"`
	Data       []byte            `json:"d"`
	Metadata   map[string]string `json:"md"`
}

type Down4Message struct {
	Type        string `json:"t"`
	Root        string `json:"rt,omitempty"`
	MessageID   string `json:"id"`
	SenderID    string `json:"s"`
	ForwarderID string `json:"f,omitempty"`
	Text        string `json:"txt,omitempty"`
	Timestamp   int64  `json:"ts"`
	Replies     string `json:"r,omitempty"`
	Nodes       string `json:"n,omitempty"`
	MediaID     string `json:"m,omitempty"`
	PaymentID   string `json:"p,omitempty"`
}

type PaymentRequest struct {
	Sender    string   `json:"s"`
	Targets   []string `json:"tr"`
	Payment   []byte   `json:"pay"`
	PaymentID string   `json:"id"`
}

type PingRequest struct {
	Targets  []string `json:"tr"`
	Text     string   `json:"txt"`
	SenderID string   `json:"id"`
}

type SnipRequest struct {
	Targets []string     `json:"tr"`
	Message Down4Message `json:"msg"`
	Media   Down4Media   `json:"m"`
}

type ChatRequest struct {
	GroupName  string       `json:"gn"`
	Targets    []string     `json:"tr"`
	Message    Down4Message `json:"msg"`
	Media      Down4Media   `json:"m"`
	WithUpload bool         `json:"wu"`
}

type HyperchatRequest struct {
	Targets    []string     `json:"tr"`
	WordPairs  []string     `json:"wp"`
	Message    Down4Message `json:"msg"`
	Media      Down4Media   `json:"m"`
	WithUpload bool         `json:"wu"`
}

type GroupRequest struct {
	Targets    []string     `json:"tr"`
	Message    Down4Message `json:"msg"`
	Media      Down4Media   `json:"m"`
	WithUpload bool         `json:"wu"`
	GroupID    string       `json:"id"`
	GroupName  string       `json:"gn"`
	GroupMedia Down4Media   `json:"gm"`
	Private    bool         `json:"pv"`
}

type FireStoreNode struct {
	Neuter     string   `json:"nt,omitempty" firestore:"nt,omitempty"`
	Identifier string   `json:"id" firestore:"id"`
	Type       string   `json:"t" firestore:"t"`
	Name       string   `json:"nm" firestore:"nm"`
	Lastname   string   `json:"ln,omitempty" firestore:"ln,omitempty"`
	ImageID    string   `json:"im" firestore:"im"`
	Private    bool     `json:"pv" firestore:"pv"`
	Latitude   float32  `json:"lat,omitempty" firestore:"lat,omitempty"`
	Longitude  float32  `json:"lng,omitempty" firestore:"lng,omitempty"`
	Friends    []string `json:"frd,omitempty" firestore:"frd,omitempty"`
	Group      []string `json:"grp,omitempty" firestore:"grp,omitempty"`
	Messages   []string `json:"msg,omitempty" firestore:"msg,omitempty"`
	Admins     []string `json:"adm,omitempty" firestore:"adm,omitempty"`
	Childs     []string `json:"chl,omitempty" firestore:"chl,omitempty"`
	Parents    []string `json:"prt,omitempty" firestore:"prt,omitempty"`
	Words      []string `json:"wrd,omitempty" firestore:"wrd,omitempty"`
}

type FullNode struct {
	Neuter     string     `json:"nt,omitempty"`
	Identifier string     `json:"id"`
	Type       string     `json:"t"`
	Name       string     `json:"nm"`
	Lastname   string     `json:"ln,omitempty"`
	Image      Down4Media `json:"im"`
	Private    bool       `json:"pv"`
	Latitude   float32    `json:"lat,omitempty"`
	Longitude  float32    `json:"lng,omitempty"`
	Friends    []string   `json:"frd,omitempty"`
	Group      []string   `json:"grp,omitempty"`
	Messages   []string   `json:"msg,omitempty"`
	Admins     []string   `json:"adm,omitempty"`
	Childs     []string   `json:"chl,omitempty"`
	Parents    []string   `json:"prt,omitempty"`
	Words      []string   `json:"wrd,omitempty"`
}

func (fn *FullNode) ToFireStoreNode() *FireStoreNode {
	return &FireStoreNode{
		Name:       fn.Name,
		Lastname:   fn.Lastname,
		Identifier: fn.Identifier,
		Private:    fn.Private,
		ImageID:    fn.Image.Identifier,
		Latitude:   fn.Latitude,
		Longitude:  fn.Longitude,
		Friends:    fn.Friends,
		Group:      fn.Group,
		Messages:   fn.Messages,
		Admins:     fn.Admins,
		Childs:     fn.Childs,
		Parents:    fn.Parents,
		Words:      fn.Words,
	}
}

type UserInfo struct {
	Secret   string            `json:"sh"`
	Activity int64             `json:"ac"`
	Token    string            `json:"tkn"`
	Messages map[string]string `json:"m"`
	Snips    map[string]string `json:"s"`
	Payments map[string]string `json:"p"`
}

type InitUserInfo struct {
	Secret     string     `json:"sh"`
	Token      string     `json:"tkn"`
	Neuter     string     `json:"nt"`
	Identifier string     `json:"id"`
	Name       string     `json:"nm"`
	Lastname   string     `json:"ln"`
	Image      Down4Media `json:"im"`
}
