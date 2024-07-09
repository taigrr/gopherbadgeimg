package main

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"

	"github.com/makeworld-the-better-one/dither"
	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %v <profile | splash> <infile.png>", os.Args[0])
	}
	infile := os.Args[2]
	if _, err := os.Stat(infile); err != nil {
		log.Fatalf("could not stat %v: %v", infile, err)
	}
	sourceImage, err := LoadImg(infile)
	if err != nil {
		log.Fatalf("error loading source image: %v", err)
	}
	var imgBits []byte

	// profile image is 128x120
	// splash image is 246x128
	switch os.Args[1] {
	case "profile":
		imgBits = ImgToBytes(120, 128, sourceImage)
		if err != nil {
			log.Fatalf("error: could not translate image to bytes: %v", err)
		}
	case "splash":
		imgBits = ImgToBytes(246, 128, sourceImage)
	default:
		log.Fatalf("unknown command %v", os.Args[1])
	}
	err = WriteToGoFile(fmt.Sprintf("%s-generated.go", os.Args[1]), os.Args[1], imgBits)
	if err != nil {
		log.Fatalf("error writing image to file: %v", err)
	}
	err = WriteToBinFile(fmt.Sprintf("%s.bin", os.Args[1]), imgBits)
	if err != nil {
		log.Fatalf("error writing image to file: %v", err)
	}
	fmt.Println(EncodeToString(imgBits))
}

func EncodeToString(imageBits []byte) string {
	return base64.StdEncoding.EncodeToString(imageBits)
}

// provides a .bin file that's just the raw bytes of the image
// you can then use a go:embed directive to bake this bin file into your code at
// compile time (be nice to your editor's memory!).
func WriteToBinFile(filename string, imageBits []byte) error {
	outf, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outf.Close()
	_, err = outf.Write(imageBits)
	return err
}

// create a go file with the bytes hardcoded into a variable at build
func WriteToGoFile(filename, variablename string, imageBits []byte) error {
	outf, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outf.Close()
	_, err = outf.Write([]byte("// Code generated by " + os.Args[0] + " DO NOT EDIT.\n\npackage main\n\nvar " + variablename + " = []byte{"))
	if err != nil {
		return err
	}

	for i, b := range imageBits {
		if i%32 == 0 {
			_, err = outf.Write([]byte("\n\t"))
			if err != nil {
				return err
			}
		}
		bStr := fmt.Sprintf("0x%02X, ", b)
		_, err = outf.Write([]byte(bStr))
		if err != nil {
			return err
		}
	}
	_, err = outf.Write([]byte("\n}\n"))
	return err
}

// Loads and decodes filename into image.Image pointer
func LoadImg(infile string) (*image.Image, error) {
	f, err := os.Open(infile)
	if err != nil {
		return nil, err
	}
	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return &src, nil
}

// Resize image to requested size and converts to bitmap byte slice
func ImgToBytes(x, y int, inputImg *image.Image) []byte {
	// work on values not pointers
	src := *inputImg
	// create a new, rectangular image that's the size we want
	dst := image.NewRGBA(image.Rect(0, 0, x, y))
	// use NearestNeighbor algo to fit our original image into the smaller (or bigger!?) image
	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	// Our e-ink display uses one bit for each pixel, on or off.
	// Therefore, we need one bit for each pixel.
	// Since we have a byte slice, and 8 bytes per bit, divide by 8
	imageBits := make([]byte, x*y/8)

	// Again, on or off, white or black are our only color options
	palette := []color.Color{
		color.Black,
		color.White,
	}

	// using our palette, create a dithering struct
	// and dither our image to get some false shading.
	// read more here: https://en.wikipedia.org/wiki/Floyd%E2%80%93Steinberg_dithering
	d := dither.NewDitherer(palette)
	d.Matrix = dither.FloydSteinberg
	dithered := d.Dither(dst)
	// this nil check is necessary since the library will often write
	// the dithered image to dst, but not always. Read their docs for more info
	if dithered == nil {
		dithered = dst
	}

	// loop over the x axis first, then y as screen updates LTR, top to bottom
	// (vertical axis must be inner loop) for the badge layout
	for i := 0; i < x; i++ {
		for j := 0; j < y; j++ {
			// grab dithered image point, determine if bit should be 1 or a 0
			r, g, b, _ := dithered.At(i, j).RGBA()
			if r+g+b == 0 {
				// use bit shifting + integer division & modulo arithmetic to change
				// the individual bits we want to set
				imageBits[(i*y+j)/8] = imageBits[(i*y+j)/8] | (1 << uint(7-(i*y+j)%8))
			}
		}
	}
	return imageBits
}

// This code allows you to re-convert a bitmap variable file back
//func decodeToPng() {
//	dst := image.NewRGBA(image.Rect(0, 0, 246, 128))
//	outPng, _ := os.Create("splash.png")
//	defer outPng.Close()
//	for j := 0; j < 246; j++ {
//		for i := 0; i < 128; i++ {
//			offset := i + j*128
//			bit := tainigo[offset/8] & (1 << uint(7-offset%8))
//			if bit != 0 {
//				dst.Set(245-j, i, color.RGBA{255, 255, 255, 255})
//			} else {
//				dst.Set(245-j, i, color.RGBA{0, 0, 0, 255})
//			}
//		}
//	}
//	png.Encode(outPng, dst)
//}
