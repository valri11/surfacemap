package slippymath

import "math"

const (
	TileSizePx = 256
)

type Tile struct {
	Z uint32
	X float64
	Y float64
}

type GeoCoord struct {
	Lon float64
	Lat float64
}

// Returns lat/lon coordinates of top left corner of requested tile
func TileToLonLat(zoom uint32, x float64, y float64) (lon, lat float64) {
	maxtiles := float64(uint64(1 << zoom))

	lon = 360.0 * (x/maxtiles - 0.5)
	lat = 2.0*math.Atan(math.Exp(math.Pi-(2*math.Pi)*(y/maxtiles)))*(180.0/math.Pi) - 90.0

	return lon, lat
}

// Returns lat/lon coordinates of center of requested tile
func TileCenterToLonLat(zoom uint32, x float64, y float64) (lon, lat float64) {

	zoom += 1
	x = x*2 + 1
	y = y*2 + 1

	maxtiles := float64(uint64(1 << zoom))

	lon = 360.0 * (x/maxtiles - 0.5)
	lat = 2.0*math.Atan(math.Exp(math.Pi-(2*math.Pi)*(y/maxtiles)))*(180.0/math.Pi) - 90.0

	return lon, lat
}

// resolution = 156543.03 meters/pixel * cos(latitude) / (2 ^ zoomlevel)

// Returns meters per pixel at the center of requested tile
func TilePixelResolution(zoom uint32, x float64, y float64) (float64, error) {

	// from tile coord to lat/lon
	var latRad float64

	if x == 0 && y == 0 {
		latRad = 0
	} else {
		_, lat := TileCenterToLonLat(zoom, x, y)

		// radians = degrees * (pi/180)
		// degrees = radians * (180/pi)

		//lonRad := lon * math.Pi / 180.0
		latRad = lat * math.Pi / 180.0
	}

	res := 156543.0351 * math.Cos(latRad) / float64(math.Pow(2, float64(zoom)))

	return res, nil
}
