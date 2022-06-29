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
}

type Down4Message struct {
	Root               string     `json:"rt"`
	MessageID          string     `json:"msgid"`
	SenderID           string     `json:"sdrid"`
	SenderName         string     `json:"sdrnm"`
	SenderLastName     string     `json:"sdrln"`
	SenderThumbnail    string     `json:"sdrtn"`
	ForwarderID        string     `json:"fdrid"`
	ForwarderName      string     `json:"fdrnm"`
	ForwarderLastName  string     `json:"fdrln"`
	ForwarderThumbnail string     `json:"fdrtn"`
	Text               string     `json:"txt"`
	Timestamp          int64      `json:"ts"`
	IsChat             bool       `json:"ischt"`
	Reactions          []string   `json:"r"`
	Nodes              []string   `json:"n"`
	Media              Down4Media `json:"m"`
}

type FriendRequest struct {
	RequesterID       string   `json:"id"`
	RequesterName     string   `json:"nm"`
	RequesterLastName string   `json:"ln"`
	Targets           []string `json:"trgts"`
}

type FriendRequestAccepted struct {
	Sender   string `json:"sender"`
	Accepter string `json:"accepter"`
}

type MessageRequest struct {
	WithUpload  bool         `json:"wu"`
	IsHyperchat bool         `json:"ihc"`
	IsGroup     bool         `json:"ig"`
	GroupNode   PseudoNode   `json:"g"`
	Message     Down4Message `json:"msg"`
	Targets     []string     `json:"trgts"`
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
