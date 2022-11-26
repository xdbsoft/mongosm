package mongosm

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"math"
)

//n returns 2^level
func n(level int) int {
	return 1 << uint(level)
}

//X2Lon transforms x into a longitude in degree at a given level.
func X2Lon(level int, x int) float64 {
	return float64(x)/float64(n(level))*360.0 - 180.0
}

//Lon2X transforms a longitude in degree into x at a given level.
func Lon2X(level int, longitudeDeg float64) int {
	return int(float64(n(level)) * (longitudeDeg + 180.) / 360.)
}

//Y2Lat transforms y into a latitude in degree at a given level.
func Y2Lat(level int, y int) float64 {
	var yosm = y

	latitudeRad := math.Atan(math.Sinh(math.Pi * (1. - 2.*float64(yosm)/float64(n(level)))))
	return -(latitudeRad * 180.0 / math.Pi)
}

//Lat2Y transforms a latitude in degree into y at a given level.
func Lat2Y(level int, latitudeDeg float64) int {
	latitudeRad := latitudeDeg / 180 * math.Pi
	yosm := int(float64(n(level)) * (1. - math.Log(math.Tan(latitudeRad)+1/math.Cos(latitudeRad))/math.Pi) / 2.)

	return n(level) - yosm - 1
}

//Encode encodes an image in the given format. Only jpg and png are supported.
func Encode(img image.Image, format string) ([]byte, error) {

	var b bytes.Buffer
	if format == "jpg" {
		err := jpeg.Encode(&b, img, nil)
		if err != nil {
			return nil, err
		}
	} else if format == "png" {
		err := png.Encode(&b, img)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("Unsupported image format: '" + format + "'. Only 'jpg' or 'png' allowed.")
	}

	return b.Bytes(), nil
}
