package cmd

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"math"
	"os"
	"testing"
	"time"

	lrucache "github.com/hashicorp/golang-lru"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valri11/surfacemap/slippymath"

	"github.com/fogleman/contourmap"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func Test_cache_1(t *testing.T) {
	tileCache, err := lrucache.New(5)
	assert.Nil(t, err)

	data := []byte{1, 2, 3, 45}

	key1 := "key1"
	tileCache.Add(key1, data)

	dat1, ok := tileCache.Get(key1)
	assert.True(t, ok)

	dat2, ok := dat1.([]byte)
	assert.True(t, ok)

	assert.Equal(t, len(data), len(dat2))
}

func Test_contour_closed_1(t *testing.T) {
	inFile := "./test_data/terrarium_14_11583_6049.png"

	tile_X := 11583
	tile_Y := 6049
	zoom := 14

	dRead := 0

	dt1 := time.Now()

	dat, err := ioutil.ReadFile(inFile)
	require.NoError(t, err)

	img, _, err := image.Decode(bytes.NewReader(dat))
	require.NoError(t, err)

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	data := make([]float64, width*height)
	idx := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dr, dg, db, da := img.At(x, y).RGBA()
			h := rgbaToHeight(dr, dg, db, da)
			data[idx] = h
			idx++
		}
	}

	m := contourmap.FromFloat64s(width, height, data)
	m = m.Closed()

	z0 := m.Min
	z1 := m.Max
	t.Logf("min: %f, max: %f\n", z0, z1)

	lvlInterval := float64(50)

	zLevel := math.Ceil(z0/lvlInterval) * lvlInterval

	fc := geojson.NewFeatureCollection()

	for zLevel < z1 {
		contours := m.Contours(zLevel)
		for _, contour := range contours {
			ls := orb.LineString{}
			for _, point := range contour {
				lon, lat := slippymath.TileToLonLat(
					uint32(zoom+8),
					float64(tile_X*256)+point.X, float64(tile_Y*256)+point.Y)
				pt := orb.Point{lon, lat}
				ls = append(ls, pt)
			}
			t.Logf("z=%f, line: %v\n", zLevel, ls)
			feat := geojson.NewFeature(ls)
			feat.Properties["elevation"] = zLevel
			fc.Append(feat)
		}
		zLevel += lvlInterval
	}

	rawJSON, _ := fc.MarshalJSON()
	fmt.Printf("%v\n", string(rawJSON))

	err = os.WriteFile("dat1_closed.geojson", rawJSON, 0644)
	require.NoError(t, err)

	dt2 := time.Now()

	t.Logf("read %d bytes in %v\n", dRead, dt2.Sub(dt1))
}

func Test_contour_open_1(t *testing.T) {
	inFile := "./test_data/terrarium_14_11583_6049.png"

	tile_X := 11583
	tile_Y := 6049
	zoom := 14

	dRead := 0

	dt1 := time.Now()

	dat, err := ioutil.ReadFile(inFile)
	require.NoError(t, err)

	img, _, err := image.Decode(bytes.NewReader(dat))
	require.NoError(t, err)

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	data := make([]float64, width*height)
	idx := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			dr, dg, db, da := img.At(x, y).RGBA()
			h := rgbaToHeight(dr, dg, db, da)
			data[idx] = h
			idx++
		}
	}

	m := contourmap.FromFloat64s(width, height, data)
	//m = m.Closed()

	z0 := m.Min
	z1 := m.Max
	t.Logf("min: %f, max: %f\n", z0, z1)

	lvlInterval := float64(50)

	zLevel := math.Ceil(z0/lvlInterval) * lvlInterval

	fc := geojson.NewFeatureCollection()

	for zLevel < z1 {
		contours := m.Contours(zLevel)
		for _, contour := range contours {
			ls := orb.LineString{}
			for _, point := range contour {
				lon, lat := slippymath.TileToLonLat(
					uint32(zoom+8),
					float64(tile_X*256)+point.X, float64(tile_Y*256)+point.Y)
				pt := orb.Point{lon, lat}
				ls = append(ls, pt)
			}
			t.Logf("z=%f, line: %v\n", zLevel, ls)
			feat := geojson.NewFeature(ls)
			feat.Properties["elevation"] = zLevel
			fc.Append(feat)
		}
		zLevel += lvlInterval
	}

	rawJSON, _ := fc.MarshalJSON()
	fmt.Printf("%v\n", string(rawJSON))

	err = os.WriteFile("dat1_open.geojson", rawJSON, 0644)
	require.NoError(t, err)

	dt2 := time.Now()

	t.Logf("read %d bytes in %v\n", dRead, dt2.Sub(dt1))
}

func Test_hillshade_1(t *testing.T) {

	inFile := "./test_data/terrarium_14_11583_6049.png"

	dat, err := ioutil.ReadFile(inFile)
	require.NoError(t, err)

	img, _, err := image.Decode(bytes.NewReader(dat))
	require.NoError(t, err)

	tile_X := 11583
	tile_Y := 6049
	zoom := 14

	pixel_res, err := slippymath.TilePixelResolution(uint32(zoom), float64(tile_X), float64(tile_Y))
	require.NoError(t, err)
	assert.InEpsilon(t, 7.040, pixel_res, 0.001)

	h_factor := 1.0
	altitude := 45.0
	azimuth := 315.0
	imgOut, err := HillshadeImage(img, pixel_res, h_factor, altitude, azimuth)
	require.NoError(t, err)

	f, _ := os.Create("image_out.png")
	png.Encode(f, imgOut)
}
