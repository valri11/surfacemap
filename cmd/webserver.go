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
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"
	"strconv"
	"sync"
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
	"github.com/paulmach/orb/simplify"
	"github.com/spf13/cobra"
	"github.com/valri11/surfacemap/slippymath"

	"github.com/fogleman/colormap"
	"github.com/fogleman/contourmap"
	"github.com/fogleman/gg"

	lrucache "github.com/hashicorp/golang-lru"
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
}

const (
	tilesBucket = "elevation-tiles-prod"
	awsRegion   = "us-east-1"
)

const (
	CacheSize = 500
	TileSize  = 256

	MaxConcurrency = 8
)

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
	cfg       aws.Config
	s3Config  s3Config
	tileCache *lrucache.Cache
}

func NewTerra(cfg aws.Config, s3Config s3Config) (*terra, error) {
	tileCache, err := lrucache.New(CacheSize)
	if err != nil {
		return nil, err
	}
	t := terra{
		cfg:       cfg,
		s3Config:  s3Config,
		tileCache: tileCache,
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
	r.HandleFunc("/terra/{z}/{x}/{y}.img", t.tilesHandler)
	r.HandleFunc("/terrain/{z}/{x}/{y}.img", t.tilesTerrainHandler)
	r.HandleFunc("/contours/{z}/{x}/{y}.{format}", t.tilesContoursHandler)

	// Where ORIGIN_ALLOWED is like `scheme://dns[:port]`, or `*` (insecure)
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "content-type", "username", "password", "Referer"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	// start server listen with error handling
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

	//oName := fmt.Sprintf("v2/normal/%d/%d/%d.png", z, x, y)
	oName := fmt.Sprintf("v2/terrarium/%d/%d/%d.png", z, x, y)

	var buf *bytes.Buffer
	// cache lookup
	dt1 := time.Now()
	if h.tileCache.Contains(oName) {
		obj, ok := h.tileCache.Get(oName)
		if !ok {
			http.Error(w, "cache error", http.StatusBadRequest)
			return
		}

		cacheData, ok := obj.([]byte)
		if !ok {
			http.Error(w, "cache error", http.StatusBadRequest)
			return
		}
		buf = bytes.NewBuffer(cacheData)

		dt2 := time.Now()
		log.Printf("Cache hit: %s, read: %d in %v", oName, buf.Len(), dt2.Sub(dt1))
	} else {
		s3Client := s3.NewFromConfig(h.cfg)
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

		buf = new(bytes.Buffer)
		buf.ReadFrom(goo.Body)
		dRead := buf.Len()

		goo.Body.Close()
		// add cache entry
		cacheData := make([]byte, buf.Len())
		copy(cacheData, buf.Bytes())
		h.tileCache.Add(oName, cacheData)

		dt2 := time.Now()
		log.Printf("S3 GetObject: %s, read: %d in %v", oName, dRead, dt2.Sub(dt1))
	}
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

func (h *terra) tilesTerrainHandler(w http.ResponseWriter, r *http.Request) {
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

	//oName := fmt.Sprintf("v2/normal/%d/%d/%d.png", z, x, y)
	oName := fmt.Sprintf("v2/terrarium/%d/%d/%d.png", z, x, y)

	var buf *bytes.Buffer
	// cache lookup
	dt1 := time.Now()
	if h.tileCache.Contains(oName) {
		obj, ok := h.tileCache.Get(oName)
		if !ok {
			http.Error(w, "cache error", http.StatusBadRequest)
			return
		}

		cacheData, ok := obj.([]byte)
		if !ok {
			http.Error(w, "cache error", http.StatusBadRequest)
			return
		}
		buf = bytes.NewBuffer(cacheData)

		dt2 := time.Now()
		log.Printf("Cache hit: %s, read: %d in %v", oName, buf.Len(), dt2.Sub(dt1))
	} else {
		s3Client := s3.NewFromConfig(h.cfg)
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

		buf = new(bytes.Buffer)
		buf.ReadFrom(goo.Body)
		dRead := buf.Len()

		goo.Body.Close()
		// add cache entry
		cacheData := make([]byte, buf.Len())
		copy(cacheData, buf.Bytes())
		h.tileCache.Add(oName, cacheData)

		dt2 := time.Now()
		log.Printf("S3 GetObject: %s, read: %d in %v", oName, dRead, dt2.Sub(dt1))
	}
	w.Header().Set("Content-Type", "image/png")

	dt1 = time.Now()
	pixel_res, err := slippymath.TilePixelResolution(uint32(z), float64(x), float64(y))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h_factor := 1.0
	altitude := 45.0
	azimuth := 315.0
	imgOut, err := HillshadeImage(img, pixel_res, h_factor, altitude, azimuth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dt2 := time.Now()
	log.Printf("Hillshade completed in %v", dt2.Sub(dt1))

	buf = new(bytes.Buffer)
	err = png.Encode(buf, imgOut)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out := buf.Bytes()

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
	// request surrounding tiles

	steps := [9][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {0, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

	tiles := make([]*bytes.Buffer, 9)

	dRead := 0
	dt1 := time.Now()
	for idx, d := range steps {
		oName := fmt.Sprintf("v2/terrarium/%d/%d/%d.png", zoom, tile_X+d[0], tile_Y+d[1])

		// cache lookup
		if h.tileCache.Contains(oName) {
			dtRead1 := time.Now()
			obj, ok := h.tileCache.Get(oName)
			if !ok {
				http.Error(w, "cache error", http.StatusBadRequest)
				return
			}

			cacheData, ok := obj.([]byte)
			if !ok {
				http.Error(w, "cache error", http.StatusBadRequest)
				return
			}

			data := bytes.NewBuffer(cacheData)

			dRead += data.Len()
			tiles[idx] = data

			dtRead2 := time.Now()
			log.Printf("Cache hit: %s, read: %d in %v", oName, data.Len(), dtRead2.Sub(dtRead1))
		} else {
			dtRead1 := time.Now()

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

			data := new(bytes.Buffer)
			data.ReadFrom(goo.Body)

			dRead += data.Len()
			tiles[idx] = data

			// add cache entry
			cacheData := make([]byte, data.Len())
			copy(cacheData, data.Bytes())
			h.tileCache.Add(oName, cacheData)

			goo.Body.Close()

			dtRead2 := time.Now()
			log.Printf("S3 GetObject: %s, read: %d in %v", oName, data.Len(), dtRead2.Sub(dtRead1))
		}
	}
	dt2 := time.Now()

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
			ls := orb.LineString{}
			for _, point := range contour {

				lon, lat := slippymath.TileToLonLat(
					uint32(zoom+8),
					float64((tile_X-1)*TileSize)+point.X, float64((tile_Y-1)*TileSize)+point.Y)
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

		layers.Simplify(simplify.DouglasPeucker(1.0))

		// Depending on use-case remove empty geometry, those too small to be
		// represented in this tile space.
		// In this case lines shorter than 1, and areas smaller than 2.
		layers.RemoveEmpty(1.0, 2.0)

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

func transformToSunshade(data []byte) ([]byte, error) {
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

func rgbaToHeight(r uint32, g uint32, b uint32, a uint32) float64 {

	r &= 0xff
	g &= 0xff
	b &= 0xff

	// (red * 256 + green + blue / 256) - 32768
	h := float64(r*256 + g)
	h += float64(b) / 256
	h -= 32768
	return h
}

func HillshadeImage(img image.Image,
	pixel_res float64,
	h_factor float64,
	altitude float64,
	azimuth float64) (image.Image, error) {

	//h_factor := 1.0
	//altitude := 45.0
	//azimuth := 315.0

	zenith_deg := 90.0 - altitude
	zenith_rad := zenith_deg * math.Pi / 180.0

	cosZenithRad := math.Cos(zenith_rad)
	sinZenithRad := math.Sin(zenith_rad)

	azimuth_math := 360.0 - azimuth + 90.0
	azimuth_rad := azimuth_math * math.Pi / 180.0

	// a d g
	// b e h
	// c f i

	// dz/dx = ((c + 2f + i) - (a + 2d + g)) / (8 * pixel_res)
	// dz/dy = ((g + 2h + i) - (a + 2b + c))/ (8 * pixel_res)
	//
	// slope = atan(z_factor * sqrt((dz/dx)^2 + (dz/dy)^2))
	// aspect = atan2(dz/dy, -dz/dx)
	//
	// shaded relief = 255 * ((cos(90 - altitude) * cos(slope))
	// 		+ (sin(90 - altitude) * sin(slope) * cos(azimuth – aspect)))
	//
	// If [dz/dx] is non-zero:
	//   Aspect_rad = atan2 ([dz/dy], -[dz/dx])
	//     if Aspect_rad < 0 then
	//     	Aspect_rad = 2 * pi + Aspect_rad
	//   If [dz/dx] is zero:
	//     if [dz/dy] > 0 then
	//       Aspect_rad = pi / 2
	//     else if [dz/dy] < 0 then
	//       Aspect_rad = 2 * pi - pi / 2
	//     else
	//       Aspect_rad = Aspect_rad

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	getHeightAtPixel := func(img image.Image, x int, y int) float64 {
		dr, dg, db, da := img.At(x, y).RGBA()
		h := rgbaToHeight(dr, dg, db, da)
		return h
	}

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	imgOut := image.NewGray(image.Rectangle{upLeft, lowRight})

	var wg sync.WaitGroup
	sem := make(chan bool, MaxConcurrency)

	// a(-1,-1) d(0,-1) g(1,-1)
	// b(-1,0) e(0,0) h(1,0)
	// c(-1,1) f(0,1) i(1,1)
	for y := 1; y < height-1; y++ {

		wg.Add(1)
		sem <- true

		y := y

		go func() {
			defer func() {
				<-sem
				wg.Done()
			}()

			for x := 1; x < width-1; x++ {

				a := getHeightAtPixel(img, x-1, y-1)
				b := getHeightAtPixel(img, x-1, y)
				c := getHeightAtPixel(img, x-1, y+1)
				d := getHeightAtPixel(img, x, y-1)
				f := getHeightAtPixel(img, x, y+1)
				g := getHeightAtPixel(img, x+1, y-1)
				h := getHeightAtPixel(img, x+1, y)
				i := getHeightAtPixel(img, x+1, y+1)

				dz_dx := ((c + 2*f + i) - (a + 2*d + g)) / (8 * pixel_res)
				dz_dy := ((g + 2*h + i) - (a + 2*b + c)) / (8 * pixel_res)

				slope_rad := math.Atan(h_factor * math.Sqrt(dz_dx*dz_dx+dz_dy*dz_dy))

				var aspect_rad float64
				if dz_dx != 0.0 {
					aspect_rad = math.Atan2(dz_dy, -dz_dx)
					if aspect_rad < 0.0 {
						aspect_rad += 2 * math.Pi
					}
				} else {
					if dz_dy > 0 {
						aspect_rad = math.Pi / 2
					} else if dz_dy < 0.0 {
						aspect_rad = 2*math.Pi - math.Pi/2
					}
				}

				hillshade := math.Floor(255.0 * ((cosZenithRad * math.Cos(slope_rad)) +
					(sinZenithRad * math.Sin(slope_rad) * math.Cos(azimuth_rad-aspect_rad))))
				if hillshade < 0 {
					hillshade = 0
				}

				col := color.Gray{uint8(hillshade)}
				imgOut.Set(x, y, col)
			}
		}()
	}

	wg.Wait()

	// fixup artefacts along lines - copy shade from neighbour
	y := 0
	for x := 0; x < width; x++ {
		col := imgOut.At(x, y+1)
		imgOut.Set(x, y, col)
	}
	y = height - 1
	for x := 0; x < width; x++ {
		col := imgOut.At(x, y-1)
		imgOut.Set(x, y, col)
	}
	x := 0
	for y := 0; y < height; y++ {
		col := imgOut.At(x+1, y)
		imgOut.Set(x, y, col)
	}
	x = width - 1
	for y := 0; y < height; y++ {
		col := imgOut.At(x-1, y)
		imgOut.Set(x, y, col)
	}

	return imgOut, nil
}
