package backend

type Down4Media struct {
	Identifier string            `json:"id"`
	Data       string            `json:"d,omitempty"`
	Thumbnail  string            `json:"tn,omitempty"`
	NetworkURL string            `json:"nw,omitempty"`
	Metadata   map[string]string `json:"md"`
}

// func (m Down4Media) Metadata() *map[string]string {
// 	return &map[string]string{
// 		"o":   m.Owner,
// 		"id":  m.Identifier,
// 		"ts":  strconv.FormatInt(m.Timestamp, 10),
// 		"ar":  strconv.FormatFloat(m.AspectRatio, 'f', 4, 32),
// 		"vid": strconv.FormatBool(m.IsVideo),
// 		"trv": strconv.FormatBool(m.IsReversed),
// 		"shr": strconv.FormatBool(m.IsShareable),
// 		"ptv": strconv.FormatBool(m.IsPaidToView),
// 		"sqr": strconv.FormatBool(m.IsSquared),
// 		"pto": strconv.FormatBool(m.IsPaiedToOwn),
// 		"txt": m.Text,
// 	}
// }

// type MediaMetadata struct {
// 	Owner       string `json:"o"`
// 	AspectRatio string `json:"ar"`
// 	IsPayToView string `json:"ptv"`
// 	IsPayToOwn  string `json:"pto"`
// 	IsShareable string `json:"shr"`
// 	IsSquare    string `json:"sqr"`
// 	IsVideo     string `json:"vid"`
// 	Timestamp   string `json:"ts"`
// 	Text        string `json:"txt"`
// }

type Down4Message struct {
	Root        string `json:"rt,omitempty"`
	MessageID   string `json:"id"`
	SenderID    string `json:"s"`
	ForwarderID string `json:"f,omitempty"`
	Text        string `json:"txt,omitempty"`
	Timestamp   int64  `json:"ts"`
	Replies     string `json:"r,omitempty"`
	Nodes       string `json:"n,omitempty"`
	MediaID     string `json:"m,omitempty"`
}

type PaymentRequest struct {
	Sender     string   `json:"s"`
	Targets    []string `json:"tr"`
	Payment    []byte   `json:"pay"`
	Identifier string   `json:"id"`
	TextNote   string   `json:"txt"`
}

type PingRequest struct {
	Targets  []string `json:"tr"`
	Text     string   `json:"txt"`
	SenderID string   `json:"s"`
}

type SnipRequest struct {
	Sender    string   `json:"s"`
	MediaID   string   `json:"m"`
	Targets   []string `json:"tr"`
	Root      string   `json:"rt"`
	GroupName string   `json:"gn"`
}

type ChatRequest struct {
	GroupName string       `json:"gn"`
	Targets   []string     `json:"tr"`
	Message   Down4Message `json:"msg"`
}

type HyperchatRequest struct {
	Targets   []string     `json:"tr"`
	WordPairs []string     `json:"wp"`
	Message   Down4Message `json:"msg"`
}

type GroupRequest struct {
	Targets    []string     `json:"tr"`
	Message    Down4Message `json:"msg"`
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
		Type:       fn.Type,
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
