/*
Copyright © 2022 Val Gridnev

*/
package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/mvt"
	"github.com/paulmach/orb/geojson"
	"github.com/paulmach/orb/maptile"
	"github.com/spf13/cobra"

	"github.com/fogleman/colormap"
	"github.com/fogleman/contourmap"
	"github.com/fogleman/gg"
)

const (
	N          = 12
	Scale      = 1
	Background = "77C4D3"
	Palette    = "70a80075ab007bb00080b30087b8008ebd0093bf009ac400a1c900a7cc00aed100b6d600bcd900c4de00cce300d2e600dbeb00e1ed00eaf200f3f700fafa00ffff05ffff12ffff1cffff29ffff36ffff42ffff4fffff5cffff66ffff73ffff80ffff8cffff99ffffa3ffffb0ffffbdffffc9ffffd6ffffe3ffffedfffffafcfcfcf7f7f7f5f5f5f0f0f0edededebebebe6e6e6e3e3e3dedededbdbdbd6d6d6d4d4d4cfcfcfccccccc7c7c7c4c4c4c2c2c2bdbdbdbababab5b5b5b3b3b3b3b3b3"
)

// webserverCmd represents the webserver command
var webserverCmd = &cobra.Command{
	Use:   "webserver",
	Short: "Contour vector tile server",
	Long:  `Contour vector tile server`,
	Run:   mainCmd,
}

func init() {
	rootCmd.AddCommand(webserverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// webserverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// webserverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

const (
	tilesBucket = "elevation-tiles-prod"
	awsRegion   = "us-east-1"
)

const TileSize = 256

type FeatureOutFormat string

const (
	FeatureOutGeoJSON FeatureOutFormat = "geojson"
	FeatureOutMVT                      = "mvt"
)

type s3Config struct {
	region string
	bucket string
}

type terra struct {
	cfg      aws.Config
	s3Config s3Config
}

func NewTerra(cfg aws.Config, s3Config s3Config) (*terra, error) {
	t := terra{
		cfg:      cfg,
		s3Config: s3Config,
	}
	return &t, nil
}

func mainCmd(cmd *cobra.Command, args []string) {
	fmt.Println("webserver called")

	ctx := context.Background()
	awsCfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsRegion))
	if err != nil {
		panic(err)
	}

	t, err := NewTerra(awsCfg, s3Config{region: awsRegion, bucket: tilesBucket})
	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	//r.HandleFunc("/", HomeHandler)
	r.HandleFunc("/terra/{z}/{x}/{y}.img", t.tilesHandler)
	r.HandleFunc("/contours/{z}/{x}/{y}.{format}", t.tilesContoursHandler)
	//http.Handle("/", r)

	// Where ORIGIN_ALLOWED is like `scheme://dns[:port]`, or `*` (insecure)
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "content-type", "username", "password", "Referer"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	// start server listen
	// with error handling
	log.Fatal(http.ListenAndServe("0.0.0.0:8000", handlers.CORS(originsOk, headersOk, methodsOk)(r)))
}

func (h *terra) tilesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)

	log.Printf("Tiles params: z=%v, x=%v, y=%v\n", vars["z"], vars["x"], vars["y"])

	z, err := strconv.Atoi(vars["z"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	x, err := strconv.Atoi(vars["x"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	y, err := strconv.Atoi(vars["y"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	oName := fmt.Sprintf("v2/normal/%d/%d/%d.png", z, x, y)

	s3Client := s3.NewFromConfig(h.cfg)
	goi := &s3.GetObjectInput{
		Bucket: aws.String(h.s3Config.bucket),
		Key:    aws.String(oName),
	}

	dt1 := time.Now()
	goo, err := s3Client.GetObject(ctx, goi)
	if err != nil {
		log.Printf("req: %s, ERR: %v", oName, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(goo.Body)
	dRead := buf.Len()

	goo.Body.Close()
	dt2 := time.Now()

	log.Printf("read %d bytes in %v\n", dRead, dt2.Sub(dt1))

	w.Header().Set("Content-Type", "image/png")

	out := buf.Bytes()

	//	out, err := transformToColormap(buf.Bytes())
	//	if err != nil {
	//		log.Printf("req: %s, ERR: %v", oName, err)
	//		http.Error(w, err.Error(), http.StatusBadRequest)
	//		return
	//	}

	w.Write(out)
}

func (h *terra) tilesContoursHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)

	outFormat := FeatureOutFormat(vars["format"])
	switch outFormat {
	case FeatureOutGeoJSON, FeatureOutMVT:
	default:
		err := errors.New("unsupported output format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	interval := "100"
	intervals, ok := r.URL.Query()["interval"]
	if ok {
		interval = intervals[0]
	}
	iLvl, err := strconv.Atoi(interval)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lvlInterval := float64(iLvl)

	log.Printf("Contours params: z=%v, x=%v, y=%v, interval=%s\n",
		vars["z"], vars["x"], vars["y"], interval)

	zoom, err := strconv.Atoi(vars["z"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tile_X, err := strconv.Atoi(vars["x"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tile_Y, err := strconv.Atoi(vars["y"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s3Client := s3.NewFromConfig(h.cfg)
	// request surround tiles

	steps := [9][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {0, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

	tiles := make([]*bytes.Buffer, 9)

	dRead := 0
	dt1 := time.Now()
	for idx, d := range steps {
		oName := fmt.Sprintf("v2/terrarium/%d/%d/%d.png", zoom, tile_X+d[0], tile_Y+d[1])

		//		if !(d[0] == 0 && d[1] == 0) {
		//			continue
		//		}

		goi := &s3.GetObjectInput{
			Bucket: aws.String(h.s3Config.bucket),
			Key:    aws.String(oName),
		}

		goo, err := s3Client.GetObject(ctx, goi)
		if err != nil {
			log.Printf("req: %s, ERR: %v", oName, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(goo.Body)

		dRead += buf.Len()

		tiles[idx] = buf

		goo.Body.Close()
	}
	dt2 := time.Now()

	//	oName := fmt.Sprintf("v2/terrarium/%d/%d/%d.png", zoom, tile_X, tile_Y)
	//
	//	s3Client := s3.NewFromConfig(h.cfg)
	//	goi := &s3.GetObjectInput{
	//		Bucket: aws.String(h.s3Config.bucket),
	//		Key:    aws.String(oName),
	//	}
	//
	//	dt1 := time.Now()
	//
	//	goo, err := s3Client.GetObject(ctx, goi)
	//	if err != nil {
	//		log.Printf("req: %s, ERR: %v", oName, err)
	//		http.Error(w, err.Error(), http.StatusBadRequest)
	//		return
	//	}
	//
	//	buf := new(bytes.Buffer)
	//	buf.ReadFrom(goo.Body)
	//	dRead := buf.Len()
	//
	//	goo.Body.Close()
	//	dt2 := time.Now()

	log.Printf("read %d bytes in %v\n", dRead, dt2.Sub(dt1))

	data := make([]float64, 9*TileSize*TileSize)

	for idx := 0; idx < 9; idx++ {
		buf := tiles[idx]
		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		if err != nil {
			log.Printf("req: ERR: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for y := 0; y < TileSize; y++ {
			for x := 0; x < TileSize; x++ {
				dr, dg, db, da := img.At(x, y).RGBA()
				dr &= 0xff
				dg &= 0xff
				db &= 0xff
				da &= 0xff
				h := rgbaToHeight(uint16(dr), uint16(dg), uint16(db), uint16(da))

				idxH := 0
				if idx == 0 {
					idxH = x + y*3*TileSize
					data[idxH] = h
				}
				if idx == 1 {
					idxH = x + TileSize + y*3*TileSize
					data[idxH] = h
				}
				if idx == 2 {
					idxH = x + 2*TileSize + y*3*TileSize
					data[idxH] = h
				}
				if idx == 3 {
					idxH = x + (y+TileSize)*3*TileSize
					data[idxH] = h
				}
				if idx == 4 {
					idxH = (x + TileSize) + (y+TileSize)*3*TileSize
					data[idxH] = h
				}
				if idx == 5 {
					idxH = (x + 2*TileSize) + (y+TileSize)*3*TileSize
					data[idxH] = h
				}
				if idx == 6 {
					idxH = (x) + (y+2*TileSize)*3*TileSize
					data[idxH] = h
				}
				if idx == 7 {
					idxH = (x + TileSize) + (y+2*TileSize)*3*TileSize
					data[idxH] = h
				}
				if idx == 8 {
					idxH = (x + 2*TileSize) + (y+2*TileSize)*3*TileSize
					data[idxH] = h
				}

				//data[idxH] = h
			}
		}
	}

	//	buf := tiles[4]
	//	img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
	//	if err != nil {
	//		log.Printf("req: ERR: %v", err)
	//		http.Error(w, err.Error(), http.StatusBadRequest)
	//		return
	//	}
	//
	//	bounds := img.Bounds()
	//	width, height := bounds.Max.X, bounds.Max.Y
	//
	//	//data := make([]float64, width*height)
	//	idx := 0
	//	for y := 0; y < height; y++ {
	//		for x := 0; x < width; x++ {
	//			dr, dg, db, da := img.At(x, y).RGBA()
	//			dr &= 0xff
	//			dg &= 0xff
	//			db &= 0xff
	//			da &= 0xff
	//			h := rgbaToHeight(uint16(dr), uint16(dg), uint16(db), uint16(da))
	//			data[idx] = h
	//			idx++
	//		}
	//	}

	//m := contourmap.FromFloat64s(width, height, data)
	m := contourmap.FromFloat64s(3*TileSize, 3*TileSize, data)
	//m = m.Closed()

	z0 := m.Min
	z1 := m.Max

	zLevel := math.Ceil(z0/lvlInterval) * lvlInterval

	fc := geojson.NewFeatureCollection()

	for zLevel <= z1 {
		contours := m.Contours(zLevel)
		for _, contour := range contours {
			ls := orb.LineString{}
			for _, point := range contour {

				//				if point.X < float64(TileSize) || point.X > float64(2*TileSize) {
				//					continue
				//				}
				//				if point.Y < float64(TileSize) || point.Y > float64(2*TileSize) {
				//					continue
				//				}
				//
				//				point.X -= float64(TileSize)
				//				point.Y -= float64(TileSize)

				//lon, lat := toGeo(float64(tile_X*TileSize)+point.X, float64(tile_Y*TileSize)+point.Y, uint32(zoom+8))
				lon, lat := toGeo(float64((tile_X-1)*TileSize)+point.X, float64((tile_Y-1)*TileSize)+point.Y, uint32(zoom+8))
				pt := orb.Point{lon, lat}
				ls = append(ls, pt)
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

	var out []byte

	if outFormat == FeatureOutGeoJSON {
		out, err = fc.MarshalJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
	} else if outFormat == FeatureOutMVT {

		colMvt := make(map[string]*geojson.FeatureCollection)
		colMvt["contours"] = fc

		// Convert to a layers object and project to tile coordinates.
		layers := mvt.NewLayers(colMvt)
		layers.ProjectToTile(maptile.New(uint32(tile_X), uint32(tile_Y), maptile.Zoom(zoom)))

		// encoding using the Mapbox Vector Tile protobuf encoding.
		out, err = mvt.Marshal(layers) // this data is NOT gzipped.
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/vnd.mapbox-vector-tile")
	}

	w.Write(out)
}

func transformToColormap(data []byte) ([]byte, error) {
	im, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	m := contourmap.FromImage(im).Closed()
	z0 := m.Min
	z1 := m.Max

	imw := int(float64(m.W) * Scale)
	imh := int(float64(m.H) * Scale)

	dc := gg.NewContext(imw, imh)
	dc.SetRGB(1, 1, 1)
	dc.SetColor(colormap.ParseColor(Background))
	dc.Clear()
	dc.Scale(Scale, Scale)

	pal := colormap.New(colormap.ParseColors(Palette))
	for i := 0; i < N; i++ {
		t := float64(i) / (N - 1)
		z := z0 + (z1-z0)*t
		contours := m.Contours(z + 1e-9)
		for _, c := range contours {
			dc.NewSubPath()
			for _, p := range c {
				dc.LineTo(p.X, p.Y)
			}
		}
		dc.SetColor(pal.At(t))
		dc.FillPreserve()
		dc.SetRGB(0, 0, 0)
		dc.SetLineWidth(1)
		dc.Stroke()
	}

	out := new(bytes.Buffer)
	png.Encode(out, dc.Image())

	return out.Bytes(), nil
}

func rgbaToHeight(r uint16, g uint16, b uint16, a uint16) float64 {
	// (red * 256 + green + blue / 256) - 32768
	h := float64(r*256 + g)
	h += float64(b) / 256
	h -= 32768
	return h
}

func toGeo(x, y float64, level uint32) (lng, lat float64) {
	maxtiles := float64(uint64(1 << level))

	lng = 360.0 * (x/maxtiles - 0.5)
	lat = 2.0*math.Atan(math.Exp(math.Pi-(2*math.Pi)*(y/maxtiles)))*(180.0/math.Pi) - 90.0

	return lng, lat
}
