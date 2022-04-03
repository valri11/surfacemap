package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"testing"
	"time"

	"github.com/fogleman/contourmap"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valri11/surfacemap/slippymath"
)

type TileProvider struct {
	tiles map[string][]byte
}

func NewTileProvider() *TileProvider {
	tp := TileProvider{
		tiles: make(map[string][]byte),
	}
	return &tp
}

func (tp *TileProvider) GetTile(z uint32, x uint32, y uint32) ([]byte, error) {

	key := fmt.Sprintf("%d_%d_%d", z, x, y)

	buf, ok := tp.tiles[key]
	if !ok {
		return nil, errors.New("Not found")
	}

	return buf, nil
}

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

func Benchmark_contour_1(b *testing.B) {

	inFiles := []string{
		"./test_data/terrarium_14_11582_6048.png",
		"./test_data/terrarium_14_11583_6048.png",
		"./test_data/terrarium_14_11584_6048.png",
		"./test_data/terrarium_14_11582_6049.png",
		"./test_data/terrarium_14_11583_6049.png",
		"./test_data/terrarium_14_11584_6049.png",
		"./test_data/terrarium_14_11582_6050.png",
		"./test_data/terrarium_14_11583_6050.png",
		"./test_data/terrarium_14_11584_6050.png",
	}

	tiles := make([]*bytes.Buffer, 9)

	for idx, inFile := range inFiles {
		dat, err := ioutil.ReadFile(inFile)
		require.NoError(b, err)

		data := bytes.NewBuffer(dat)

		tiles[idx] = data
	}

	//tile_X := 11583
	//tile_Y := 6049
	//zoom := 14

	for n := 0; n < b.N; n++ {

		data := make([]float64, 9*TileSize*TileSize)

		for idx := 0; idx < 9; idx++ {
			buf := tiles[idx]

			img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
			require.NoError(b, err)

			bounds := img.Bounds()
			rgba := image.NewRGBA(bounds)
			draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

			for y := 0; y < TileSize; y++ {
				for x := 0; x < TileSize; x++ {
					//dr, dg, db, da := img.At(x, y).RGBA()
					pix_idx := (y*TileSize + x) * 4
					pix := rgba.Pix[pix_idx : pix_idx+4]
					dr := uint32(pix[0])
					dg := uint32(pix[1])
					db := uint32(pix[2])
					da := uint32(pix[3])
					h := rgbaToHeight(dr, dg, db, da)

					idxH := 0
					if idx == 0 {
						idxH = x + y*3*TileSize
					}
					if idx == 1 {
						idxH = x + TileSize + y*3*TileSize
					}
					if idx == 2 {
						idxH = x + 2*TileSize + y*3*TileSize
					}
					if idx == 3 {
						idxH = x + (y+TileSize)*3*TileSize
					}
					if idx == 4 {
						idxH = (x + TileSize) + (y+TileSize)*3*TileSize
					}
					if idx == 5 {
						idxH = (x + 2*TileSize) + (y+TileSize)*3*TileSize
					}
					if idx == 6 {
						idxH = (x) + (y+2*TileSize)*3*TileSize
					}
					if idx == 7 {
						idxH = (x + TileSize) + (y+2*TileSize)*3*TileSize
					}
					if idx == 8 {
						idxH = (x + 2*TileSize) + (y+2*TileSize)*3*TileSize
					}

					data[idxH] = h
				}
			}

			//m := contourmap.FromFloat64s(3*TileSize, 3*TileSize, data)
			//contourmap.FromFloat64s(3*TileSize, 3*TileSize, data)

			//z0 := m.Min
			//z1 := m.Max

		}
	}
}

func Test_contour_2(t *testing.T) {

	inFiles := []struct {
		key      string
		dataFile string
	}{
		{"14_11582_6048", "./test_data/terrarium_14_11582_6048.png"},
		{"14_11583_6048", "./test_data/terrarium_14_11583_6048.png"},
		{"14_11584_6048", "./test_data/terrarium_14_11584_6048.png"},
		{"14_11582_6049", "./test_data/terrarium_14_11582_6049.png"},
		{"14_11583_6049", "./test_data/terrarium_14_11583_6049.png"},
		{"14_11584_6049", "./test_data/terrarium_14_11584_6049.png"},
		{"14_11582_6050", "./test_data/terrarium_14_11582_6050.png"},
		{"14_11583_6050", "./test_data/terrarium_14_11583_6050.png"},
		{"14_11584_6050", "./test_data/terrarium_14_11584_6050.png"},
	}

	tileProvider := NewTileProvider()

	for _, inDt := range inFiles {
		dat, err := ioutil.ReadFile(inDt.dataFile)
		require.NoError(t, err)

		tileProvider.tiles[inDt.key] = dat
	}

	tile_X := uint32(11583)
	tile_Y := uint32(6049)
	zoom := uint32(14)

	// increase tile by 1 pixel line for earch side
	// width = width + 2
	// height = height +2

	bufWidth := TileSize + 2
	bufHeight := TileSize + 2

	tileData, err := tileProvider.GetTile(zoom, tile_X, tile_Y)
	require.NoError(t, err)

	img, _, err := image.Decode(bytes.NewReader(tileData))
	require.NoError(t, err)

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	data := make([]float64, bufWidth*bufHeight)

	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {

			pix_idx := (y*TileSize + x) * 4
			pix := rgba.Pix[pix_idx : pix_idx+4]
			dr := uint32(pix[0])
			dg := uint32(pix[1])
			db := uint32(pix[2])
			da := uint32(pix[3])
			h := rgbaToHeight(dr, dg, db, da)

			dt_idx := (y+1)*bufHeight + (x + 1)
			data[dt_idx] = h
		}
	}

	yb := 0
	for xb := 1; xb < bufWidth-1; xb++ {
		dt_src_idx := (yb+1)*bufHeight + xb
		h := data[dt_src_idx]
		dt_dst_idx := yb*bufHeight + xb
		data[dt_dst_idx] = h
	}

	yb = bufHeight - 1
	for xb := 1; xb < bufWidth-1; xb++ {
		dt_src_idx := (yb-1)*bufHeight + xb
		h := data[dt_src_idx]
		dt_dst_idx := yb*bufHeight + xb
		data[dt_dst_idx] = h
	}

	xb := 0
	for yb := 0; yb < bufHeight; yb++ {
		dt_src_idx := yb*bufHeight + xb + 1
		h := data[dt_src_idx]
		dt_dst_idx := yb*bufHeight + xb
		data[dt_dst_idx] = h
	}

	xb = bufWidth - 1
	for yb := 0; yb < bufHeight; yb++ {
		dt_src_idx := yb*bufHeight + xb - 1
		h := data[dt_src_idx]
		dt_dst_idx := yb*bufHeight + xb
		data[dt_dst_idx] = h
	}

	m := contourmap.FromFloat64s(bufWidth, bufHeight, data)

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
					float64(tile_X*256)+point.X-1, float64(tile_Y*256)+point.Y-1)
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

	outFileName := fmt.Sprintf("Test_contour_2_%d_%d_%d.geojson", zoom, tile_X, tile_Y)
	err = os.WriteFile(outFileName, rawJSON, 0644)
	require.NoError(t, err)
}

func Test_contour_3(t *testing.T) {

	inFiles := []struct {
		key      string
		dataFile string
	}{
		{"14_11582_6048", "./test_data/terrarium_14_11582_6048.png"},
		{"14_11583_6048", "./test_data/terrarium_14_11583_6048.png"},
		{"14_11584_6048", "./test_data/terrarium_14_11584_6048.png"},
		{"14_11582_6049", "./test_data/terrarium_14_11582_6049.png"},
		{"14_11583_6049", "./test_data/terrarium_14_11583_6049.png"},
		{"14_11584_6049", "./test_data/terrarium_14_11584_6049.png"},
		{"14_11582_6050", "./test_data/terrarium_14_11582_6050.png"},
		{"14_11583_6050", "./test_data/terrarium_14_11583_6050.png"},
		{"14_11584_6050", "./test_data/terrarium_14_11584_6050.png"},
	}

	tileProvider := NewTileProvider()

	for _, inDt := range inFiles {
		dat, err := ioutil.ReadFile(inDt.dataFile)
		require.NoError(t, err)

		tileProvider.tiles[inDt.key] = dat
	}

	tile_X := 11583
	tile_Y := 6049
	zoom := uint32(14)

	lvlInterval := float64(50.0)

	dt1 := time.Now()

	data := make([]float64, 9*TileSize*TileSize)

	steps := [9][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {0, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

	for idx := 0; idx < 9; idx++ {

		d := steps[idx]

		tX := tile_X + d[0]
		tY := tile_Y + d[1]

		tileData, err := tileProvider.GetTile(zoom, uint32(tX), uint32(tY))
		require.NoError(t, err)

		img, _, err := image.Decode(bytes.NewReader(tileData))
		require.NoError(t, err)

		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

		for y := 0; y < TileSize; y++ {
			for x := 0; x < TileSize; x++ {
				//dr, dg, db, da := img.At(x, y).RGBA()
				pix_idx := (y*TileSize + x) * 4
				pix := rgba.Pix[pix_idx : pix_idx+4]
				dr := uint32(pix[0])
				dg := uint32(pix[1])
				db := uint32(pix[2])
				da := uint32(pix[3])
				h := rgbaToHeight(dr, dg, db, da)

				idxH := 0
				if idx == 0 {
					idxH = x + y*3*TileSize
				}
				if idx == 1 {
					idxH = x + TileSize + y*3*TileSize
				}
				if idx == 2 {
					idxH = x + 2*TileSize + y*3*TileSize
				}
				if idx == 3 {
					idxH = x + (y+TileSize)*3*TileSize
				}
				if idx == 4 {
					idxH = (x + TileSize) + (y+TileSize)*3*TileSize
				}
				if idx == 5 {
					idxH = (x + 2*TileSize) + (y+TileSize)*3*TileSize
				}
				if idx == 6 {
					idxH = (x) + (y+2*TileSize)*3*TileSize
				}
				if idx == 7 {
					idxH = (x + TileSize) + (y+2*TileSize)*3*TileSize
				}
				if idx == 8 {
					idxH = (x + 2*TileSize) + (y+2*TileSize)*3*TileSize
				}

				data[idxH] = h
			}
		}
	}

	m := contourmap.FromFloat64s(3*TileSize, 3*TileSize, data)

	z0 := m.Min
	z1 := m.Max

	zLevel := math.Ceil(z0/lvlInterval) * lvlInterval

	fc := geojson.NewFeatureCollection()

	for zLevel <= z1 {
		contours := m.Contours(zLevel)
		for _, contour := range contours {
			ls := make(orb.LineString, len(contour))
			for idx, point := range contour {

				lon, lat := slippymath.TileToLonLat(
					uint32(zoom+8),
					float64((tile_X-1)*TileSize)+point.X, float64((tile_Y-1)*TileSize)+point.Y)
				pt := orb.Point{lon, lat}
				ls[idx] = pt
			}
			if len(ls) == 0 {
				continue
			}
			feat := geojson.NewFeature(ls)
			feat.Properties["elevation"] = zLevel
			fc.Append(feat)
		}
		zLevel += lvlInterval
	}

	out, err := fc.MarshalJSON()
	require.NoError(t, err)

	dt2 := time.Now()
	log.Printf("Contour completed in %v\n", dt2.Sub(dt1))

	outFileName := fmt.Sprintf("Test_contour_3_%d_%d_%d.geojson", zoom, tile_X, tile_Y)
	err = os.WriteFile(outFileName, out, 0644)
	require.NoError(t, err)
}
