package cmd

import (
	"bytes"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valri11/surfacemap/slippymath"
)

func Benchmark_hillshade_1(b *testing.B) {

	inFile := "./test_data/terrarium_14_11583_6049.png"

	dat, err := ioutil.ReadFile(inFile)
	require.NoError(b, err)

	img, _, err := image.Decode(bytes.NewReader(dat))
	require.NoError(b, err)

	tile_X := 11583
	tile_Y := 6049
	zoom := 14

	pixel_res, err := slippymath.TilePixelResolution(uint32(zoom), float64(tile_X), float64(tile_Y))
	require.NoError(b, err)
	assert.InEpsilon(b, 7.040, pixel_res, 0.001)

	h_factor := 1.0
	altitude := 45.0
	azimuth := 315.0

	var imgOut image.Image
	for n := 0; n < b.N; n++ {
		imgOut, err = HillshadeImage(img, pixel_res, h_factor, altitude, azimuth)
		require.NoError(b, err)
	}

	f, _ := os.Create("image_out.png")
	png.Encode(f, imgOut)
}
