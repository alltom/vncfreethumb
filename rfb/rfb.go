/*
Package rfb defines representations and serialization for messages in the RFB (Remote Framebuffer) protocol, which is used for VNC.

Types that do not not have a protocol version suffix such as "RFB33" are appropriate for use with all versions of the RFB protocol.

See the RFCs for details, but the initial handshake goes like this:

	server sends ProtocolVersionMessage
	client sends ProtocolVersionMessage
	server sends AuthenticationSchemeMessageRFB33
		If AuthenticationSchemeVNC:
			server sends VNCAuthenticationChallengeMessage
			client sends VNCAuthenticationResponseMessage
			server sends VNCAuthenticationResultMessage
	client sends ClientInitialisationMessage
	server sends ServerInitialisationMessage

Thereafter, client and server enter message processing loops. The first byte identifies the message type, which dictates the length of the payload, so all clients and servers must process all event types.

Clients may send:

	Type 0	SetPixelFormatMessage
	Type 1	FixColourMapEntries — uncommon, not implemented by this library
	Type 2	SetEncodingsMessage
	Type 3	FramebufferUpdateRequestMessage
	Type 4	KeyEventMessage
	Type 5	PointerEventMessage
	Type 6	ClientCutTextMessage

Servers may send:

	Type 0	FramebufferUpdate
	Type 1	SetColourMapEntries — uncommon, not implemented by this library
	Type 2	BellMessage
	Type 3	ServerCutTextMessage
*/
package rfb

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/text/encoding/charmap"
	"io"
)

type ProtocolVersionMessage struct {
	Major, Minor int
}

func (m *ProtocolVersionMessage) Read(r io.Reader) error {
	var buf [12]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	if _, err := fmt.Sscanf(string(buf[:]), "RFB %03d.%03d\n", &m.Major, &m.Minor); err != nil {
		return fmt.Errorf("parse: %v", err)
	}
	return nil
}

func (m *ProtocolVersionMessage) Write(w io.Writer) error {
	buf := []byte(fmt.Sprintf("RFB %03d.%03d\n", m.Major, m.Minor))
	if len(buf) != 12 {
		return fmt.Errorf("expected formatted message to be 12 bytes, but %q is %d", string(buf), len(buf))
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return nil
}

type AuthenticationSchemeMessageRFB33 struct {
	Scheme AuthenticationScheme
}

type AuthenticationScheme uint32

const (
	AuthenticationSchemeInvalid = AuthenticationScheme(0)
	AuthenticationSchemeNone    = AuthenticationScheme(1)
	AuthenticationSchemeVNC     = AuthenticationScheme(2)
)

func (m *AuthenticationSchemeMessageRFB33) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	m.Scheme = AuthenticationScheme(bo.Uint32(buf[:]))
	return nil
}

func (m *AuthenticationSchemeMessageRFB33) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [4]byte
	bo.PutUint32(buf[:], uint32(m.Scheme))
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	return nil
}

type VNCAuthenticationChallengeMessage [16]byte

func (m *VNCAuthenticationChallengeMessage) Read(r io.Reader) error {
	_, err := io.ReadFull(r, m[:])
	return err
}

func (m *VNCAuthenticationChallengeMessage) Write(w io.Writer) error {
	_, err := w.Write(m[:])
	return err
}

type VNCAuthenticationResponseMessage [16]byte

func (m *VNCAuthenticationResponseMessage) Read(r io.Reader) error {
	_, err := io.ReadFull(r, m[:])
	return err
}

func (m *VNCAuthenticationResponseMessage) Write(w io.Writer) error {
	_, err := w.Write(m[:])
	return err
}

type VNCAuthenticationResultMessage struct {
	Result VNCAuthenticationResult
}

type VNCAuthenticationResult uint32

const (
	VNCAuthenticationResultOK      = VNCAuthenticationResult(0)
	VNCAuthenticationResultFailed  = VNCAuthenticationResult(1)
	VNCAuthenticationResultTooMany = VNCAuthenticationResult(2)
)

func (m *VNCAuthenticationResultMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	m.Result = VNCAuthenticationResult(bo.Uint32(buf[:]))
	return nil
}

func (m *VNCAuthenticationResultMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [4]byte
	bo.PutUint32(buf[:], uint32(m.Result))
	_, err := w.Write(buf[:])
	return err
}

type ClientInitialisationMessage struct {
	// If true, share the desktop with other clients.
	// If false, disconnect all other clients.
	Shared bool
}

func (m *ClientInitialisationMessage) Read(r io.Reader) error {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	m.Shared = buf[0] != 0
	return nil
}

func (m *ClientInitialisationMessage) Write(w io.Writer) error {
	var buf [1]byte
	if m.Shared {
		buf[0] = 1
	}
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	return nil
}

type ServerInitialisationMessage struct {
	FramebufferWidth  uint16
	FramebufferHeight uint16
	PixelFormat       PixelFormat
	Name              string
}

func (m *ServerInitialisationMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [255]byte
	if _, err := io.ReadFull(r, buf[:24]); err != nil {
		return err
	}
	m.FramebufferWidth = bo.Uint16(buf[0:])
	m.FramebufferHeight = bo.Uint16(buf[2:])
	m.PixelFormat.Read(buf[4:], bo)
	nameLength := bo.Uint32(buf[20:])
	if int(nameLength) > len(buf) {
		return fmt.Errorf("name is too long: %d > %d", nameLength, len(buf))
	}
	if _, err := io.ReadFull(r, buf[:nameLength]); err != nil {
		return err
	}
	m.Name = string(buf[:nameLength])
	return nil
}

func (m *ServerInitialisationMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [24]byte
	bo.PutUint16(buf[0:], m.FramebufferWidth)
	bo.PutUint16(buf[2:], m.FramebufferHeight)
	m.PixelFormat.Write(buf[4:], bo)
	bo.PutUint32(buf[20:], uint32(len([]byte(m.Name))))
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	if _, err := w.Write([]byte(m.Name)); err != nil {
		return err
	}
	return nil
}

type SetPixelFormatMessage struct {
	PixelFormat PixelFormat
}

func (m *SetPixelFormatMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [20]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	if buf[0] != 0 {
		return fmt.Errorf("expected message type 0, but found %d", buf[0])
	}
	m.PixelFormat.Read(buf[4:], bo)
	return nil
}

func (m *SetPixelFormatMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [16]byte
	m.PixelFormat.Write(buf[4:], bo)
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	return nil
}

type SetEncodingsMessage struct {
	EncodingTypes []uint32
}

const (
	EncodingTypeRaw           = uint32(0)
	EncodingTypeCopyRectangle = uint32(1)
	EncodingTypeRRE           = uint32(2)
	EncodingTypeCoRRE         = uint32(4)
	EncodingTypeHextile       = uint32(5)
)

func (m *SetEncodingsMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [255]byte
	if _, err := io.ReadFull(r, buf[:4]); err != nil {
		return err
	}
	if buf[0] != 2 {
		return fmt.Errorf("expected message type 2, but found %d", buf[0])
	}
	encodingCount := bo.Uint16(buf[2:])
	if int(encodingCount) > len(buf)/4 {
		return fmt.Errorf("too many encodings: %d > %d", encodingCount, len(buf)/4)
	}
	if _, err := io.ReadFull(r, buf[:encodingCount*4]); err != nil {
		return err
	}
	m.EncodingTypes = nil
	for i := uint16(0); i < encodingCount; i++ {
		m.EncodingTypes = append(m.EncodingTypes, bo.Uint32(buf[i*4:]))
	}
	return nil
}

func (m *SetEncodingsMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [255]byte

	maxCount := len(buf[4:]) / 4
	if len(m.EncodingTypes) > maxCount {
		return fmt.Errorf("too many encoding types: %d > %d", len(m.EncodingTypes), maxCount)
	}

	buf[0] = 2
	bo.PutUint16(buf[2:], uint16(len(m.EncodingTypes)))
	for idx, encodingType := range m.EncodingTypes {
		bo.PutUint32(buf[4+idx*4:], encodingType)
	}
	if _, err := w.Write(buf[:4+4*len(m.EncodingTypes)]); err != nil {
		return err
	}
	return nil
}

type FramebufferUpdateRequestMessage struct {
	// If true, only updates to changed portions of the framebuffer are requested.
	// If false, the entire region should be returned and EncodingTypeCopyRectangle is not supported.
	Incremental bool

	X      uint16
	Y      uint16
	Width  uint16
	Height uint16
}

func (m *FramebufferUpdateRequestMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [10]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	if buf[0] != 3 {
		return fmt.Errorf("expected message type 3, but found %d", buf[0])
	}

	m.Incremental = buf[1] != 0
	m.X = bo.Uint16(buf[2:])
	m.Y = bo.Uint16(buf[4:])
	m.Width = bo.Uint16(buf[6:])
	m.Height = bo.Uint16(buf[8:])

	return nil
}

func (m *FramebufferUpdateRequestMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [10]byte
	buf[0] = 3 // Message type
	if m.Incremental {
		buf[1] = 1
	} else {
		buf[1] = 0
	}
	bo.PutUint16(buf[2:], m.X)
	bo.PutUint16(buf[4:], m.Y)
	bo.PutUint16(buf[6:], m.Width)
	bo.PutUint16(buf[8:], m.Height)
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	return nil
}

type KeyEventMessage struct {
	Pressed bool
	KeySym  uint32 // Defined in Xlib Reference Manual and <X11/keysymdef.h>
}

func (m *KeyEventMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	if buf[0] != 4 {
		return fmt.Errorf("expected message type 4, but found %d", buf[0])
	}
	m.Pressed = buf[1] != 0
	m.KeySym = bo.Uint32(buf[4:])
	return nil
}

func (m *KeyEventMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [8]byte
	buf[0] = 4
	if m.Pressed {
		buf[1] = 1
	}
	bo.PutUint32(buf[4:], m.KeySym)
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	return nil
}

type PointerEventMessage struct {
	ButtonMask uint8
	X          uint16
	Y          uint16
}

func (m *PointerEventMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [6]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	if buf[0] != 5 {
		return fmt.Errorf("expected message type 5, but found %d", buf[0])
	}
	m.ButtonMask = buf[1]
	m.X = bo.Uint16(buf[2:])
	m.Y = bo.Uint16(buf[4:])
	return nil
}

func (m *PointerEventMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [6]byte
	buf[0] = 5
	buf[1] = m.ButtonMask
	bo.PutUint16(buf[2:], m.X)
	bo.PutUint16(buf[4:], m.Y)
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	return nil
}

type ClientCutTextMessage struct {
	Text string
}

func (m *ClientCutTextMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [255]byte
	if _, err := io.ReadFull(r, buf[:8]); err != nil {
		return err
	}
	if buf[0] != 6 {
		return fmt.Errorf("expected message type 6, but found %d", buf[0])
	}
	textLength := bo.Uint32(buf[4:])
	if int(textLength) > len(buf) {
		return fmt.Errorf("text length too long: %d > %d", textLength, len(buf))
	}
	if _, err := io.ReadFull(r, buf[:textLength]); err != nil {
		return err
	}
	converted, err := charmap.ISO8859_1.NewDecoder().Bytes(buf[:textLength])
	if err != nil {
		return fmt.Errorf("couldn't convert text to UTF-8 in ClientCutText: %v", err)
	}
	m.Text = string(converted)
	return nil
}

func (m *ClientCutTextMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	converted, err := charmap.ISO8859_1.NewEncoder().Bytes([]byte(m.Text))
	if err != nil {
		return fmt.Errorf("encode text: %v", err)
	}
	if len(converted) > int(^uint32(0)) {
		return fmt.Errorf("text too long: %d bytes > %d bytes", len(converted), ^uint32(0))
	}

	var buf [8]byte
	buf[0] = 6
	bo.PutUint32(buf[4:], uint32(len(converted)))
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	if _, err := w.Write(converted); err != nil {
		return err
	}
	return nil
}

type FramebufferUpdateMessage struct {
	Rectangles []*FramebufferUpdateRect
}

type FramebufferUpdateRect struct {
	X            uint16
	Y            uint16
	Width        uint16
	Height       uint16
	EncodingType uint32 // Unsigned per spec, but often interpreted signed
	PixelData    []byte
}

func (m *FramebufferUpdateMessage) Read(r io.Reader, bo binary.ByteOrder, pixelFormat PixelFormat) error {
	var buf [4]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	if buf[0] != 0 {
		return fmt.Errorf("expected message type 0, but found %d", buf[0])
	}
	count := bo.Uint16(buf[2:])
	m.Rectangles = nil
	for i := uint16(0); i < count; i++ {
		rect := &FramebufferUpdateRect{}
		if err := rect.Read(r, bo, pixelFormat); err != nil {
			return err
		}
		m.Rectangles = append(m.Rectangles, rect)
	}
	return nil
}

func (m *FramebufferUpdateMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [4]byte
	buf[0] = 0
	bo.PutUint16(buf[2:], uint16(len(m.Rectangles)))
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	for _, rect := range m.Rectangles {
		if err := rect.Write(w, bo); err != nil {
			return err
		}
	}
	return nil
}

func (rect *FramebufferUpdateRect) Read(r io.Reader, bo binary.ByteOrder, pixelFormat PixelFormat) error {
	var buf [12]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	rect.X = bo.Uint16(buf[0:])
	rect.Y = bo.Uint16(buf[2:])
	rect.Width = bo.Uint16(buf[4:])
	rect.Height = bo.Uint16(buf[6:])
	rect.EncodingType = bo.Uint32(buf[8:])
	if rect.EncodingType != 0 {
		// TODO: Allow caller to provide additional decoders.
		return fmt.Errorf("only raw encoding is supported, but found %d", rect.EncodingType)
	}
	rect.PixelData = make([]byte, int(pixelFormat.BitsPerPixel/8)*int(rect.Width)*int(rect.Height))
	if _, err := io.ReadFull(r, rect.PixelData); err != nil {
		return err
	}
	return nil
}

func (rect *FramebufferUpdateRect) Write(w io.Writer, bo binary.ByteOrder) error {
	var buf [12]byte
	bo.PutUint16(buf[0:], rect.X)
	bo.PutUint16(buf[2:], rect.Y)
	bo.PutUint16(buf[4:], rect.Width)
	bo.PutUint16(buf[6:], rect.Height)
	bo.PutUint32(buf[8:], uint32(rect.EncodingType))
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	if _, err := w.Write(rect.PixelData); err != nil {
		return err
	}
	return nil
}

type BellMessage struct{}

func (m *BellMessage) Read(r io.Reader) error {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	if buf[0] != 2 {
		return fmt.Errorf("expected message type 2, but found %d", buf[0])
	}
	return nil
}

func (m *BellMessage) Write(w io.Writer) error {
	_, err := w.Write([]byte{2})
	return err
}

type ServerCutTextMessage struct {
	Text string
}

func (m *ServerCutTextMessage) Read(r io.Reader, bo binary.ByteOrder) error {
	var buf [255]byte
	if _, err := io.ReadFull(r, buf[:8]); err != nil {
		return err
	}
	if buf[0] != 3 {
		return fmt.Errorf("expected message type 6, but found %d", buf[0])
	}
	textLength := bo.Uint32(buf[4:])
	if int(textLength) > len(buf) {
		return fmt.Errorf("text length too long: %d > %d", textLength, len(buf))
	}
	if _, err := io.ReadFull(r, buf[:textLength]); err != nil {
		return err
	}
	converted, err := charmap.ISO8859_1.NewDecoder().Bytes(buf[:textLength])
	if err != nil {
		return fmt.Errorf("couldn't convert text to UTF-8 in ClientCutText: %v", err)
	}
	m.Text = string(converted)
	return nil
}

func (m *ServerCutTextMessage) Write(w io.Writer, bo binary.ByteOrder) error {
	converted, err := charmap.ISO8859_1.NewEncoder().Bytes([]byte(m.Text))
	if err != nil {
		return fmt.Errorf("encode text: %v", err)
	}
	if len(converted) > int(^uint32(0)) {
		return fmt.Errorf("text too long: %d bytes > %d bytes", len(converted), ^uint32(0))
	}

	var buf [8]byte
	buf[0] = 3
	bo.PutUint32(buf[4:], uint32(len(converted)))
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}
	if _, err := w.Write(converted); err != nil {
		return err
	}
	return nil
}

type PixelFormat struct {
	BitsPerPixel uint8
	BitDepth     uint8
	BigEndian    bool

	// RGB definitions below are used if true.
	// If false, palette mode is used, which is unsupported by this library.
	TrueColor bool

	RedMax     uint16
	GreenMax   uint16
	BlueMax    uint16
	RedShift   uint8
	GreenShift uint8
	BlueShift  uint8
}

// buf must contain at least 16 bytes.
func (pf *PixelFormat) Read(buf []byte, bo binary.ByteOrder) {
	pf.BitsPerPixel = buf[0]
	pf.BitDepth = buf[1]
	pf.BigEndian = buf[2] != 0
	pf.TrueColor = buf[3] != 0

	pf.RedMax = bo.Uint16(buf[4:])
	pf.GreenMax = bo.Uint16(buf[6:])
	pf.BlueMax = bo.Uint16(buf[8:])
	pf.RedShift = buf[10]
	pf.GreenShift = buf[11]
	pf.BlueShift = buf[12]
}

// buf must contain at least 16 bytes.
func (pf *PixelFormat) Write(buf []byte, bo binary.ByteOrder) {
	buf[0] = pf.BitsPerPixel
	buf[1] = pf.BitDepth
	if pf.BigEndian {
		buf[2] = 1
	} else {
		buf[2] = 0
	}
	if pf.TrueColor {
		buf[3] = 1
	} else {
		buf[3] = 0
	}
	bo.PutUint16(buf[4:], pf.RedMax)
	bo.PutUint16(buf[6:], pf.GreenMax)
	bo.PutUint16(buf[8:], pf.BlueMax)
	buf[10] = pf.RedShift
	buf[11] = pf.GreenShift
	buf[12] = pf.BlueShift
}
