package messages

type Down4Media struct {
	Identifier string            `json:"id"`
	Data       []byte            `json:"d"`
	Metadata   map[string]string `json:"md"`
}

type PseudoNode struct {
	Identifier string     `json:"id"`
	Image      Down4Media `json:"im"`
	Name       string     `json:"nm"`
	LastName   string     `json:"ln"`
	Group      []string   `json:"grp"`
}

type Down4Message struct {
	Type        string               `json:"t"`
	Root        string               `json:"rt"`
	MessageID   string               `json:"id"`
	SenderID    string               `json:"s"`
	ForwarderID string               `json:"f"`
	Text        string               `json:"txt"`
	Timestamp   int64                `json:"ts"`
	Replies     string               `json:"r"`
	Nodes       string               `json:"n"`
	Media       Down4Media           `json:"m"`
	Payment     Down4InternetPayment `json:"pay"`
}

func (msg *Down4Message) ToRTDB() *map[string]interface{} {
	m := make(map[string]interface{})

	m["id"] = msg.MessageID
	m["ts"] = msg.Timestamp
	m["s"] = msg.SenderID
	m["t"] = msg.Type

	if len(msg.Payment.PaymentID) > 0 {
		m["pay"] = map[string]string{"id": msg.Payment.PaymentID}
	}

	if len(msg.Root) > 0 {
		m["rt"] = msg.Root
	}

	if len(msg.ForwarderID) > 0 {
		m["f"] = msg.ForwarderID
	}

	if len(msg.Replies) > 0 {
		m["r"] = msg.Replies
	}

	if len(msg.Nodes) > 0 {
		m["n"] = msg.Nodes
	}

	if len(msg.Media.Identifier) > 0 {
		m["m"] = map[string]string{"id": msg.Media.Identifier}
	}

	if len(msg.Text) > 0 {
		m["txt"] = msg.Text
	}

	return &m
}

type PingRequest struct {
	Targets []string     `json:"trgts"`
	Message Down4Message `json:"msg"`
}

type SnipRequest struct {
	Targets []string     `json:"trgts"`
	Message Down4Message `json:"msg"`
}

type ChatRequest struct {
	GroupName  string       `json:"gn"`
	Targets    []string     `json:"trgts"`
	Message    Down4Message `json:"msg"`
	WithUpload bool         `json:"wu"`
}

type HyperchatRequest struct {
	Targets    []string     `json:"trgts"`
	WordPairs  []string     `json:"wp"`
	Message    Down4Message `json:"msg"`
	WithUpload bool         `json:"wu"`
}

type GroupRequest struct {
	Targets    []string     `json:"trgts"`
	Message    Down4Message `json:"msg"`
	WithUpload bool         `json:"wu"`
	GroupName  string       `json:"gn"`
	GroupID    string       `json:"id"`
	GroupMedia Down4Media   `json:"m"`
}

type FireStoreNode struct {
	Identifier string   `json:"id"`
	Type       string   `json:"t"`
	Name       string   `json:"nm"`
	Lastname   string   `json:"ln"`
	ImageID    string   `json:"im"`
	Latitude   float32  `json:"lat"`
	Longitude  float32  `json:"lng"`
	Friends    []string `json:"frd"`
	Messages   []string `json:"msg"`
	Admins     []string `json:"adm"`
	Childs     []string `json:"chl"`
	Parents    []string `json:"prt"`
	Words      []string `json:"wrd"`
}

type Down4InternetPayment struct {
	Sender    string   `json:"s"`
	Targets   []string `json:"trgts"`
	Payment   []byte   `json:"pay"`
	PaymentID string   `json:"id"`
}
