package main

import (
	"fmt"
	"github.com/alltom/vncfreethumb/rfb"
	"github.com/nfnt/resize"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
)

var (
	foldColor = image.NewUniform(color.RGBA{0, 0, 0xff, 0xff})
)

const (
	windowWidth  = 1200
	windowHeight = 720
)

type UI struct {
	Title         string
	Width, Height int

	windows     []*Window
	pendingCrop image.Rectangle

	keyPressing  bool
	eventHandler func(keyEvent *rfb.KeyEventMessage, pointerEvent *rfb.PointerEventMessage)
}

type Window struct {
	img            image.Image
	crop, lastCrop image.Rectangle
	scale          float64
	scaled         image.Image

	pos    image.Point
	moving bool
}

func (win *Window) ScreenRect() image.Rectangle {
	width := int(float64(win.crop.Dx()) * win.scale)
	height := int(float64(win.crop.Dy()) * win.scale)
	return image.Rect(win.pos.X, win.pos.Y, win.pos.X+width, win.pos.Y+height)
}

func (win *Window) ScreenToWindow(pt image.Point) image.Point {
	sr := win.ScreenRect()
	x := float64(pt.X-sr.Min.X)/win.scale + float64(win.crop.Min.X)
	y := float64(pt.Y-sr.Min.Y)/win.scale + float64(win.crop.Min.Y)
	return image.Pt(int(x), int(y)) // round?
}

func (win *Window) WindowToScreen(pt image.Point) image.Point {
	x := float64(pt.X-win.crop.Min.X)*win.scale + float64(win.pos.X)
	y := float64(pt.Y-win.crop.Min.Y)*win.scale + float64(win.pos.Y)
	return image.Pt(int(x), int(y)) // round?
}

func pmulf(pt image.Point, k float64) image.Point {
	return image.Pt(int(math.Round(float64(pt.X)*k)), int(math.Round(float64(pt.Y)*k)))
}

func rmulf(r image.Rectangle, k float64) image.Rectangle {
	return image.Rectangle{pmulf(r.Min, k), pmulf(r.Max, k)}
}

func (win *Window) Render() {
	r := rmulf(win.img.Bounds(), win.scale)
	scaled := resize.Resize(uint(r.Dx()), uint(r.Dy()), win.img, resize.Lanczos3)
	scaled2 := image.NewRGBA(r)
	draw.Draw(scaled2, r, scaled, scaled.Bounds().Min, draw.Src)
	win.scaled = scaled2
}

func NewUI(wdir string) (*UI, error) {
	fileInfos, err := ioutil.ReadDir(wdir)
	if err != nil {
		return nil, fmt.Errorf("list files in %q: %v", wdir, err)
	}

	var windows []*Window
	for _, info := range fileInfos {
		img, err := func() (image.Image, error) {
			f, err := os.Open(filepath.Join(wdir, info.Name()))
			if err != nil {
				return nil, fmt.Errorf("open image: %v", err)
			}
			img, _, err := image.Decode(f)
			if err != nil {
				return nil, fmt.Errorf("decode %q: %v", info.Name(), err)
			}
			return img, nil
		}()
		if err != nil {
			log.Print(err)
			continue
		}

		win := &Window{img: img, crop: img.Bounds(), lastCrop: img.Bounds(), scale: 0.5, pos: image.Pt(0, 0)}
		win.Render()
		windows = append(windows, win)
	}

	ui := &UI{Title: "freethumb", Width: windowWidth, Height: windowHeight, windows: windows}
	ui.eventHandler = ui.defaultEventHandler
	return ui, nil
}

func (ui *UI) Update(img draw.Image, keyEvent *rfb.KeyEventMessage, pointerEvent *rfb.PointerEventMessage) image.Rectangle {
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{0xee, 0xee, 0xee, 0xff}), image.ZP, draw.Src)

	ui.eventHandler(keyEvent, pointerEvent)

	for _, win := range ui.windows {
		if win.moving {
			draw.DrawMask(img, image.Rectangle{win.WindowToScreen(win.img.Bounds().Min), win.WindowToScreen(win.img.Bounds().Max)}, win.scaled, image.ZP, image.NewUniform(color.Alpha{0x22}), image.ZP, draw.Over)
		}
		r := win.ScreenRect()
		draw.Draw(img, r, win.scaled, pmulf(win.crop.Min, win.scale), draw.Src)

		if win.crop.Min.X != win.img.Bounds().Min.X {
			draw.Draw(img, image.Rect(r.Min.X-2, r.Min.Y, r.Min.X, r.Max.Y), foldColor, image.ZP, draw.Src)
		}
		if win.crop.Min.Y != win.img.Bounds().Min.Y {
			draw.Draw(img, image.Rect(r.Min.X, r.Min.Y-2, r.Max.X, r.Min.Y), foldColor, image.ZP, draw.Src)
		}
		if win.crop.Max.X != win.img.Bounds().Max.X {
			draw.Draw(img, image.Rect(r.Max.X, r.Min.Y, r.Max.X+2, r.Max.Y), foldColor, image.ZP, draw.Src)
		}
		if win.crop.Max.Y != win.img.Bounds().Max.Y {
			draw.Draw(img, image.Rect(r.Min.X, r.Max.Y, r.Max.X, r.Max.Y+2), foldColor, image.ZP, draw.Src)
		}
	}

	draw.Draw(img, ui.pendingCrop, image.NewUniform(color.NRGBA{0xb7, 0x96, 0xd4, 0x88}), image.ZP, draw.Over)

	return image.Rect(0, 0, ui.Width, ui.Height)
}

func (ui *UI) moveToFront(windowIdx int) {
	win := ui.windows[windowIdx]
	for i := windowIdx; i <= len(ui.windows)-2; i++ {
		ui.windows[i] = ui.windows[i+1]
	}
	ui.windows[len(ui.windows)-1] = win
}

func (ui *UI) defaultEventHandler(keyEvent *rfb.KeyEventMessage, pointerEvent *rfb.PointerEventMessage) {
	loc := image.Pt(int(pointerEvent.X), int(pointerEvent.Y))
	targetWin := -1
	for idx, win := range ui.windows {
		if loc.In(win.ScreenRect()) {
			targetWin = idx
		}
	}

	if !ui.keyPressing && keyEvent.Pressed && targetWin != -1 {
		win := ui.windows[targetWin]
		oldcrop := win.crop
		switch keyEvent.KeySym {
		case 119: // w
			if win.crop.Min.Y != win.img.Bounds().Min.Y {
				win.crop.Min.Y = win.img.Bounds().Min.Y
			} else {
				win.crop.Min.Y = win.ScreenToWindow(loc).Y
			}
		case 97: // a
			if win.crop.Min.X != win.img.Bounds().Min.X {
				win.crop.Min.X = win.img.Bounds().Min.X
			} else {
				win.crop.Min.X = win.ScreenToWindow(loc).X
			}
		case 115: // s
			if win.crop.Max.Y != win.img.Bounds().Max.Y {
				win.crop.Max.Y = win.img.Bounds().Max.Y
			} else {
				win.crop.Max.Y = win.ScreenToWindow(loc).Y
			}
		case 100: // d
			if win.crop.Max.X != win.img.Bounds().Max.X {
				win.crop.Max.X = win.img.Bounds().Max.X
			} else {
				win.crop.Max.X = win.ScreenToWindow(loc).X
			}
		}
		win.pos.X += int(float64(win.crop.Min.X-oldcrop.Min.X) * win.scale)
		win.pos.Y += int(float64(win.crop.Min.Y-oldcrop.Min.Y) * win.scale)

		ui.keyPressing = true
	}
	if !keyEvent.Pressed {
		ui.keyPressing = false
	}

	if pointerEvent.ButtonMask&0b1 > 0 && targetWin != -1 {
		win := ui.windows[targetWin]
		ui.moveToFront(targetWin)

		win.moving = true
		lastX, lastY := loc.X, loc.Y
		ui.eventHandler = func(keyEvent *rfb.KeyEventMessage, pointerEvent *rfb.PointerEventMessage) {
			if pointerEvent.ButtonMask&0b1 == 0 {
				win.moving = false
				ui.eventHandler = ui.defaultEventHandler
				return
			}
			dp := image.Pt(int(pointerEvent.X)-lastX, int(pointerEvent.Y)-lastY)
			win.pos = win.pos.Add(dp)
			lastX, lastY = int(pointerEvent.X), int(pointerEvent.Y)
		}
	} else if pointerEvent.ButtonMask&0b100 > 0 && targetWin != -1 {
		win := ui.windows[targetWin]
		ui.moveToFront(targetWin)

		ui.eventHandler = func(keyEvent *rfb.KeyEventMessage, pointerEvent *rfb.PointerEventMessage) {
			loc2 := image.Pt(int(pointerEvent.X), int(pointerEvent.Y))
			if pointerEvent.ButtonMask&0b100 == 0 {
				if math.Hypot(float64(loc2.X-loc.X), float64(loc2.Y-loc.Y)) < 10 {
					oldcrop := win.crop
					if win.img.Bounds() == win.crop {
						win.crop, win.lastCrop = win.lastCrop, win.crop
					} else {
						win.crop, win.lastCrop = win.img.Bounds(), win.crop
					}
					win.pos.X += int(float64(win.crop.Min.X-oldcrop.Min.X) * win.scale)
					win.pos.Y += int(float64(win.crop.Min.Y-oldcrop.Min.Y) * win.scale)
				} else {
					newcrop := image.Rectangle{win.ScreenToWindow(loc), win.ScreenToWindow(loc2)}.Canon()
					win.pos = win.WindowToScreen(newcrop.Min)
					win.crop = newcrop
				}
				ui.pendingCrop = image.ZR

				ui.eventHandler = ui.defaultEventHandler
				return
			}
			ui.pendingCrop = image.Rectangle{loc, loc2}.Canon()
		}
	}
}
