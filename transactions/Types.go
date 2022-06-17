package transaction

type Transaction struct {
	Targets  []string `json:"targets"`
	Satoshis int      `json:"sats"`
	From     string   `json:"from"`
	Each     bool     `json:"each"`
	Low      int      `json:"low"`
	High     int      `json:"high"`
}

type PublicMoneyInfo struct {
	Neuter string `json:"nt"`
	Change uint32 `json:"cg"`
	Index  uint32 `json:"ix"`
}
