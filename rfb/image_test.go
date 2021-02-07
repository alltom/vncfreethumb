package rfb

/*
Measurements on MacBook Air M1, battery power, with go test -bench=. -benchtime=20s

Draw from PixelFormatImage to image.RGBA

Width	FPS	Time
1	11.2M	89.3ns
512	80	12.46ms
1024	21.3	47ms

Draw from image.RGBA to PixelFormatImage

Width	FPS	Time
1	9.5M	105 nanoseconds
512	90.9	11 milliseconds
1024	23.2	43 milliseconds

CopyToRGBA

Width	FPS	Time
1	82.6M	12.1ns
512	465	2.15ms
1024	116	8.63ms

CopyFromRGBA
1	95.2M	10.5ns
512	607	1.65ms
1024	153	6.55ms
*/

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

var (
	pixelFormat = PixelFormat{
		BitsPerPixel: 32,
		BitDepth:     24,
		BigEndian:    true,
		TrueColor:    true,
		RedMax:       0xff, GreenMax: 0xff, BlueMax: 0xff,
		RedShift: 24, GreenShift: 16, BlueShift: 8,
	}
	pixelFormatWeird = PixelFormat{
		BitsPerPixel: 8,
		BitDepth:     6,
		BigEndian:    false,
		TrueColor:    true,
		RedMax:       0b11, GreenMax: 0b11, BlueMax: 0b1111,
		RedShift: 6, GreenShift: 4, BlueShift: 0,
	}
)

func TestColor(t *testing.T) {
	tests := []struct {
		pixelFormat PixelFormat
		r, g, b     uint16
		r2, g2, b2  uint32
	}{
		{pixelFormat, 0xabcd, 0xbcde, 0xcdef, 0xabab, 0xbcbc, 0xcdcd},
		{pixelFormatWeird, 0xffff, 0xaaaa, 0x5555, 0xffff, 0xaaaa, 0x5555},
	}
	for _, test := range tests {
		img, _ := NewPixelFormatImage(test.pixelFormat, image.Rect(0, 0, 1, 1))
		img.Set(0, 0, color.RGBA64{test.r, test.g, test.b, 0xff})
		r2, g2, b2, _ := img.At(0, 0).RGBA()
		if r2 != test.r2 || g2 != test.g2 || b2 != test.b2 {
			t.Errorf("expected <%x, %x, %x>, got <%x, %x, %x>", test.r2, test.g2, test.b2, r2, g2, b2)
		}
	}
}

func benchmarkDrawToRGBA(b *testing.B, width, height int) {
	r := image.Rect(0, 0, width, height)
	src, _ := NewPixelFormatImage(pixelFormat, r)
	dst := image.NewRGBA(r)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		draw.Draw(dst, r, src, src.Bounds().Min, draw.Src)
	}
}

func BenchmarkDrawToRGBA1(b *testing.B)    { benchmarkDrawToRGBA(b, 1, 1) }
func BenchmarkDrawToRGBA512(b *testing.B)  { benchmarkDrawToRGBA(b, 512, 512) }
func BenchmarkDrawToRGBA1024(b *testing.B) { benchmarkDrawToRGBA(b, 1024, 1024) }

func benchmarkCopyToRGBA(b *testing.B, width, height int) {
	r := image.Rect(0, 0, width, height)
	img, _ := NewPixelFormatImage(pixelFormat, r)
	img2 := image.NewRGBA(r)
	for i := 0; i < b.N; i++ {
		if err := img.CopyToRGBA(img2); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkCopyToRGBA1(b *testing.B)    { benchmarkCopyToRGBA(b, 1, 1) }
func BenchmarkCopyToRGBA512(b *testing.B)  { benchmarkCopyToRGBA(b, 512, 512) }
func BenchmarkCopyToRGBA1024(b *testing.B) { benchmarkCopyToRGBA(b, 1024, 1024) }

func benchmarkDrawFromRGBA(b *testing.B, width, height int) {
	r := image.Rect(0, 0, width, height)
	src := image.NewRGBA(r)
	dst, _ := NewPixelFormatImage(pixelFormat, r)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		draw.Draw(dst, r, src, src.Bounds().Min, draw.Src)
	}
}

func BenchmarkDrawFromRGBA1(b *testing.B)    { benchmarkDrawFromRGBA(b, 1, 1) }
func BenchmarkDrawFromRGBA512(b *testing.B)  { benchmarkDrawFromRGBA(b, 512, 512) }
func BenchmarkDrawFromRGBA1024(b *testing.B) { benchmarkDrawFromRGBA(b, 1024, 1024) }

func benchmarkCopyFromRGBA(b *testing.B, width, height int) {
	r := image.Rect(0, 0, width, height)
	dst, _ := NewPixelFormatImage(pixelFormat, r)
	src := image.NewRGBA(r)
	for i := 0; i < b.N; i++ {
		if err := dst.CopyFromRGBA(src); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkCopyFromRGBA1(b *testing.B)    { benchmarkCopyFromRGBA(b, 1, 1) }
func BenchmarkCopyFromRGBA512(b *testing.B)  { benchmarkCopyFromRGBA(b, 512, 512) }
func BenchmarkCopyFromRGBA1024(b *testing.B) { benchmarkCopyFromRGBA(b, 1024, 1024) }
