package rfb

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
)

// PixelFormatImage represents an image using the wire format specified by PixelFormat. Supports arbitrary drawing with At and Set, but for speed, use CopyToRGBA and CopyFromRGBA.
type PixelFormatImage struct {
	Pix         []uint8
	Rect        image.Rectangle
	PixelFormat PixelFormat

	bo            binary.ByteOrder
	bytesPerPixel int
}

// PixelFormatColor represents a color using the wire format specified by PixelFormat.
type PixelFormatColor struct {
	Pixel       uint32
	PixelFormat PixelFormat
}

func (c PixelFormatColor) RGBA() (r, g, b, a uint32) {
	// Extract components
	r = (c.Pixel >> c.PixelFormat.RedShift) & uint32(c.PixelFormat.RedMax)
	g = (c.Pixel >> c.PixelFormat.GreenShift) & uint32(c.PixelFormat.GreenMax)
	b = (c.Pixel >> c.PixelFormat.BlueShift) & uint32(c.PixelFormat.BlueMax)

	// Scale to 0xffff
	r = (r * 0xffff) / uint32(c.PixelFormat.RedMax)
	g = (g * 0xffff) / uint32(c.PixelFormat.GreenMax)
	b = (b * 0xffff) / uint32(c.PixelFormat.BlueMax)
	a = 0xffff

	return
}

func NewPixelFormatImage(pixelFormat PixelFormat, bounds image.Rectangle) (*PixelFormatImage, error) {
	if pixelFormat.RedMax > 0xffff || pixelFormat.GreenMax > 0xffff || pixelFormat.BlueMax > 0xffff {
		return nil, fmt.Errorf("RedMax, GreenMax, and BlueMax must be less than 0xffff, but were %x, %x, and %x", pixelFormat.RedMax, pixelFormat.GreenMax, pixelFormat.BlueMax)
	}
	if pixelFormat.BitsPerPixel != 32 && pixelFormat.BitsPerPixel != 16 && pixelFormat.BitsPerPixel != 8 {
		return nil, fmt.Errorf("BitsPerPixel must be 32, 16, or 8, but it's %d", pixelFormat.BitsPerPixel)
	}

	bytesPerPixel := int(pixelFormat.BitsPerPixel / 8)
	var bo binary.ByteOrder = binary.BigEndian
	if !pixelFormat.BigEndian {
		bo = binary.LittleEndian
	}
	return &PixelFormatImage{
		make([]uint8, bytesPerPixel*bounds.Dx()*bounds.Dy()),
		bounds,
		pixelFormat,
		bo,
		bytesPerPixel,
	}, nil
}

func (img *PixelFormatImage) ColorModel() color.Model {
	panic("not implemented")
}

func (img *PixelFormatImage) Bounds() image.Rectangle {
	return img.Rect
}

func (img *PixelFormatImage) At(x, y int) color.Color {
	idx := img.idx(x, y)
	var pixel uint32
	switch img.PixelFormat.BitsPerPixel {
	case 8:
		pixel = uint32(img.Pix[idx])
	case 16:
		pixel = uint32(img.bo.Uint16(img.Pix[idx:]))
	case 32:
		pixel = img.bo.Uint32(img.Pix[idx:])
	default:
		// Checked in NewPixelFormatImage.
		panic("unsupported BitsPerPixel")
	}

	return PixelFormatColor{pixel, img.PixelFormat}
}

func (img *PixelFormatImage) Set(x, y int, c color.Color) {
	r, g, b, _ := c.RGBA()

	var pixel uint32
	pixel |= (r * uint32(img.PixelFormat.RedMax) / 0xffff) << img.PixelFormat.RedShift
	pixel |= (g * uint32(img.PixelFormat.GreenMax) / 0xffff) << img.PixelFormat.GreenShift
	pixel |= (b * uint32(img.PixelFormat.BlueMax) / 0xffff) << img.PixelFormat.BlueShift

	idx := img.idx(x, y)
	switch img.PixelFormat.BitsPerPixel {
	case 8:
		img.Pix[idx] = uint8(pixel)
	case 16:
		img.bo.PutUint16(img.Pix[idx:], uint16(pixel))
	case 32:
		img.bo.PutUint32(img.Pix[idx:], pixel)
	default:
		panic(fmt.Sprintf("BitsPerPixel must be 8, 16, or 32, but it's %d", img.PixelFormat.BitsPerPixel))
	}
}

func (src *PixelFormatImage) CopyToRGBA(dst *image.RGBA) error {
	if src.Bounds() != dst.Bounds() {
		return fmt.Errorf("expected dst bounds to be %v, but was %v", src.Bounds(), dst.Bounds())
	}

	dstidx := 0
	for srcidx := 0; srcidx < len(src.Pix); srcidx += src.bytesPerPixel {
		var pixel uint32
		switch src.PixelFormat.BitsPerPixel {
		case 8:
			pixel = uint32(src.Pix[srcidx])
		case 16:
			pixel = uint32(src.bo.Uint16(src.Pix[srcidx:]))
		case 32:
			pixel = src.bo.Uint32(src.Pix[srcidx:])
		default:
			// Checked in NewPixelFormatImage.
			panic("unsupported BitsPerPixel")
		}

		// Extract components
		r := (pixel >> src.PixelFormat.RedShift) & uint32(src.PixelFormat.RedMax)
		g := (pixel >> src.PixelFormat.GreenShift) & uint32(src.PixelFormat.GreenMax)
		b := (pixel >> src.PixelFormat.BlueShift) & uint32(src.PixelFormat.BlueMax)

		// Scale to 0xff
		dst.Pix[dstidx] = uint8((r * 0xff) / uint32(src.PixelFormat.RedMax))
		dst.Pix[dstidx+1] = uint8((g * 0xff) / uint32(src.PixelFormat.GreenMax))
		dst.Pix[dstidx+2] = uint8((b * 0xff) / uint32(src.PixelFormat.BlueMax))
		dst.Pix[dstidx+3] = 0xff
		dstidx += 4
	}
	return nil
}

// CopyFromRGBA copies src into dst, returning an error if their bounds are not exactly equal. The alpha channel is ignored.
func (dst *PixelFormatImage) CopyFromRGBA(src *image.RGBA) error {
	if src.Bounds() != dst.Bounds() {
		return fmt.Errorf("expected dst bounds to be %v, but was %v", src.Bounds(), dst.Bounds())
	}

	dstidx := 0
	for srcidx := 0; srcidx < len(src.Pix); srcidx += 4 {
		var pixel uint32
		pixel |= ((uint32(src.Pix[srcidx]) * uint32(dst.PixelFormat.RedMax)) / 0xff) << dst.PixelFormat.RedShift
		pixel |= ((uint32(src.Pix[srcidx+1]) * uint32(dst.PixelFormat.GreenMax)) / 0xff) << dst.PixelFormat.GreenShift
		pixel |= ((uint32(src.Pix[srcidx+2]) * uint32(dst.PixelFormat.BlueMax)) / 0xff) << dst.PixelFormat.BlueShift

		switch dst.PixelFormat.BitsPerPixel {
		case 8:
			dst.Pix[dstidx] = uint8(pixel)
		case 16:
			dst.bo.PutUint16(dst.Pix[dstidx:], uint16(pixel))
		case 32:
			dst.bo.PutUint32(dst.Pix[dstidx:], pixel)
		default:
			// Checked in NewPixelFormatImage.
			panic("unsupported BitsPerPixel")
		}
		dstidx += dst.bytesPerPixel
	}
	return nil
}

func (img *PixelFormatImage) idx(x, y int) int {
	return (img.bytesPerPixel*img.Rect.Dx())*(y-img.Rect.Min.Y) + img.bytesPerPixel*(x-img.Rect.Min.X)
}
