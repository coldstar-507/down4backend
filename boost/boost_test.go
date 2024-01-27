package boost

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/coldstar-507/down4backend/utils"
	"github.com/mmcloughlin/geohash"
)

type boostd struct {
	center latlon
	perim  []latlon
}

func TestLayers(t *testing.T) {
	t.Log("Starting TestLayers")

	jonesPos := latlon{
		Lat: 44.77231091565398,
		Lon: -72.5507737118483,
	}
	rugPos := latlon{
		Lat: 45.904258512242286,
		Lon: -72.2507737118483,
	}

	jonesHash := geohash.EncodeWithPrecision(jonesPos.Lat, jonesPos.Lon, 4)
	rugHash := geohash.EncodeWithPrecision(rugPos.Lat, rugPos.Lon, 4)

	t.Logf("jonesHash=%s\nrugHash=%s\n", jonesHash, rugHash)

	a := area{
		Center: latlon{
			Lat: 45.904258512242286,
			Lon: -72.2507737118483,
		},
		Perim: []latlon{
			{
				Lat:     43.77231091565398,
				Lon:     -72.2507737118483,
				RefDist: 237.0617566126798,
			},
			{
				Lat:     44.06251444297506,
				Lon:     -73.7543096676021,
				RefDist: 236.4707305606959,
			},
			{
				Lat:     44.848147251478814,
				Lon:     -74.85497437822046,
				RefDist: 234.8581857803903,
			},
			{
				Lat:     45.904258512242286,
				Lon:     -75.25784562335588,
				RefDist: 232.6615328018019,
			},
			{
				Lat:     46.940654121329025,
				Lon:     -74.85497437822046,
				RefDist: 230.47345405250886,
			},
			{
				Lat:     47.686868561809305,
				Lon:     -73.7543096676021,
				RefDist: 228.87801053290622,
			},
			{
				Lat:     47.9573693727533,
				Lon:     -72.2507737118483,
				RefDist: 228.29551152766882,
			},
			{
				Lat:     47.686868561809305,
				Lon:     -70.74723775609455,
				RefDist: 228.87801053290406,
			},
			{
				Lat:     46.940654121329025,
				Lon:     -69.64657304547613,
				RefDist: 230.47345405250886,
			},
			{
				Lat:     45.904258512242286,
				Lon:     -69.24370180034076,
				RefDist: 232.6615328017986,
			},
			{
				Lat:     44.848147251478814,
				Lon:     -69.64657304547613,
				RefDist: 234.8581857803903,
			},
			{
				Lat:     44.06251444297506,
				Lon:     -70.74723775609453,
				RefDist: 236.47073056069428,
			},
		},
	}
	layers := calcLayers2(a)
	t.Logf("layers: %v\n", layers)

}

func TestBoostRequest(t *testing.T) {
	br := &boostRequest2{
		Token:        "fTC-jAgkRGK95ie31zipgX:APA91bH3_I-g_diBliCsk9wX19E_p0Y02u2jkNqZI-RCVIMqX49xJr6pI5yykqsLvPbraVIhl_UMOIuH7MdR5KsCujK_LYLMgzZ3l-1K-bAVtP9FTjnGalHaqO7OtNEiskQ5K4CggVyj",
		SenderID:     "jones~america~1",
		DeviceID:     "KrFECXUhBZA/rhCHFLIklw==",
		Limit:        1000,
		PricePerHead: 100,
		MaxAge:       39,
		MinAge:       18,
		Genders: []string{
			"male",
			"female",
			"",
		},
		S1:            "hlyIWdhP74MK3TQ8VddELWr7Y40=",
		ChangeAddress: "WJ5SdnBLDUHTbSJZbQ4p+UQbFQw=",
		InputSats:     49049,
		PartialTx:     "AQAAAAG8ZA4k+7JYpuYFPc7p/s/mYvijAZkStljajdeAV/tQFwIAAABrSDBFAiEAx5z3tJ7iG9cepRSTKfKITsmeQsotMNe2KBWrxv3osr4CIFLNLTXgleF9jcbtsTtd84QCALocNcihFIHe9ogcO1d8QiEDZcWAbLkHFO5jSQqWIkqXTerL2Y6p+vWu9zu03Fs8z/L/////AAAAAAA=",
		Areas: []area{
			{
				Center: latlon{
					Lat: 48.84652958761982,
					Lon: -67.52709923696294,
				},
				Perim: []latlon{
					{
						Lat:     48.819904753395235,
						Lon:     -67.52709923696294,
						RefDist: 2.960546488526211,
					},
					{
						Lat:     48.82770445201567,
						Lon:     -67.55570009028744,
						RefDist: 2.9603160596871154,
					},
					{
						Lat:     48.84652958761982,
						Lon:     -67.5675469516299,
						RefDist: 2.9597598031002152,
					},
					{
						Lat:     48.86534764900105,
						Lon:     -67.55570009028744,
						RefDist: 2.9592036142658196,
					},
					{
						Lat:     48.873140273399045,
						Lon:     -67.52709923696294,
						RefDist: 2.958973253182349,
					},
					{
						Lat:     48.86534764900105,
						Lon:     -67.49849838363843,
						RefDist: 2.9592036142658196,
					},
					{
						Lat:     48.84652958761982,
						Lon:     -67.48665152229599,
						RefDist: 2.9597598031002152,
					},
					{
						Lat:     48.82770445201567,
						Lon:     -67.49849838363843,
						RefDist: 2.9603160596871154,
					},
				},
			},
		},
		BoostMessage: map[string]interface{}{
			"id":        "-Nnpq_X7qoB4XGPyXDqo~america~0",
			"type":      "chat",
			"senderID":  "jones~america~1",
			"root":      "boost",
			"nodes":     "",
			"txt":       "Bonne main d'applaudissement.",
			"timestamp": "1704932605217",
		},
	}

	b, _ := json.Marshal(br)
	body := bytes.NewReader(b)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", body)

	HandleBoostRequest(w, r)

	var rsp []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("error unmarshalling res: %v\n", err)
	}

	ids := utils.Map(rsp, func(e map[string]interface{}) string {
		return e["Id"].(string)
	})

	t.Log(ids)
}
