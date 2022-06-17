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

type ChatRequestWithMediaUpload struct {
	Message Down4Message `json:"msg"`
	Targets []string     `json:"trgts"`
}

type ChatRequestWithNotifOnly struct {
	Notif   map[string]string `json:"ntf"`
	Targets []string          `json:"trgts"`
}

type HyperchatRequestWithNotifOnly struct {
	Notif     map[string]string `json:"ntf"`
	Hyperchat PseudoNode        `json:"hc"`
	Targets   []string          `json:"trgts"`
}

type HyperchatRequestWithMediaUpload struct {
	Message   Down4Message `json:"msg"`
	Hyperchat PseudoNode   `json:"hc"`
	Targets   []string     `json:"trgts"`
}

type PingRequest struct {
	Notif   map[string]string `json:"ntf"`
	Targets []string          `json:"trgts"`
}

func (chatReq *ChatRequestWithMediaUpload) ToNotification() *map[string]string {

	m := make(map[string]string)

	m["t"] = "cht"
	m["rt"] = (*chatReq).Message.Root
	m["msgid"] = (*chatReq).Message.MessageID
	m["txt"] = (*chatReq).Message.Text
	m["sdrid"] = (*chatReq).Message.SenderID
	m["sdrnm"] = (*chatReq).Message.SenderName
	m["sdrtn"] = (*chatReq).Message.SenderThumbnail
	m["fdrid"] = (*chatReq).Message.ForwarderID
	m["fdrnm"] = (*chatReq).Message.ForwarderName
	m["fdrtn"] = (*chatReq).Message.ForwarderThumbnail
	m["mid"] = (*chatReq).Message.Media.Identifier
	m["n"] = strings.Join((*chatReq).Message.Nodes, " ")
	m["r"] = strings.Join((*chatReq).Message.Reactions, " ")
	m["ischt"] = strconv.FormatBool((*chatReq).Message.IsChat)
	m["ts"] = strconv.FormatInt((*chatReq).Message.Timestamp, 10)

	return &m
}

func (req *HyperchatRequestWithMediaUpload) ToNotification() *map[string]string {

	m := make(map[string]string)

	m["hcid"] = (*req).Hyperchat.Identifier
	m["hcnm"] = (*req).Hyperchat.Name
	m["hcmid"] = (*req).Hyperchat.Image.Identifier
	m["hcln"] = (*req).Hyperchat.LastName

	m["t"] = "hc"
	m["rt"] = (*req).Message.Root
	m["msgid"] = (*req).Message.MessageID
	m["txt"] = (*req).Message.Text
	m["sdrid"] = (*req).Message.SenderID
	m["sdrnm"] = (*req).Message.SenderName
	m["sdrtn"] = (*req).Message.SenderThumbnail
	m["fdrid"] = (*req).Message.ForwarderID
	m["fdrnm"] = (*req).Message.ForwarderName
	m["fdrtn"] = (*req).Message.ForwarderThumbnail
	m["mid"] = (*req).Message.Media.Identifier
	m["n"] = strings.Join((*req).Message.Nodes, " ")
	m["r"] = strings.Join((*req).Message.Reactions, " ")
	m["ischt"] = strconv.FormatBool((*req).Message.IsChat)
	m["ts"] = strconv.FormatInt((*req).Message.Timestamp, 10)

	return &m
}

// type Down4Message struct {
// 	Root               string     `json:"rt"`
// 	MessageID          string     `json:"msgid"`
// 	SenderID           string     `json:"sdrid"`
// 	SenderName         string     `json:"sdrnm"`
// 	SenderLastName     string     `json:"sdrln"`
// 	SenderThumbnail    string     `json:"sdrtn"`
// 	ForwarderID        string     `json:"fdrid"`
// 	ForwarderName      string     `json:"fdrnm"`
// 	ForwarderLastName  string     `json:"fdrln"`
// 	ForwarderThumbnail string     `json:"fdrtn"`
// 	Text               string     `json:"txt"`
// 	Timestamp          int64      `json:"ts"`
// 	IsChat             bool       `json:"ischt"`
// 	Reactions          []string   `json:"r"`
// 	Nodes              []string   `json:"n"`
// 	Media              Down4Media `json:"m"`
// }

func (msg *Down4Message) ToNotification() *map[string]string {
	m := make(map[string]string)

	m["rt"] = (*msg).Root
	m["msgid"] = (*msg).MessageID
	m["sdrid"] = (*msg).SenderID
	m["sdrnm"] = (*msg).SenderName
	m["sdrln"] = (*msg).SenderLastName
	m["sdrtn"] = (*msg).SenderThumbnail
	m["fdrid"] = (*msg).ForwarderID
	m["fdrnm"] = (*msg).ForwarderName
	m["fdrln"] = (*msg).ForwarderLastName
	m["fdrtn"] = (*msg).ForwarderThumbnail
	m["txt"] = (*msg).Text
	m["ts"] = strconv.FormatInt((*msg).Timestamp, 10)
	m["ischt"] = strconv.FormatBool((*msg).IsChat)
	m["r"] = strings.Join((*msg).Reactions, " ")
	m["n"] = strings.Join((*msg).Nodes, " ")
	m["mid"] = (*msg).Media.Identifier

	return &m
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

	if (*req).IsGroup {
		m["gid"] = (*req).GroupNode.Identifier
		m["gnm"] = (*req).GroupNode.Name
		m["gln"] = (*req).GroupNode.LastName
		m["gim"] = (*req).GroupNode.Image.Identifier
	} else if (*req).IsHyperchat {
		m["hcid"] = (*req).GroupNode.Identifier
		m["hcnm"] = (*req).GroupNode.Name
		m["hcln"] = (*req).GroupNode.LastName
		m["hcim"] = (*req).GroupNode.Image.Identifier
	}

	m["rt"] = (*req).Message.Root
	m["msgid"] = (*req).Message.MessageID
	m["sdrid"] = (*req).Message.SenderID
	m["sdrnm"] = (*req).Message.SenderName
	m["sdrln"] = (*req).Message.SenderLastName
	m["sdrtn"] = (*req).Message.SenderThumbnail
	m["fdrid"] = (*req).Message.ForwarderID
	m["fdrnm"] = (*req).Message.ForwarderName
	m["fdrln"] = (*req).Message.ForwarderLastName
	m["fdrtn"] = (*req).Message.ForwarderThumbnail
	m["txt"] = (*req).Message.Text
	m["ts"] = strconv.FormatInt((*req).Message.Timestamp, 10)
	m["ischt"] = strconv.FormatBool((*req).Message.IsChat)
	m["r"] = strings.Join((*req).Message.Reactions, " ")
	m["n"] = strings.Join((*req).Message.Nodes, " ")
	m["mid"] = (*req).Message.Media.Identifier

	return &m
}

// class MessageRequest {
// 	final bool withUpload;
// 	final Node? groupOrHyperchat;
// 	final Down4Message msg;
// 	final List<Identifier> targets;
// 	MessageRequest({
// 	  required this.msg,
// 	  required this.targets,
// 	  this.groupOrHyperchat,
// 	  this.withUpload = false,
// 	});
// 	Map<String, dynamic> toJson() => {
// 		  if (groupOrHyperchat != null)
// 			"g": {
// 			  "id": groupOrHyperchat!.id,
// 			  "im": groupOrHyperchat!.image,
// 			  "nm": groupOrHyperchat!.name,
// 			  if (groupOrHyperchat!.lastName != null)
// 				"ln": groupOrHyperchat!.lastName,
// 			},
// 		  if (withUpload) "msg": msg.toJson() else "ntf": msg.toNotif(),
// 		  "trgts": targets,
// 		};
//   }
