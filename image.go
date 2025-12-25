package main

import (
	_ "embed"
	"fmt"
	"github.com/wii-tools/arclib"
	tpl "github.com/wii-tools/libtpl"
	"github.com/wii-tools/lzx/lz10"
	"golang.org/x/image/draw"
	"image"
	"io"
	// Importing as a side effect allows for the image library to check for these formats
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

const (
	MaxImageDimension = 8192

	// MaxMailSize is the largest possible size mail can be, as per KD.
	MaxMailSize = 1578040
)

type ImageCoordinates struct {
	Width   int
	Height  int
	XOffset int
	YOffset int
}

var (
	letterMap = map[string]ImageCoordinates{
		"a": {64, 144, 0, 0},
		"b": {384, 144, 64, 0},
		"c": {64, 144, 448, 0},
		"d": {64, 168, 0, 144},
		"e": {384, 168, 64, 144},
		"f": {64, 168, 448, 144},
		"g": {64, 64, 0, 312},
		"h": {384, 64, 64, 312},
		"i": {64, 64, 448, 312},
	}

	//go:embed assets/letter_LZ.bin
	letterArchiveBase []byte

	//go:embed assets/letterhead.u8
	letterHeadArchiveBase []byte
)

func resize(data io.Reader) (image.Image, error) {
	originalImage, _, err := image.Decode(data)
	if err != nil {
		return nil, err
	}

	width := originalImage.Bounds().Size().X
	height := originalImage.Bounds().Size().Y

	if width > MaxImageDimension {
		// Allows for proper scaling.
		height = height * MaxImageDimension / width
		width = MaxImageDimension
	}

	if height > MaxImageDimension {
		width = width * MaxImageDimension / height
		height = MaxImageDimension
	}

	if width != MaxImageDimension && height != MaxImageDimension {
		// No resize needs to occur.
		return originalImage, nil
	}

	newImage := image.NewRGBA(image.Rect(0, 0, width, height))
	// BiLinear mode is the slowest out of the available but offers highest quality.
	draw.BiLinear.Scale(newImage, newImage.Bounds(), originalImage, originalImage.Bounds(), draw.Over, nil)
	return newImage, nil
}

func resizeNoScale(data io.Reader, w int, h int) (image.Image, error) {
	originalImage, _, err := image.Decode(data)
	if err != nil {
		return nil, err
	}

	newImage := image.NewRGBA(image.Rect(0, 0, w, h))
	// BiLinear mode is the slowest out of the available but offers highest quality.
	draw.BiLinear.Scale(newImage, newImage.Bounds(), originalImage, originalImage.Bounds(), draw.Over, nil)
	return newImage, nil
}

func makeLetterImages(data io.Reader) ([]byte, error) {
	// First resize
	resized, err := resizeNoScale(data, 512, 376)
	if err != nil {
		return nil, err
	}

	// Next, we create the archive containing the images.
	letterArc, err := arclib.Load(letterArchiveBase)
	if err != nil {
		return nil, err
	}

	imgDir, err := letterArc.OpenDir("img")
	if err != nil {
		return nil, err
	}

	// After, we create all 9 separate images.
	for s, coords := range letterMap {
		// Create image for us to write to
		newImage := image.NewRGBA(image.Rect(0, 0, coords.Width, coords.Height))

		// Create the rectangle at which the current image data is located at
		bounds := image.Rect(coords.XOffset, coords.YOffset, coords.XOffset+coords.Width, coords.YOffset+coords.Height)
		draw.BiLinear.Scale(newImage, newImage.Bounds(), resized, bounds.Bounds(), draw.Over, nil)

		// Convert to TPL, RGB5A3.
		encoded, err := tpl.ToRGB5A3(newImage)
		if err != nil {
			return nil, err
		}

		// Finally, add to the archive
		imgDir.WriteFile(fmt.Sprintf("my_Letter_%s.tpl", s), encoded)
	}

	// Save the edited archive.
	b, err := letterArc.Save()
	if err != nil {
		return nil, err
	}

	return lz10.Compress(b)
}

func makeThumbnail(data io.Reader) ([]byte, error) {
	// First resize
	resized, err := resizeNoScale(data, 144, 96)
	if err != nil {
		return nil, err
	}

	thumbnailArc, err := arclib.Load(letterArchiveBase)
	if err != nil {
		return nil, err
	}

	imgDir, err := thumbnailArc.OpenDir("img")
	if err != nil {
		return nil, err
	}

	tplThumb, err := tpl.ToRGB5A3(resized)
	if err != nil {
		return nil, err
	}

	imgDir.WriteFile("my_LetterS_b.tpl", tplThumb)
	b, err := thumbnailArc.Save()
	if err != nil {
		return nil, err
	}

	return lz10.Compress(b)
}
