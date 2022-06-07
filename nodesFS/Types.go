package nodesFS

type Down4Media struct {
	Identifier string `json:"id"`
	Data       []byte `json:"d"`
}

type PublicMoneyInfo struct {
	NeuterString string `json:"nt"`
	Index        uint32 `json:"ix"`
	Change       uint32 `json:"cg"`
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
}

type OutputNode struct {
	Identifier string     `json:"id"`
	Type       string     `json:"t"`
	Name       string     `json:"nm"`
	Lastname   string     `json:"ln"`
	Image      Down4Media `json:"im"`
	Latitude   float32    `json:"lat"`
	Longitude  float32    `json:"lng"`
	Friends    []string   `json:"frd"`
	Messages   []string   `json:"msg"`
	Admins     []string   `json:"adm"`
	Childs     []string   `json:"chl"`
	Parents    []string   `json:"prt"`
}

type UserInfo struct {
	Secret   string `json:"secret"`
	Activity int64  `json:"activity"`
	Token    string `json:"token"`
}
