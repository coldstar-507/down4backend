package boost

import (
	"context"
	"fmt"
	"math"

	"cloud.google.com/go/firestore"
	"github.com/coldstar-507/down4backend/server"
	"github.com/coldstar-507/down4backend/utils"
	"github.com/mmcloughlin/geohash"
)

const precision = 4

func init() {
	ctx := context.Background()
	server.ServerInit(ctx)
}

func geoDist(lat1, lon1, lat2, lon2 float64) float64 {
	R := 6371.0                  // Radius of the earth in km
	dLat := deg2rad(lat2 - lat1) // deg2rad below
	dLon := deg2rad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(deg2rad(lat1))*math.Cos(deg2rad(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	d := R * c // Distance in km
	return d
}

func deg2rad(deg float64) float64 {
	return deg * (math.Pi / 180)
}

func validHash(lat, lon, rad float64, hash string) bool {
	box := geohash.BoundingBox(hash)
	dists := []float64{
		geoDist(lat, lon, box.MaxLat, box.MaxLng),
		geoDist(lat, lon, box.MaxLat, box.MinLng),
		geoDist(lat, lon, box.MinLat, box.MaxLng),
		geoDist(lat, lon, box.MinLat, box.MinLng),
	}
	for _, d := range dists {
		if d < rad {
			return true
		}
	}
	return false
}

func calcLayers(lat, lon, rad float64) [][]string {
	centerHash := geohash.EncodeWithPrecision(lat, lon, precision)
	layers := [][]string{{centerHash}}
	flat := []string{centerHash}

	var getLayers func([]string) [][]string
	getLayers = func(l []string) [][]string {
		curLayer := make([]string, 0)
		for _, e := range l {
			nbs := geohash.Neighbors(e)
			for _, nb := range nbs {
				if !utils.Contains[string](nb, flat) && validHash(lat, lon, rad, nb) {
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
	return getLayers(layers[0])
}

type boostRequest struct {
	Limit   int      `json:"limit"`
	MaxAge  int      `json:"maxAge"`
	MinAge  int      `json:"minAge"`
	Genders []string `json:"genders"` // "male", "female", ""
	Lat     float64  `json:"lat"`
	Lon     float64  `json:"lon"`
	Rad     float64  `json:"rad"`
}

func (br *boostRequest) buildQuery(c *firestore.Client, layer []string) *firestore.Query {
	q := c.Collection("users").
		Where("age", "<=", br.MaxAge).
		Where("age", ">=", br.MinAge).
		Where("gender", "in", br.Genders).
		Where("geohash", "in", layer)
	return &q
}

type user struct {
	Lat    float64 `firestore:"latitude"`
	Lon    float64 `firestore:"longitude"`
	Token  string  `firestore:"token"`
	Neuter string  `firestore:"neuter"`
}

func handleBoostRequest(br *boostRequest) {
	ctx := context.Background()

	layers := calcLayers(br.Lat, br.Lon, br.Rad)
	fmt.Printf("layers: %v\n", layers)

	users := make([]*user, 0, br.Limit)

	for _, l := range layers {
		q := br.buildQuery(server.Client.Firestore, l)
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

			if geoDist(usr.Lat, usr.Lon, br.Lat, br.Lon) < br.Rad {
				users = append(users, &usr)
				if len(users) >= br.Limit {
					break
				}
			}
		}

		if len(users) >= br.Limit {
			break
		}
	}

	fmt.Printf("we found %v users for the boost\n", len(users))
}
