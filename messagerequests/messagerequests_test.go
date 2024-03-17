package messagerequests

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http/httptest"
	"strconv"
	// "sync"
	"testing"

	"github.com/coldstar-507/down4backend/utils"
)

func TestMakeChatNum(t *testing.T) {
	m64 := strconv.FormatUint(math.MaxUint64, 10)
	m32 := strconv.FormatUint(math.MaxUint32, 10)
	t.Logf("m64=%s, len=%d\n", m64, len(m64))
	t.Logf("m32=%s, len=%d\n", m32, len(m32))
}

func TestProcessMessage(t *testing.T) {
	mqs := []*MessageRequest{{
		Targets: []MessageTarget{{
			UserId:    "hashirama-america-0r",
			DeviceId:  "HKobPYpVoR1RxfXJt2CsJ64JuCX",
			Token:     "e7blxHf6SHKTcjc-mF6La1:APA91bEmqrp1EOVvibwbdBf9QJ4fLQJzGApKbigTqrsiYPR7KZZMPW0MEUoWQeuqOG9xQOBqswww8uamb-wE5yMGD-mR_A7h9CgHJuNvVc71NA-tdexF-r97FAf1UW0kiTin2-10ouHB",
			ShowNotif: true,
			DoPush:    true,
		}, {
			UserId:    "scammer-america-0r",
			DeviceId:  "2VvkskVWSteL5duLxkyFmtBZXnHT",
			Token:     "frZb09J2Sdm_kTbV8BCCrL:APA91bEJ4XVOglMtOV7-x2qgVJWhQGQmk3uIl6v0vtlak0XTpPiACS9LznYno--76tZTCvsnBjc0mYLDrzfhGCzMwN7daBiG0OHYZ7mDXze7PPbBuEacbe4kI-GZhSMKIeijERW68pof",
			ShowNotif: true,
			DoPush:    true,
		}},
		Msg: map[string]string{
			"id":        "ABCDEFG@scammer-america-0r^hashirama-america-0rc",
			"type":      "chat",
			"senderId":  "scammer-america-0r",
			"tag":       "YmwogGuHRYW",
			"txt":       "It's all fucking based",
			"timestamp": strconv.FormatInt(utils.UnixMilli(), 10),
		},
		Push:   "",
		Header: "Scammer",
		Body:   "I will tell you right away: you can't talk before me",
		Sender: "scammer-america-0r",
		Root:   "scammer-america-0r^hashirama-america-0r",
	},
		{
			Targets: []MessageTarget{{
				UserId:    "hashirama-america-0r",
				DeviceId:  "HKobPYpVoR1RxfXJt2CsJ64JuCX",
				Token:     "e7blxHf6SHKTcjc-mF6La1:APA91bEmqrp1EOVvibwbdBf9QJ4fLQJzGApKbigTqrsiYPR7KZZMPW0MEUoWQeuqOG9xQOBqswww8uamb-wE5yMGD-mR_A7h9CgHJuNvVc71NA-tdexF-r97FAf1UW0kiTin2-10ouHB",
				ShowNotif: true,
				DoPush:    true,
			}, {
				UserId:    "scammer-america-0r",
				DeviceId:  "2VvkskVWSteL5duLxkyFmtBZXnHT",
				Token:     "frZb09J2Sdm_kTbV8BCCrL:APA91bEJ4XVOglMtOV7-x2qgVJWhQGQmk3uIl6v0vtlak0XTpPiACS9LznYno--76tZTCvsnBjc0mYLDrzfhGCzMwN7daBiG0OHYZ7mDXze7PPbBuEacbe4kI-GZhSMKIeijERW68pof",
				ShowNotif: true,
				DoPush:    true,
			}},
			Msg: map[string]string{
				"id":        "jlkasdf@scammer-america-0r^hashirama-america-0rc",
				"type":      "chat",
				"senderId":  "hashirama-america-0r",
				"tag":       "YmwogGuHRYW",
				"txt":       "Listening to Flavour Trip ^_^",
				"timestamp": strconv.FormatInt(utils.UnixMilli(), 10),
			},
			Push:   "",
			Header: "Hashirama",
			Body:   "Degrowth is a scam",
			Sender: "scammer-america-0r",
			Root:   "scammer-america-0r^hashirama-america-0r",
		},
		{
			Targets: []MessageTarget{{
				UserId:    "hashirama-america-0r",
				DeviceId:  "HKobPYpVoR1RxfXJt2CsJ64JuCX",
				Token:     "e7blxHf6SHKTcjc-mF6La1:APA91bEmqrp1EOVvibwbdBf9QJ4fLQJzGApKbigTqrsiYPR7KZZMPW0MEUoWQeuqOG9xQOBqswww8uamb-wE5yMGD-mR_A7h9CgHJuNvVc71NA-tdexF-r97FAf1UW0kiTin2-10ouHB",
				ShowNotif: true,
				DoPush:    true,
			}, {
				UserId:    "scammer-america-0r",
				DeviceId:  "2VvkskVWSteL5duLxkyFmtBZXnHT",
				Token:     "frZb09J2Sdm_kTbV8BCCrL:APA91bEJ4XVOglMtOV7-x2qgVJWhQGQmk3uIl6v0vtlak0XTpPiACS9LznYno--76tZTCvsnBjc0mYLDrzfhGCzMwN7daBiG0OHYZ7mDXze7PPbBuEacbe4kI-GZhSMKIeijERW68pof",
				ShowNotif: true,
				DoPush:    true,
			}},
			Msg: map[string]string{
				"id":        "jlkasdf@scammer-america-0r^hashirama-america-0rc",
				"type":      "chat",
				"senderId":  "hashirama-america-0r",
				"tag":       "YmwogGuHRYW",
				"txt":       "CRAIG WRIGHT IS MY HERO",
				"timestamp": strconv.FormatInt(utils.UnixMilli(), 10),
			},
			Push:   "",
			Header: "Hashirama",
			Body:   "There are actually 0 studies proving this",
			Sender: "scammer-america-0r",
			Root:   "scammer-america-0r^hashirama-america-0r",
		},
	}

	// var wg sync.WaitGroup
	for _, mq := range mqs {
		//wg.Add(1)
		//go func(mq_ *MessageRequest) {
		//defer wg.Done()
		js, _ := json.Marshal(mq)
		r := httptest.NewRequest("POST", "/", bytes.NewReader(js))
		w := httptest.NewRecorder()
		ProcessMessage(w, r)
		//}(mq)
	}
	//	wg.Wait()
	t.Log("We Are DONE")
}
