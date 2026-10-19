package main

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	profileBytes = profileWidth * profileHeight / 8
	splashBytes  = splashWidth * splashHeight / 8
)

// createTestImage generates a small test PNG at the given path.
func createTestImage(t *testing.T, path string, width, height int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// fill with a gradient so dithering has something to work with
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gray := uint8((x + y) * 255 / (width + height))
			img.Set(x, y, color.RGBA{R: gray, G: gray, B: gray, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}

func TestLoadImg(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "test.png")
	createTestImage(t, imgPath, 64, 64)

	img, err := LoadImg(imgPath)
	if err != nil {
		t.Fatalf("LoadImg failed: %v", err)
	}
	if img == nil {
		t.Fatal("LoadImg returned nil image")
	}
	bounds := img.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		t.Fatalf("expected 64x64, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestLoadImg_NotFound(t *testing.T) {
	_, err := LoadImg("/nonexistent/file.png")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadImg_InvalidFile(t *testing.T) {
	tmp := t.TempDir()
	badPath := filepath.Join(tmp, "bad.png")
	if err := os.WriteFile(badPath, []byte("not an image"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadImg(badPath)
	if err == nil {
		t.Fatal("expected error for invalid image data")
	}
}

func TestImgToBytes_Profile(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "test.png")
	createTestImage(t, imgPath, 200, 200)

	img, err := LoadImg(imgPath)
	if err != nil {
		t.Fatalf("LoadImg failed: %v", err)
	}

	// profile: 120x128 = 15360 pixels / 8 = 1920 bytes
	result := ImgToBytes(profileWidth, profileHeight, img)
	if len(result) != profileBytes {
		t.Fatalf("expected %d bytes, got %d", profileBytes, len(result))
	}
}

func TestImgToBytes_Splash(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "test.png")
	createTestImage(t, imgPath, 300, 200)

	img, err := LoadImg(imgPath)
	if err != nil {
		t.Fatalf("LoadImg failed: %v", err)
	}

	// splash: 246x128 = 31488 pixels / 8 = 3936 bytes
	result := ImgToBytes(splashWidth, splashHeight, img)
	if len(result) != splashBytes {
		t.Fatalf("expected %d bytes, got %d", splashBytes, len(result))
	}
}

func TestImgToBytes_AllWhite(t *testing.T) {
	// all-white image should produce all-zero bytes (black bits are set)
	whiteImg := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			whiteImg.Set(x, y, color.White)
		}
	}
	var img image.Image = whiteImg
	result := ImgToBytes(8, 8, img)
	for i, b := range result {
		if b != 0 {
			t.Fatalf("expected all zeros for white image, got 0x%02X at index %d", b, i)
		}
	}
}

func TestImgToBytes_AllBlack(t *testing.T) {
	// all-black image should produce all-set bytes
	blackImg := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			blackImg.Set(x, y, color.Black)
		}
	}
	var img image.Image = blackImg
	result := ImgToBytes(8, 8, img)
	for i, b := range result {
		if b != 0xFF {
			t.Fatalf("expected all 0xFF for black image, got 0x%02X at index %d", b, i)
		}
	}
}

func TestEncodeToString(t *testing.T) {
	input := []byte{0x00, 0xFF, 0xAB, 0xCD}
	result := EncodeToString(input)
	expected := base64.StdEncoding.EncodeToString(input)
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestEncodeToString_Empty(t *testing.T) {
	result := EncodeToString([]byte{})
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestWriteToBinFile(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "test.bin")
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	if err := WriteToBinFile(outPath, data); err != nil {
		t.Fatalf("WriteToBinFile failed: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading output file: %v", err)
	}
	if len(got) != len(data) {
		t.Fatalf("expected %d bytes, got %d", len(data), len(got))
	}
	for i := range data {
		if got[i] != data[i] {
			t.Fatalf("byte %d: expected 0x%02X, got 0x%02X", i, data[i], got[i])
		}
	}
}

func TestWriteToBinFile_BadPath(t *testing.T) {
	err := WriteToBinFile("/nonexistent/dir/file.bin", []byte{0x01})
	if err == nil {
		t.Fatal("expected error for bad path")
	}
}

func TestWriteToGoFile(t *testing.T) {
	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "test-generated.go")
	data := []byte{0x01, 0x02, 0x03}

	if err := WriteToGoFile(outPath, "testVar", data); err != nil {
		t.Fatalf("WriteToGoFile failed: %v", err)
	}

	got, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(got)

	if !strings.Contains(content, "package main") {
		t.Fatal("missing package declaration")
	}
	if !strings.Contains(content, "var testVar = []byte{") {
		t.Fatal("missing variable declaration")
	}
	if !strings.Contains(content, "DO NOT EDIT") {
		t.Fatal("missing generated comment")
	}
	if !strings.Contains(content, "0x01") {
		t.Fatal("missing byte value 0x01")
	}
}

func TestWriteToGoFile_BadPath(t *testing.T) {
	err := WriteToGoFile("/nonexistent/dir/gen.go", "x", []byte{0x01})
	if err == nil {
		t.Fatal("expected error for bad path")
	}
}

func TestDecodeSplashBin(t *testing.T) {
	splashData := make([]byte, splashBytes)
	if len(splashData) != splashBytes {
		t.Fatalf("expected splash data to be %d bytes, got %d", splashBytes, len(splashData))
	}

	tmp := t.TempDir()
	outPath := filepath.Join(tmp, "splash.png")

	dst := image.NewRGBA(image.Rect(0, 0, splashWidth, splashHeight))
	for j := 0; j < splashWidth; j++ {
		for i := 0; i < splashHeight; i++ {
			offset := i + j*splashHeight
			bit := splashData[offset/8] & (1 << uint(7-offset%8))
			if bit != 0 {
				dst.Set(splashWidth-1-j, i, color.RGBA{R: 255, G: 255, B: 255, A: 255})
			} else {
				dst.Set(splashWidth-1-j, i, color.RGBA{R: 0, G: 0, B: 0, A: 255})
			}
		}
	}

	outPng, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outPng.Close()

	if err := png.Encode(outPng, dst); err != nil {
		t.Fatal(err)
	}

	// Verify the file was created and is non-empty
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("generated PNG is empty")
	}
}

func TestRun_Profile(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "test.png")
	createTestImage(t, imgPath, 200, 200)

	// run from tmp dir so generated files land there
	t.Chdir(tmp)

	err := run([]string{"gopherbadgeimg", "profile", imgPath})
	if err != nil {
		t.Fatalf("run profile failed: %v", err)
	}

	// check that profile-generated.go was created
	if _, err := os.Stat(filepath.Join(tmp, "profile-generated.go")); err != nil {
		t.Fatal("profile-generated.go not created")
	}
	// check that profile.bin was created
	info, err := os.Stat(filepath.Join(tmp, "profile.bin"))
	if err != nil {
		t.Fatal("profile.bin not created")
	}
	if info.Size() != profileBytes {
		t.Fatalf("profile.bin expected %d bytes, got %d", profileBytes, info.Size())
	}
}

func TestRun_Splash(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "test.png")
	createTestImage(t, imgPath, 300, 200)

	t.Chdir(tmp)

	err := run([]string{"gopherbadgeimg", "splash", imgPath})
	if err != nil {
		t.Fatalf("run splash failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(tmp, "splash.bin"))
	if err != nil {
		t.Fatal("splash.bin not created")
	}
	if info.Size() != splashBytes {
		t.Fatalf("splash.bin expected %d bytes, got %d", splashBytes, info.Size())
	}
}

func TestRun_WrongArgCount(t *testing.T) {
	err := run([]string{"gopherbadgeimg"})
	if err == nil {
		t.Fatal("expected error for wrong arg count")
	}
}

func TestRun_NoArgs(t *testing.T) {
	err := run(nil)
	if err == nil {
		t.Fatal("expected error for missing args")
	}
	if !strings.Contains(err.Error(), "usage: gopherbadgeimg") {
		t.Fatalf("expected fallback executable name in error, got: %v", err)
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	tmp := t.TempDir()
	imgPath := filepath.Join(tmp, "test.png")
	createTestImage(t, imgPath, 64, 64)

	err := run([]string{"gopherbadgeimg", "invalid", imgPath})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("expected 'unknown command' in error, got: %v", err)
	}
}

func TestRun_NonexistentFile(t *testing.T) {
	err := run([]string{"gopherbadgeimg", "profile", "/nonexistent/file.png"})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
