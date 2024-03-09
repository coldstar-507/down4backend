package messagerequests

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func TestMessageRequests(t *testing.T) {
	mq := &mq{
		Mts: []mt{{
			UserID:    "comfy~america~0",
			DeviceID:  "H0iCjoPoZwkXVRAVW61JWw==",
			Token:     "dG4vEEjESBC0uor4k1kd-1:APA91bEiOqFZ7sxK6oQz43FDo12UNoobtykMJovYiGTMgxSReCaIlQtc515_T2ORZxH1AFqooFOQuDYc8DCB5hkZLHJz9RfdZvws68KD1avDCQq8gfG85J1YhLwh1z2vbFuM9iwsNnXo",
			ShowNotif: true,
			DoPush:    false,
		}, {
			UserID:    "hashirama~america~0",
			DeviceID:  "XkRBcZDVb5872ZSKiYfsFw==",
			Token:     "fd-p8CbjQHuCwgM4LHktZi:APA91bH3ybHbor6p72PkSEwrPrO8RaMWJpfnznadscy-JhTki4uz7_Ic03kp2ytGqMrMi_0-QFJCXULh3fOUZzrqPdFCUZz_JPJovoe_VIZQ_k7wCslH2zp61NmyxF6Bgez2FIK_Symf",
			ShowNotif: true,
			DoPush:    false,
		}},
		Push:     "m!-Nc5ttBMeCAYH01DtkMB~america~1",
		Header:   "Mathematics",
		Body:     "Hashirama Senju: Piece of shit.",
		SenderID: "hashirama~america~0",
		RootID:   "-Nc5r8EPCuqp5M5s-GDl~america~1",
	}

	js, _ := json.Marshal(mq)

	r := httptest.NewRequest("POST", "/", bytes.NewReader(js))
	w := httptest.NewRecorder()

	ProcessMessage(w, r)

}
