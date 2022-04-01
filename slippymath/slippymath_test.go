package slippymath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	epsilon = 0.0001
)

func Test_pixelResolution(t *testing.T) {

	testData := []struct {
		tile        Tile
		expectedRes float64
	}{
		{
			Tile{0, 0, 0},
			156543.0351,
		},
		{
			Tile{3, 7, 3},
			18150.2994,
		},
		{
			Tile{3, 7, 4},
			18150.2994,
		},
		{
			Tile{11, 1946, 1016},
			76.416,
		},
		{
			Tile{14, 11583, 6049},
			7.0402,
		},
		{
			Tile{17, 40971, 66614},
			1.1927,
		},
	}

	for _, tst := range testData {
		res, err := TilePixelResolution(tst.tile.Z, tst.tile.X, tst.tile.Y)
		require.NoError(t, err)
		lon, lat := TileCenterToLonLat(tst.tile.Z, tst.tile.X, tst.tile.Y)
		t.Logf("Z=%d, X=%f, Y=%f, lon=%f, lat=%f, res=%f",
			tst.tile.Z, tst.tile.X, tst.tile.Y,
			lon, lat, res)
		assert.InEpsilon(t, tst.expectedRes, res, 0.001)
	}
}

func Test_tileToLonLat(t *testing.T) {
	// tile, expected lon/lat

	testData := []struct {
		tile Tile
		geo  GeoCoord
	}{
		{
			Tile{14, 11583, 6049},
			GeoCoord{74.509277, 42.536892},
		},
		{
			Tile{14, 11584, 6049},
			GeoCoord{74.531250, 42.536892},
		},
		{
			Tile{14, 11582, 6049},
			GeoCoord{74.487305, 42.536892},
		},
	}

	for _, tst := range testData {
		lon, lat := TileToLonLat(tst.tile.Z, tst.tile.X, tst.tile.Y)
		t.Logf("lon=%f, lat=%f", lon, lat)
		assert.InEpsilon(t, tst.geo.Lon, lon, epsilon)
		assert.InEpsilon(t, tst.geo.Lat, lat, epsilon)
	}

}
