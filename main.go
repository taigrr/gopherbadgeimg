package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"os"

	"github.com/makeworld-the-better-one/dither"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: %v [profile | splash] <infile.go>", os.Args[0])
	}
	infile := os.Args[2]
	if _, err := os.Stat(infile); err != nil {
		log.Fatalf("could not stat %v: %v", infile, err)
	}
	switch os.Args[1] {
	case "profile":
		imgBuild(infile, 120, 128)
	case "splash":
		imgBuild(infile, 246, 128)
	default:
		log.Fatalf("unknown command %v", os.Args[1])
	}
}

// profile is 128x120// splash is 246x128 bytes, one bit per pixel
func imgBuild(infile string, x, y int) {
	f, err := os.Open(infile)
	if err != nil {
		log.Fatalf("could not open %v: %v", infile, err)
	}
	src, _, err := image.Decode(f)
	if err != nil {
		log.Fatalf("could not decode %v: %v", infile, err)
	}
	dst := image.NewRGBA(image.Rect(0, 0, x, y))
	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)
	imageBits := make([]byte, x*y/8)

	palette := []color.Color{
		color.Black,
		color.White,
	}

	d := dither.NewDitherer(palette)
	d.Matrix = dither.FloydSteinberg

	dithered := d.Dither(dst)
	if dithered == nil {
		dithered = dst
	}

	// Now use img - save it as PNG, display it on the screen, etc

	for i := 0; i < x; i++ {
		for j := 0; j < y; j++ {
			r, g, b, _ := dithered.At(i, j).RGBA()
			// Convert to grayscale
			gray := int(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b))
			if gray > 128 {
				gray = 1
			} else {
				gray = 0
				imageBits[(i*y+j)/8] = imageBits[(i*y+j)/8] | (1 << uint(7-(i*y+j)%8))
			}
		}
	}
	outf, err := os.Create("image.go")
	if err != nil {
		log.Fatalf("could not create image.go: %v", err)
	}
	defer outf.Close()
	outf.Write([]byte("package main\n\n"))
	outf.Write([]byte("var tainigo = []byte{"))

	for i, b := range imageBits {
		if i%32 == 0 {
			outf.Write([]byte("\n\t"))
		}
		bStr := fmt.Sprintf("0x%02X, ", b)
		outf.Write([]byte(bStr))
	}
	outf.Write([]byte("\n}\n"))
}

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
