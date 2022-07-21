package cmd

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sync"
)

func TransparentGrayscale(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	toTransparentPixel := func(rgba *image.NRGBA, x int, y int) color.Color {
		pix_idx := (y*width + x) * 4
		pix := rgba.Pix[pix_idx : pix_idx+4]
		dr := pix[0]
		//dg := uint32(pix[1])
		//db := uint32(pix[2])
		//da := uint32(pix[3])

		//col := color.NRGBA{uint8(dr), uint8(dg), uint8(db), uint8(255)}
		//col := color.NRGBA{uint8(dr), uint8(dg), uint8(db), uint8(255) - uint8(dr)}
		col := color.NRGBA{0, 0, 0, 255 - dr}
		return col
	}

	rgba := image.NewNRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	imgOut := image.NewNRGBA(image.Rectangle{upLeft, lowRight})

	var wg sync.WaitGroup
	sem := make(chan bool, MaxConcurrency)

	for y := 0; y < height; y++ {
		wg.Add(1)
		sem <- true

		y := y
		go func() {
			defer func() {
				<-sem
				wg.Done()
			}()

			for x := 0; x < width; x++ {
				col := toTransparentPixel(rgba, x, y)
				imgOut.Set(x, y, col)
			}
		}()
	}
	wg.Wait()

	return imgOut
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
	/*
		getHeightAtPixel := func(img image.Image, x int, y int) float64 {
			dr, dg, db, da := img.At(x, y).RGBA()
			h := rgbaToHeight(dr, dg, db, da)
			return h
		}
	*/
	getHeightAtPixel := func(rgba *image.RGBA, x int, y int) float64 {
		pix_idx := (y*width + x) * 4
		pix := rgba.Pix[pix_idx : pix_idx+4]
		dr := uint32(pix[0])
		dg := uint32(pix[1])
		db := uint32(pix[2])
		da := uint32(pix[3])

		h := rgbaToHeight(dr, dg, db, da)
		return h
	}

	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

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

				a := getHeightAtPixel(rgba, x-1, y-1)
				b := getHeightAtPixel(rgba, x-1, y)
				c := getHeightAtPixel(rgba, x-1, y+1)
				d := getHeightAtPixel(rgba, x, y-1)
				f := getHeightAtPixel(rgba, x, y+1)
				g := getHeightAtPixel(rgba, x+1, y-1)
				h := getHeightAtPixel(rgba, x+1, y)
				i := getHeightAtPixel(rgba, x+1, y+1)

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

func ColorReliefImage(img image.Image, gm *gradientMap) (image.Image, error) {

	bounds := img.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	getHeightAtPixel := func(rgba *image.RGBA, x int, y int) float64 {
		pix_idx := (y*width + x) * 4
		pix := rgba.Pix[pix_idx : pix_idx+4]
		dr := uint32(pix[0])
		dg := uint32(pix[1])
		db := uint32(pix[2])
		da := uint32(pix[3])

		h := rgbaToHeight(dr, dg, db, da)
		return h
	}

	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	imgOut := image.NewNRGBA(image.Rectangle{upLeft, lowRight})

	var wg sync.WaitGroup
	sem := make(chan bool, MaxConcurrency)

	for y := 0; y < height; y++ {
		wg.Add(1)
		sem <- true

		y := y
		go func() {
			defer func() {
				<-sem
				wg.Done()
			}()

			for x := 0; x < width; x++ {
				h := getHeightAtPixel(rgba, x, y)
				//col := keypoints.HeightToColor(h)
				col := gm.HeightToColor(h)
				imgOut.Set(x, y, col)
			}
		}()
	}
	wg.Wait()

	return imgOut, nil
}
