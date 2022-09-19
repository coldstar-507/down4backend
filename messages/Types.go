package messages

import (
	"strconv"
	"strings"
)

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
	Root        string     `json:"rt"`
	MessageID   string     `json:"id"`
	SenderID    string     `json:"s"`
	ForwarderID string     `json:"f"`
	Text        string     `json:"txt"`
	Timestamp   int64      `json:"ts"`
	Reactions   string     `json:"r"`
	Nodes       string     `json:"n"`
	Media       Down4Media `json:"m"`
}

func (msg *Down4Message) ToRTDB() *map[string]interface{} {
	return &map[string]interface{}{
		"rt":  msg.Root,
		"id":  msg.MessageID,
		"s":   msg.SenderID,
		"f":   msg.ForwarderID,
		"txt": msg.Text,
		"ts":  msg.Timestamp,
		"r":   msg.Reactions,
		"n":   msg.Nodes,
		"m":   map[string]string{"id": msg.Media.Identifier},
	}
}

type MessageRequest struct {
	WithUpload bool         `json:"wu"`
	GroupNode  PseudoNode   `json:"g"`
	Message    Down4Message `json:"msg"`
	Targets    []string     `json:"trgts"`
}

type PingRequest struct {
	Targets []string     `json:"trgts"`
	Message Down4Message `json:"msg"`
}

type SnipRequest struct {
	Targets []string     `json:"trgts`
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
	Message    Down4Message `json:"msg"`
	WithUpload bool         `json:"wu"`
	Name       string       `json:"nm"`
	LastName   string       `json:"ln"`
}

type GroupRequest struct {
	Targets    []string     `json:"trgts"`
	Message    Down4Message `json:"msg"`
	WithUpload bool         `json:"wu"`
	GroupName  string       `json:"gn"`
	GroupID    string       `json:"id"`
	GroupMedia Down4Media   `json:"m"`
}

func (req *MessageRequest) ToNotification() *map[string]string {

	m := make(map[string]string)

	m["t"] = "chat"

	if req.IsGroup {
		m["t"] = "group" // simply override
		var friends []string
		if req.Message.ForwarderID != "" {
			friends = append(friends, req.Message.ForwarderID)
		} else {
			friends = append(friends, req.Message.SenderID)
		}
		m["gfr"] = strings.Join(friends, " ")
		m["gid"] = req.GroupNode.Identifier
		m["gnm"] = req.GroupNode.Name
		m["gim"] = req.GroupNode.Image.Identifier
	} else if req.IsHyperchat {
		m["t"] = "hyperchat" // simply override
		var friends []string
		if req.Message.ForwarderID != "" {
			friends = append(friends, req.Message.ForwarderID)
		} else {
			friends = append(friends, req.Message.SenderID)
		}
		m["hcfr"] = strings.Join(friends, " ")
		m["hcid"] = req.GroupNode.Identifier
		m["hcnm"] = req.GroupNode.Name
		m["hcln"] = req.GroupNode.LastName
		m["hcim"] = req.GroupNode.Image.Identifier
	}

	m["rt"] = req.Message.Root
	m["msgid"] = req.Message.MessageID
	m["sdrid"] = req.Message.SenderID
	m["sdrnm"] = req.Message.SenderName
	m["sdrln"] = req.Message.SenderLastName
	m["sdrtn"] = req.Message.SenderThumbnail
	m["fdrid"] = req.Message.ForwarderID
	m["fdrnm"] = req.Message.ForwarderName
	m["fdrln"] = req.Message.ForwarderLastName
	m["fdrtn"] = req.Message.ForwarderThumbnail
	m["txt"] = req.Message.Text
	m["ts"] = strconv.FormatInt(req.Message.Timestamp, 10)
	m["ischt"] = strconv.FormatBool(req.Message.IsChat)
	m["r"] = strings.Join(req.Message.Reactions, " ")
	m["n"] = strings.Join(req.Message.Nodes, " ")
	m["mid"] = req.Message.Media.Identifier

	return &m
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
