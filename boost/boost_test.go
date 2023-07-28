package boost

import (
	"testing"
)

func TestLayers(t *testing.T) {

	js := [][]float64{
		{
			48.839677,
			-67.539063,
			5.0,
		},
		{
			48.839677,
			-67.539063,
			20.0,
		},
		{
			48.839677,
			-67.539063,
			50.0,
		},
	}

	for _, j := range js {
		layers := calcLayers(j[0], j[1], j[2])
		t.Logf("layers: %v\nrad: %v\n", layers, j[2])
	}

}

func TestBoostRequest(t *testing.T) {
	brs := []*boostRequest{
		{
			MaxAge:  50,
			MinAge:  18,
			Genders: []string{"male"},
			Lat:     48.839677,
			Lon:     -67.539063,
			Rad:     5.0,
			Limit:   100,
		},
		{
			MaxAge:  50,
			MinAge:  21,
			Genders: []string{"male"},
			Lat:     48.839677,
			Lon:     -67.539063,
			Rad:     5.0,
			Limit:   100,
		},
		{
			MaxAge:  50,
			MinAge:  18,
			Genders: []string{"male"},
			Lat:     48.839677,
			Lon:     -67.539063,
			Rad:     5.0,
			Limit:   1,
		},
		{
			MaxAge:  50,
			MinAge:  18,
			Genders: []string{"female"},
			Lat:     48.839677,
			Lon:     -67.539063,
			Rad:     5.0,
			Limit:   1,
		},
	}

	for _, br := range brs {
		handleBoostRequest(br)
	}

}
