package boost

import (
	"bytes"
	"context"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"

	"cloud.google.com/go/firestore"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/coldstar-507/down4backend/bsv"
	"github.com/coldstar-507/down4backend/messagerequests"
	"github.com/coldstar-507/down4backend/server"
	"github.com/coldstar-507/down4backend/utils"

	"github.com/mmcloughlin/geohash"
)

const taal_api_key = "testnet_3860616b1cf1bb23110db44440f65899"

type latlon struct {
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	RefDist float64 `json:"refDist"`
}

const precision = 4

func init() {
	ctx := context.Background()
	server.ServerInit(ctx)
}

func geoDist(ll1, ll2 latlon) float64 {
	lat1, lon1, lat2, lon2 := ll1.Lat, ll1.Lon, ll2.Lat, ll2.Lon
	R := 6371.0                   // Radius of the earth in km
	dLat := degToRad(lat2 - lat1) // degToRad below
	dLon := degToRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c // Distance in km
	return d
}

func geoBearing(ll1, ll2 latlon) float64 {
	lat1, lon1, lat2, lon2 := ll1.Lat, ll1.Lon, ll2.Lat, ll2.Lon
	// Convert degrees to radians
	lat1Rad := degToRad(lat1)
	lon1Rad := degToRad(lon1)
	lat2Rad := degToRad(lat2)
	lon2Rad := degToRad(lon2)

	// Calculate angle using spherical law of cosines
	angle := math.Acos(math.Sin(lat1Rad)*math.Sin(lat2Rad) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Cos(lon2Rad-lon1Rad))

	// Convert angle from radians to degrees
	angle = radToDeg(angle)

	return angle
}

func degToRad(deg float64) float64 {
	return deg * (math.Pi / 180)
}

func radToDeg(rad float64) float64 {
	return rad * 180.0 / math.Pi
}

func closest(l []latlon, p latlon) latlon {
	ix, smallDist := int(0), math.MaxFloat64
	for i, x := range l {
		dist := geoDist(x, p)
		if dist < smallDist {
			smallDist = dist
			ix = i
		}
	}
	return l[ix]
}

func closest2(l []latlon, p latlon) []latlon {
	ix, smallDist := int(0), math.MaxFloat64
	ix2, smallDist2 := int(0), math.MaxFloat64
	for i, x := range l {
		dist := geoDist(x, p)
		if dist < smallDist {
			smallDist = dist
			ix = i
		}
		if dist > smallDist && dist < smallDist2 {
			smallDist2 = dist
			ix2 = i
		}
	}
	return []latlon{l[ix], l[ix2]}
}

func validHash3(a area, hash string) bool {
	box := geohash.BoundingBox(hash)
	bclat, bclon := box.Center()
	boxCenter := latlon{Lat: bclat, Lon: bclon}
	bounds := []latlon{
		{Lat: box.MaxLat, Lon: box.MaxLng},
		{Lat: box.MaxLat, Lon: box.MinLng},
		{Lat: box.MinLat, Lon: box.MaxLng},
		{Lat: box.MinLat, Lon: box.MinLng},
	}

	twoClosest := closest2(a.Perim, boxCenter)
	for _, b := range bounds {
		valid := utils.Any(twoClosest, func(ll latlon) bool {
			return geoDist(b, a.Center) < ll.RefDist
		})
		if valid {
			return true
		}
	}
	return false
}

func calcLayers2(a area) [][]string {
	clat, clon := a.Center.Lat, a.Center.Lon
	centerHash := geohash.EncodeWithPrecision(clat, clon, precision)
	layers := [][]string{{centerHash}}
	flat := []string{centerHash}

	var getLayers func([]string) [][]string
	getLayers = func(l []string) [][]string {
		curLayer := make([]string, 0)
		for _, e := range l {
			nbs := geohash.Neighbors(e)
			for _, nb := range nbs {
				if !utils.Contains(nb, flat) && validHash3(a, nb) {
					flat = append(flat, nb)
					curLayer = append(curLayer, nb)
				}
			}
		}

		nValid := len(curLayer)
		if nValid > 0 {
			layers = append(layers, curLayer)
		} else {
			return layers
		}

		if nValid > len(l) {
			return getLayers(curLayer)
		} else {
			return layers
		}

	}

	getLayers(layers[0])

	// required for firestore...
	packOfTens, ilay := [][]string{{}}, int(0)
	for _, ll := range layers {
		for _, l := range ll {
			if len(packOfTens[ilay]) < 10 {
				packOfTens[ilay] = append(packOfTens[ilay], l)
			} else {
				packOfTens = append(packOfTens, []string{})
				ilay++
				packOfTens[ilay] = append(packOfTens[ilay], l)
			}
		}
	}

	return packOfTens

}

type area struct {
	Center latlon   `json:"center"`
	Perim  []latlon `json:"perim"`
}

type boostRequest2 struct {
	Token         string                 `json:"token"`
	DeviceID      string                 `json:"deviceID"`
	SenderID      string                 `json:"senderID"`
	ChangeAddress string                 `json:"changeAddr"`
	S1            string                 `json:"s1"`
	PricePerHead  int                    `json:"pph"`
	InputSats     int                    `json:"inputSats"`
	PartialTx     string                 `json:"tx"`
	Limit         int                    `json:"limit"`
	MaxAge        int                    `json:"maxAge"`
	MinAge        int                    `json:"minAge"`
	Genders       []string               `json:"genders"` // "male", "female", ""
	Areas         []area                 `json:"areas"`
	BoostMessage  map[string]interface{} `json:"boostMessage"`
	Media         map[string]string      `json:"boostMedia"`
	MediaPayload  string                 `json:"mediaPayload"`
}

func (br *boostRequest2) buildQuery(c *firestore.Client, layer []string, lim int) *firestore.Query {
	q := c.Collection("users").
		Where("age", "<=", br.MaxAge).
		Where("age", ">=", br.MinAge).
		Where("gender", "in", br.Genders).
		Where("geohash", "in", layer).
		Limit(lim)
	return &q
}

type user struct {
	Lat    float64 `firestore:"latitude"`
	Lon    float64 `firestore:"longitude"`
	Id     string  `firestore:"id"`
	Token  string  `firestore:"token"`
	Neuter string  `firestore:"neuter"`
}

func writeBoosts(ctx context.Context, users []*user, br *boostRequest2) ([]error, int) {
	const packSize int = 20000
	nUsers := len(users)
	nPacks := int(math.Ceil(float64(nUsers) / float64(packSize)))
	ch, errs := make(chan error, nPacks), make([]error, nPacks)
	prfx := satsPrefix(br.PricePerHead)
	var rawMedia []byte
	if len(br.MediaPayload) > 0 {
		rawMedia, _ = base64.StdEncoding.DecodeString(br.MediaPayload)
	}

	for i := 0; i < nPacks; i++ {
		go func(j int) {
			packStart := int64(j * packSize)
			var packEnd int
			if j == nPacks-1 {
				packEnd = nUsers
			} else {
				packEnd = (j + 1) * packSize
			}

			m := map[string]int{}
			r := func(m map[string]int, u *user) map[string]int {
				id, _ := utils.ParseSingleRoot(u.Id)
				m[id.Region]++
				return m

			}
			
			areaMap := utils.Reduce(users[packStart:packEnd], m, r)
			reg, ishrd := utils.MaxKey(areaMap), rand.Int()%server.N_SHARD

			shrd := server.Client.Shards[reg][ishrd]
			rbuf := utils.RandomBytes(16)
			unik := base58.Encode(rbuf)
			boostId := utils.ComposedId{Unik: unik, Region: reg, Shard: ishrd}
			boostIdStr := boostId.ToString()

			payload := make(map[string]interface{})
			utils.CopyMap(br.BoostMessage, payload)
			payload["id"] = boostIdStr
			payload["sats"] = br.PricePerHead
			payload["packStart"] = packStart
			payload["packEnd"] = packEnd

			err := shrd.RealtimeDB.NewRef("boosts/"+unik).Set(ctx, payload)
			if err != nil {
				ch <- err
				return
			}

			if len(rawMedia) > 0 {
				munik := base58.Encode(utils.RandomBytes(16))
				// munik := base32.StdEncoding.EncodeToString(utils.RandomBytes(16))
				cpMid := utils.ComposedId{Unik: munik, Region: reg, Shard: ishrd}
				midStr := cpMid.ToString() + "m"
				obj := shrd.TempBucket.Object(midStr)
				mtdt := make(map[string]string, len(br.Media))
				utils.CopyMap(br.Media, mtdt)
				mtdt["id"] = midStr
				wtr := obj.NewWriter(ctx)
				wtr.Metadata = mtdt
				wtr.Write(rawMedia)
				if err = wtr.Close(); err != nil {
					log.Printf("error writing boost rawMedia: %v\n", err)
					ch <- err
					return
				}
			}

			for _, usr := range users[j*packSize : packEnd] {
				cp, err := utils.ParseSingleRoot(usr.Id)
				if err != nil {
					msg := fmt.Sprintf("invalid user root=%v", usr.Id)
					utils.NonFatal(err, msg)
					continue
				}
				db := cp.ServerShard().RealtimeDB
				k := prfx + "%" + boostIdStr
				pth := "nodes/" + cp.Unik + "/queues/boost/" + k
				err = db.NewRef(pth).Set(ctx, "")

			}
			ch <- nil
		}(i)
	}

	for i := 0; i < nPacks; i++ {
		if err := <-ch; err != nil {
			errs = append(errs, err)
		}
	}
	return errs, nPacks
}

func satsPrefix(sats int) string {
	// let's set an upper limit of pph of 1bsv // which is way over anyone will pay
	// that sets 100 000 000 sats
	// const upsat uint64 = 100000000

	// max is 4bill for a single head, which is like 42 bsv
	const upper uint32 = math.MaxUint32
	dif := upper - uint32(sats)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, dif)
	prfx := base32.HexEncoding.EncodeToString(buf.Bytes())

	return prfx
}

func HandleBoostRequest(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	var br boostRequest2
	err := json.NewDecoder(r.Body).Decode(&br)
	utils.Fatal(err, "error decoding boostRequest")

	if br.PricePerHead > math.MaxUint32 {
		log.Fatalf("price per head exceeds maximum amount of 42 bsv: %v\n", br.PricePerHead)
	}

	txbuf, err := base64.StdEncoding.DecodeString(br.PartialTx)
	utils.Fatal(err, "error decoding base64 partial tx")

	s1, err := base64.StdEncoding.DecodeString(br.S1)
	utils.Fatal(err, "error decoding base64 s1")

	addr, err := base64.StdEncoding.DecodeString(br.ChangeAddress)
	utils.Fatal(err, "error decoding base64 change address")

	tx := bsv.TxFromRdr(bytes.NewReader(txbuf))
	log.Printf("tx pre boost\n%v", tx.Formatted())

	lim := br.Limit
	users := make([]*user, 0, lim)

	for _, a := range br.Areas {
		usrs, newlim := scanArea(ctx, &br, a, lim)
		users = append(users, usrs...)
		lim = newlim
		if lim == 0 {
			break
		}
	}

	nOuts := br.Limit - lim
	if nOuts == 0 {
		log.Fatalln("error, haven't found any people to boost")
	}

	rdyTx := bsv.BoostScript(tx, s1, nOuts, br.PricePerHead, br.InputSats, addr)
	rawTx := rdyTx.Raw()
	rawTxHex, txid := hex.EncodeToString(rawTx), bsv.Txid(rawTx)
	txidHex := hex.EncodeToString(txid)
	log.Printf("ready tx\n%v", rdyTx.Formatted())
	// rawTxHex := hex.EncodeToString(rdyTx.Raw())
	log.Printf("raw hex tx\n%v\n", rawTxHex)
	// txHexRdr := strings.NewReader(rawTxHex)
	txPayload := map[string]interface{}{"txhex": rawTxHex}
	// txPayload := map[string]interface{}{"rawTx": rawTxHex}
	// txPayload := map[string]interface{}{"raw": rawTxHex}
	rawPayload, err := json.Marshal(txPayload)
	utils.Fatal(err, "err marshalling txPayload")
	payloadRdr := bytes.NewReader(rawPayload)

	// const url string = "https://test-api.bitails.io/tx/broadcast"
	const url string = "https://api.whatsonchain.com/v1/bsv/test/tx/raw"
	// const url string = "https://api.taal.com/api/v1/broadcast"
	// req, err := http.NewRequest("POST", url, payloadRdr)
	// req.Header = map[string][]string{
	// 	"Content-Type": {"application/json"},
	// 	"Authorization": {"Bearer " + taal_api_key},
	// }
	// rsp, err := http.DefaultClient.Do(req)

	rsp, err := http.Post(url, "application/json", payloadRdr)
	utils.Fatal(err, "error posting tx to miners")
	if rsp.StatusCode != 200 {
		rbuf, err := io.ReadAll(rsp.Body)
		utils.Fatal(err, "error reading response buffer")
		log.Fatalf("error broadcasting tx: %s\n", string(rbuf))
	}

	var rjson map[string]interface{}
	err = json.NewDecoder(rsp.Body).Decode(&rjson)
	utils.Fatal(err, "error decoding response")
	// txid := rjson["txid"].(string)
	status, ok := rjson["status"].(int)
	if !ok {
		log.Fatalln("error finding response status")
	}

	if status != 200 {
		title, detail := rjson["title"], rjson["detail"]
		log.Fatalf("%s\n%d\n%s\n", title, status, detail)
	}

	errs, nPacks := writeBoosts(ctx, users, &br)
	if len(errs) == nPacks {
		log.Fatalf("every boost failed, err1=%v\n", errs[0])
	}

	// change index is -> nOuts + 1 - 1 -> nOuts
	pushPayload := txidHex + "@" + strconv.FormatInt(int64(nOuts), 10)
	err = messagerequests.PushData(ctx, br.SenderID, br.DeviceID, pushPayload)
	utils.Fatal(err, "error pushing data after boost request")

	header, body := "Completed Boost", fmt.Sprintf("Found %v targets", nOuts)
	err = messagerequests.PushNotification(ctx, br.Token, body, header, "", "")
	if err != nil {
		log.Printf("not fatal, could not push notification to receipient: %v\n", err)
	}
}

func scanArea(ctx context.Context, b *boostRequest2, a area, lim int) ([]*user, int) {
	layers := calcLayers2(a)
	fmt.Printf("layers: %v\n", layers)

	users := make([]*user, 0, lim)
	var curlim int = lim

	for _, layer := range layers {
		la, li := layer, curlim
		q := b.buildQuery(server.Client.Firestore, la, li)
		it := q.Documents(ctx)
		for {
			var usr user
			doc, err := it.Next()
			if err != nil {
				break
			}

			if err = doc.DataTo(&usr); err != nil {
				break
			}

			cl2 := closest2(a.Perim, latlon{Lat: usr.Lat, Lon: usr.Lon})
			usrDist := geoDist(latlon{Lat: usr.Lat, Lon: usr.Lon}, a.Center)

			valid := utils.Any(cl2, func(ll latlon) bool {
				return usrDist <= ll.RefDist
			})

			if valid {
				users = append(users, &usr)
				curlim--
				if curlim == 0 {
					break
				}
			}
		}

		if curlim == 0 {
			break
		}
	}
	fmt.Printf("we found %v users for the boost\n", len(users))
	return users, curlim
}
