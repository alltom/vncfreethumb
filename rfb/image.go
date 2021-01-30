package rfb

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

type PixelFormatImage struct {
	Pix         []uint8
	Rect        image.Rectangle
	PixelFormat PixelFormat
}

type PixelFormatColor struct {
	Pixel       uint32
	PixelFormat PixelFormat
}

func (c PixelFormatColor) RGBA() (r, g, b, a uint32) {
	rb := (c.Pixel >> c.PixelFormat.RedShift) & uint32(c.PixelFormat.RedMax)
	gb := (c.Pixel >> c.PixelFormat.GreenShift) & uint32(c.PixelFormat.GreenMax)
	bb := (c.Pixel >> c.PixelFormat.BlueShift) & uint32(c.PixelFormat.BlueMax)

	// TODO: Integer math.
	r = uint32(float64(rb) / float64(c.PixelFormat.RedMax) * 0xffff)
	g = uint32(float64(gb) / float64(c.PixelFormat.GreenMax) * 0xffff)
	b = uint32(float64(bb) / float64(c.PixelFormat.BlueMax) * 0xffff)
	a = 0xffff
	return
}

func NewPixelFormatImage(pixelFormat PixelFormat, bounds image.Rectangle) *PixelFormatImage {
	bytesPerPixel := int(pixelFormat.BitsPerPixel / 8)
	return &PixelFormatImage{make([]uint8, bytesPerPixel*bounds.Dx()*bounds.Dy()), bounds, pixelFormat}
}

func (img *PixelFormatImage) ColorModel() color.Model {
	panic("not implemented")
}

func (img *PixelFormatImage) Bounds() image.Rectangle {
	return img.Rect
}

func (img *PixelFormatImage) At(x, y int) color.Color {
	idx := img.idx(x, y)
	bo := img.bo()
	var pixel uint32
	switch img.PixelFormat.BitsPerPixel {
	case 8:
		pixel = uint32(img.Pix[idx])
	case 16:
		pixel = uint32(bo.Uint16(img.Pix[idx:]))
	case 32:
		pixel = bo.Uint32(img.Pix[idx:])
	default:
		panic(fmt.Sprintf("BitsPerPixel must be 8, 16, or 32, but it's %d", img.PixelFormat.BitsPerPixel))
	}

	return PixelFormatColor{pixel, img.PixelFormat}

	r := (pixel >> img.PixelFormat.RedShift) & uint32(img.PixelFormat.RedMax)
	g := (pixel >> img.PixelFormat.GreenShift) & uint32(img.PixelFormat.GreenMax)
	b := (pixel >> img.PixelFormat.BlueShift) & uint32(img.PixelFormat.BlueMax)
	if img.PixelFormat.RedMax != 255 || img.PixelFormat.GreenMax != 255 || img.PixelFormat.BlueMax != 255 {
		panic(fmt.Sprintf("max red, green, and blue must be 255, but are %d, %d, and %d", img.PixelFormat.RedMax, img.PixelFormat.GreenMax, img.PixelFormat.BlueMax))
	}
	return color.NRGBA{uint8(r), uint8(g), uint8(b), 0xff}
}

func (img *PixelFormatImage) Set(x, y int, c color.Color) {
	nrgba := color.NRGBAModel.Convert(c).(color.NRGBA)

	if img.PixelFormat.RedMax > 255 || img.PixelFormat.GreenMax > 255 || img.PixelFormat.BlueMax > 255 {
		panic(fmt.Sprintf("max red, green, and blue must be <= 255, but are %d, %d, and %d", img.PixelFormat.RedMax, img.PixelFormat.GreenMax, img.PixelFormat.BlueMax))
	}
	var pixel uint32
	pixel |= (uint32(nrgba.R) & uint32(img.PixelFormat.RedMax)) << img.PixelFormat.RedShift
	pixel |= (uint32(nrgba.G) & uint32(img.PixelFormat.GreenMax)) << img.PixelFormat.GreenShift
	pixel |= (uint32(nrgba.B) & uint32(img.PixelFormat.BlueMax)) << img.PixelFormat.BlueShift

	idx := img.idx(x, y)
	bo := img.bo()
	switch img.PixelFormat.BitsPerPixel {
	case 8:
		img.Pix[idx] = uint8(pixel)
	case 16:
		bo.PutUint16(img.Pix[idx:], uint16(pixel))
	case 32:
		bo.PutUint32(img.Pix[idx:], pixel)
	default:
		panic(fmt.Sprintf("BitsPerPixel must be 8, 16, or 32, but it's %d", img.PixelFormat.BitsPerPixel))
	}
}

func (img *PixelFormatImage) bo() binary.ByteOrder {
	if img.PixelFormat.BigEndian {
		return binary.BigEndian
	}
	return binary.LittleEndian
}

func (img *PixelFormatImage) idx(x, y int) int {
	bytesPerPixel := int(img.PixelFormat.BitsPerPixel / 8)
	return (bytesPerPixel*img.Rect.Dx())*(y-img.Rect.Min.Y) + bytesPerPixel*(x-img.Rect.Min.X)
}
