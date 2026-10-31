package game

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed assets
var assetFS embed.FS

func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

func loadPNG(path string) image.Image {
	b, err := assetFS.ReadFile(path)
	if err != nil {
		return nil
	}
	im, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil
	}
	return im
}

var imgCache = map[string]*ebiten.Image{}

func ebImage(path string) *ebiten.Image {
	if im, ok := imgCache[path]; ok {
		return im
	}
	var out *ebiten.Image
	if im := loadPNG(path); im != nil {
		out = ebiten.NewImageFromImage(im)
	}
	imgCache[path] = out
	return out
}

type imageRect = image.Rectangle

func imageRectFrom(x0, y0, x1, y1 int) image.Rectangle { return image.Rect(x0, y0, x1, y1) }
