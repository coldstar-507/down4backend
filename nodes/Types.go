package nodes

type Down4Media struct {
	Identifier string            `json:"id"`
	Data       []byte            `json:"d"`
	Metadata   map[string]string `json:"md"`
}

type FireStoreNode struct {
	Neuter     string   `json:"nt"`
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
	Posts      []string `json:"pst"`
	Group      []string `json:"grp"`
	Childs     []string `json:"chl"`
	Parents    []string `json:"prt"`
	Words      []string `json:"wrd"`
}

type OutputNode struct {
	Neuter     string     `json:"nt"`
	Identifier string     `json:"id"`
	Type       string     `json:"t"`
	Name       string     `json:"nm"`
	Lastname   string     `json:"ln"`
	Image      Down4Media `json:"im"`
	Latitude   float32    `json:"lat"`
	Longitude  float32    `json:"lng"`
	Friends    []string   `json:"frd"`
	Messages   []string   `json:"msg"`
	Posts      []string   `json:"pst"`
	Admins     []string   `json:"adm"`
	Group      []string   `json:"grp"`
	Childs     []string   `json:"chl"`
	Parents    []string   `json:"prt"`
	Words      []string   `json:"wrd"`
}
