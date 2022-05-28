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
	"image/draw"
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

	"github.com/fogleman/contourmap"
	"github.com/lucasb-eyer/go-colorful"
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

	webserverCmd.Flags().BoolP("dev-mode", "", false, "development mode (http on loclahost)")
	webserverCmd.Flags().String("tls-cert", "", "TLS certificate file")
	webserverCmd.Flags().String("tls-cert-key", "", "TLS certificate key file")
}

const (
	tilesBucket = "elevation-tiles-prod"
	awsRegion   = "us-east-1"
)

const (
	CacheSize = 4 * 512
	TileSize  = 256

	epsilon = 0.00001

	MaxConcurrency = 8
)

type FeatureOutFormat string

const (
	FeatureOutGeoJSON FeatureOutFormat = "geojson"
	FeatureOutMVT     FeatureOutFormat = "mvt"
)

type s3Config struct {
	region string
	bucket string
}

type terra struct {
	cfg                aws.Config
	s3Config           s3Config
	s3Client           *s3.Client
	s3TileStore        *S3TileStore
	cacheTileStore     *CacheTileStore
	elevationTileStore *ElevationTileStore
	gradientMap        *gradientMap
}

func NewTerra(cfg aws.Config, s3Config s3Config) (*terra, error) {

	s3Client := s3.NewFromConfig(cfg)

	tileNameTempl := "v2/terrarium/%d/%d/%d.png"

	s3TileStore, err := NewS3TileStore(s3Client, s3Config.bucket, tileNameTempl)
	if err != nil {
		return nil, err
	}

	cacheTileStore, err := NewCacheTileStore(tileNameTempl, CacheSize)
	if err != nil {
		return nil, err
	}

	elevationTileStore, err := NewElevationTileStore(tileNameTempl, CacheSize)
	if err != nil {
		return nil, err
	}

	// < 0  40 120 160 #2878a0

	// 0    110 220 110 #6edc6e
	// 900  240 250 160 #f0faa0
	// 1300 230 220 170 #e6dcaa
	// 1900 220 220 220 #dcdcdc
	// 2500 250 250 250 #fafafa

	//	   0 102 153 153 #0669999
	//	   1  46 154  88 #2e9a58
	//	 600 251 255 128 #fbff80
	//	1200 224 108  31 #e06c1f
	//	2500 200  55  55 #c83737
	//	4000 215 244 244 #d7f4f4

	colorCard := []colorCard{
		{0.00, "#066999"},
		{0.01, "#2e9a58"},
		{900.0, "#fbff80"},
		{1300.0, "#e06c1f"},
		{1900.0, "#c83737"},
		{2500.0, "#d7f4f4"},
	}

	gm, err := NewGradientMap(colorCard, 0.1)
	if err != nil {
		return nil, err
	}

	t := terra{
		cfg:                cfg,
		s3Config:           s3Config,
		s3Client:           s3Client,
		s3TileStore:        s3TileStore,
		cacheTileStore:     cacheTileStore,
		elevationTileStore: elevationTileStore,
		gradientMap:        gm,
	}
	return &t, nil
}

func mainCmd(cmd *cobra.Command, args []string) {

	devMode, err := cmd.Flags().GetBool("dev-mode")
	if err != nil {
		panic(err)
	}

	tlsCertFile, err := cmd.Flags().GetString("tls-cert")
	if err != nil {
		panic(err)
	}

	tlsCertKeyFile, err := cmd.Flags().GetString("tls-cert-key")
	if err != nil {
		panic(err)
	}

	if !devMode {
		if tlsCertFile == "" || tlsCertKeyFile == "" {
			fmt.Println("must provide TLS key and certificate")
			return
		}
	}

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
	r.HandleFunc("/color-relief/{z}/{x}/{y}.img", t.colorReliefHandler)

	// Where ORIGIN_ALLOWED is like `scheme://dns[:port]`, or `*` (insecure)
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "content-type", "username", "password", "Referer"})
	originsOk := handlers.AllowedOrigins([]string{"*"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	// start server listen with error handling
	mux := handlers.CORS(originsOk, headersOk, methodsOk)(r)
	srv := &http.Server{
		Addr:         "0.0.0.0:8000",
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	if devMode {
		err = srv.ListenAndServe()
	} else {
		err = srv.ListenAndServeTLS(tlsCertFile, tlsCertKeyFile)
	}
	if err != nil {
		log.Fatal(err)
	}
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

	buf, err := h.getTile(ctx, z, x, y)
	if err != nil {
		log.Printf("req: ERR: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "image/png")

	out := buf.Bytes()

	w.Write(out)
}

func (h *terra) colorReliefHandler(w http.ResponseWriter, r *http.Request) {
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

	buf, err := h.getTile(ctx, z, x, y)
	if err != nil {
		log.Printf("req: ERR: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dt1 := time.Now()

	img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imgOut, err := ColorReliefImage(img, h.gradientMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dt2 := time.Now()
	log.Printf("ColorRelief completed in %v", dt2.Sub(dt1))

	buf = new(bytes.Buffer)
	err = png.Encode(buf, imgOut)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out := buf.Bytes()

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "max-age:28800, public")
	cacheSince := time.Now().Format(http.TimeFormat)
	cacheUntil := time.Now().Add(8 * time.Hour).Format(http.TimeFormat)
	w.Header().Set("Last-Modified", cacheSince)
	w.Header().Set("Expires", cacheUntil)

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

	buf, err := h.getTile(ctx, z, x, y)
	if err != nil {
		log.Printf("req: ERR: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	dt1 := time.Now()
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

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "max-age:28800, public")
	cacheSince := time.Now().Format(http.TimeFormat)
	cacheUntil := time.Now().Add(8 * time.Hour).Format(http.TimeFormat)
	w.Header().Set("Last-Modified", cacheSince)
	w.Header().Set("Expires", cacheUntil)

	w.Write(out)
}

func (h *terra) tilesContoursHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars, outFormat, interval, lvlInterval, zoom, tile_X, tile_Y, shouldReturn := h.getRequestContourParams(r, w)
	if shouldReturn {
		return
	}

	log.Printf("Contours params: z=%v, x=%v, y=%v, interval=%s\n",
		vars["z"], vars["x"], vars["y"], interval)

	oName := fmt.Sprintf("v2/terrarium/%d/%d/%d.png", zoom, tile_X, tile_Y)

	// request surrounding tiles

	steps := [9][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {0, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}

	dtStart := time.Now()

	dt1 := time.Now()

	data := make([]float64, 9*TileSize*TileSize)

	for idx := 0; idx < 9; idx++ {

		elevTile, err := h.getElevationTile(ctx, zoom, tile_X+steps[idx][0], tile_Y+steps[idx][1])
		if err != nil {
			log.Printf("req: %s, ERR: %v", oName, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		for y := 0; y < TileSize; y++ {
			for x := 0; x < TileSize; x++ {
				h := elevTile[y*TileSize+x]

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

	dt2 := time.Now()
	log.Printf("decoded images %v\n", dt2.Sub(dt1))

	const off_px = 3

	width := TileSize + 2*off_px
	height := TileSize + 2*off_px

	windowedData := make([]float64, width*height)

	startX := TileSize - off_px
	startY := TileSize - off_px

	w_idx := 0
	for dt_y := 0; dt_y < height; dt_y++ {
		for dt_x := 0; dt_x < width; dt_x++ {
			dt_idx := (startY+dt_y)*3*TileSize + dt_x + startX
			windowedData[w_idx] = data[dt_idx]
			w_idx++
		}
	}

	m := contourmap.FromFloat64s(width, height, windowedData)

	z0 := m.Min
	z1 := m.Max

	zLevel := math.Ceil(z0/lvlInterval) * lvlInterval

	fc := geojson.NewFeatureCollection()

	for zLevel <= z1 {
		contours := m.Contours(zLevel)
		for _, contour := range contours {
			ls := make(orb.LineString, len(contour))
			for idx, point := range contour {

				px := point.X - off_px
				py := point.Y - off_px

				lon, lat := slippymath.TileToLonLat(
					uint32(zoom+8),
					float64(tile_X*TileSize)+px, float64(tile_Y*TileSize)+py)
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

	var out []byte
	var err error

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

	dt2 = time.Now()
	log.Printf("Contour completed in %v\n", dt2.Sub(dtStart))

	w.Write(out)
}

func (h *terra) getTile(ctx context.Context, zoom int, tile_X int, tile_Y int) (*bytes.Buffer, error) {
	cacheData, err := h.cacheTileStore.GetTile(ctx, uint32(zoom), uint32(tile_X), uint32(tile_Y))

	var tile *bytes.Buffer
	oName := fmt.Sprintf("v2/terrarium/%d/%d/%d.png", zoom, tile_X, tile_Y)
	dt1 := time.Now()

	if err == nil {
		tile = bytes.NewBuffer(cacheData)

		dt2 := time.Now()
		log.Printf("Cache hit: %s, read: %d in %v", oName, tile.Len(), dt2.Sub(dt1))
	} else if err == ErrTileNotFound {

		s3Data, err := h.s3TileStore.GetTile(ctx, uint32(zoom), uint32(tile_X), uint32(tile_Y))
		if err != nil {
			return nil, err
		}

		tile = bytes.NewBuffer(s3Data)

		cacheData := make([]byte, tile.Len())
		copy(cacheData, tile.Bytes())
		h.cacheTileStore.Add(uint32(zoom), uint32(tile_X), uint32(tile_Y), cacheData)

		dt2 := time.Now()
		log.Printf("S3 GetObject: %s, read: %d in %v", oName, len(cacheData), dt2.Sub(dt1))
	} else {
		return nil, err
	}
	return tile, nil
}

func (h *terra) getElevationTile(ctx context.Context, zoom int, tile_X int, tile_Y int) ([]float64, error) {
	dt1 := time.Now()

	elevData, err := h.elevationTileStore.GetTile(ctx, uint32(zoom), uint32(tile_X), uint32(tile_Y))
	if err == nil {
		dt2 := time.Now()
		log.Printf("Cache hit: %d_%d_%d, read elevation data in %v", zoom, tile_X, tile_Y, dt2.Sub(dt1))

		return elevData, nil
	} else if err != ErrTileNotFound {
		return nil, err
	}

	dt1 = time.Now()

	tile, err := h.getTile(ctx, zoom, tile_X, tile_Y)
	if err != nil {
		return nil, err
	}

	data := make([]float64, TileSize*TileSize)
	img, _, err := image.Decode(bytes.NewReader(tile.Bytes()))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	idx := 0
	for y := 0; y < TileSize; y++ {
		for x := 0; x < TileSize; x++ {
			pix_idx := (y*TileSize + x) * 4
			pix := rgba.Pix[pix_idx : pix_idx+4]
			dr := uint32(pix[0])
			dg := uint32(pix[1])
			db := uint32(pix[2])
			da := uint32(pix[3])
			h := rgbaToHeight(dr, dg, db, da)

			data[idx] = h
			idx++
		}
	}

	h.elevationTileStore.Add(uint32(zoom), uint32(tile_X), uint32(tile_Y), data)

	dt2 := time.Now()
	log.Printf("Elevation tile: %d_%d_%d, decode elevation data in %v", zoom, tile_X, tile_Y, dt2.Sub(dt1))

	return data, nil
}

func (*terra) getRequestContourParams(r *http.Request, w http.ResponseWriter) (map[string]string, FeatureOutFormat, string, float64, int, int, int, bool) {
	vars := mux.Vars(r)

	outFormat := FeatureOutFormat(vars["format"])
	switch outFormat {
	case FeatureOutGeoJSON, FeatureOutMVT:
	default:
		err := errors.New("unsupported output format")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, "", "", 0, 0, 0, 0, true
	}

	interval := "100"
	intervals, ok := r.URL.Query()["interval"]
	if ok {
		interval = intervals[0]
	}
	iLvl, err := strconv.Atoi(interval)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, "", "", 0, 0, 0, 0, true
	}
	lvlInterval := float64(iLvl)

	zoom, err := strconv.Atoi(vars["z"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, "", "", 0, 0, 0, 0, true
	}
	tile_X, err := strconv.Atoi(vars["x"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, "", "", 0, 0, 0, 0, true
	}
	tile_Y, err := strconv.Atoi(vars["y"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil, "", "", 0, 0, 0, 0, true
	}
	return vars, outFormat, interval, lvlInterval, zoom, tile_X, tile_Y, false
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

type HeightColor struct {
	Col    colorful.Color
	Height float64
}

type GradientTable []HeightColor

type gradientMap struct {
	heighColors    map[float64]colorful.Color
	heighPrecision float64
	minColor       HeightColor
	maxColor       HeightColor
	gradients      GradientTable

	lock sync.RWMutex
}

type colorCard struct {
	height   float64
	colorHex string
}

func NewGradientMap(colorCard []colorCard, heighPrecision float64) (*gradientMap, error) {
	if colorCard == nil || len(colorCard) == 0 {
		return nil, errors.New("empty color cards")
	}

	gm := gradientMap{
		heighColors:    make(map[float64]colorful.Color),
		heighPrecision: heighPrecision,
		gradients:      make(GradientTable, 0),
	}

	gm.minColor = HeightColor{
		Height: colorCard[0].height,
		Col:    MustParseHex(colorCard[0].colorHex),
	}
	gm.maxColor = HeightColor{
		Height: colorCard[len(colorCard)-1].height,
		Col:    MustParseHex(colorCard[len(colorCard)-1].colorHex),
	}

	for idx := 1; idx < len(colorCard); idx++ {
		gm.gradients = append(gm.gradients,
			HeightColor{
				MustParseHex(colorCard[idx].colorHex),
				colorCard[idx].height,
			})
	}

	return &gm, nil
}

func (gm *gradientMap) HeightToColor(h float64) colorful.Color {

	if h <= gm.minColor.Height {
		return gm.minColor.Col
	}
	if h >= gm.maxColor.Height {
		return gm.maxColor.Col
	}

	hPos := math.Round(h/gm.heighPrecision) * gm.heighPrecision

	gm.lock.RLock()
	hCol, ok := gm.heighColors[hPos]
	gm.lock.RUnlock()
	if ok {
		return hCol
	}

	for i := 0; i < len(gm.gradients)-1; i++ {
		c1 := gm.gradients[i]
		c2 := gm.gradients[i+1]
		if c1.Height <= h && h <= c2.Height {
			// We are in between c1 and c2. Go blend them!
			hBlend := (h - c1.Height) / (c2.Height - c1.Height)
			hCol := c1.Col.BlendHcl(c2.Col, hBlend).Clamped()

			gm.lock.Lock()
			gm.heighColors[hPos] = hCol
			gm.lock.Unlock()

			return hCol
		}
	}

	return gm.maxColor.Col
}

func MustParseHex(s string) colorful.Color {
	c, err := colorful.Hex(s)
	if err != nil {
		panic("MustParseHex: " + err.Error())
	}
	return c
}
