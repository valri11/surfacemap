package cmd

import (
	"testing"

	"github.com/fogleman/contourmap"
)

func Test_cnt_simple_1(t *testing.T) {

	width := 5
	heigh := 5

	data := make([]float64, width*heigh)

	//data[0] = 2.0

	//data[2*width+2] = 2.0

	data[3*width+4] = 2.0

	m := contourmap.FromFloat64s(width, heigh, data)

	m = m.Closed()

	zLevel := 1.0

	contours := m.Contours(zLevel)
	for _, contour := range contours {
		for _, point := range contour {
			//t.Logf("%v", point)

			nx := point.X - 1.0
			ny := point.Y - 1.0

			if nx < 0.0 || ny < 0 {
				continue
			}
			if nx >= float64(width-1) || ny >= float64(heigh-1) {
				continue
			}

			t.Logf("x=%f,y=%f", nx, ny)
		}
	}

}
