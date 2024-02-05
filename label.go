package graphics

import (
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/quasilyte/gmath"
	"golang.org/x/image/font"
)

type AlignVertical uint8

const (
	AlignVerticalTop AlignVertical = iota
	AlignVerticalCenter
	AlignVerticalBottom
)

type AlignHorizontal uint8

const (
	AlignHorizontalLeft AlignHorizontal = iota
	AlignHorizontalCenter
	AlignHorizontalRight
)

type GrowVertical uint8

const (
	GrowVerticalDown GrowVertical = iota
	GrowVerticalUp
	GrowVerticalBoth
	GrowVerticalNone
)

type GrowHorizontal uint8

const (
	GrowHorizontalRight GrowHorizontal = iota
	GrowHorizontalLeft
	GrowHorizontalBoth
	GrowHorizontalNone
)

// Label is a simple text rendering object.
//
// It supports different kinds of grow/aling settings.
// The text color can be changed, but only for the whole text.
//
// Label implements gscene Graphics interface.
type Label struct {
	colorScale       ColorScale
	ebitenColorScale ebiten.ColorScale

	text string

	Pos gmath.Pos

	cache *Cache

	flags        labelFlag
	fontID       uint16
	width        uint16
	height       uint16
	boundsWidth  uint16
	boundsHeight uint16
}

type labelFlag uint16

const (
	// bit0
	labelFlagVisible labelFlag = 1 << iota
	// bit1, bit2
	labelFlagAlignVerticalBit1
	labelFlagAlignVerticalBit2
	// bit3, bit4
	labelFlagAlignHorizontalBit1
	labelFlagAlignHorizontalBit2
	// bit5, bit6
	labelFlagGrowHorizontalBit1
	labelFlagGrowHorizontalBit2
	// bit7, bit8
	labelFlagGrowVerticalBit1
	labelFlagGrowVerticalBit2
	// bit9
	labelFlagDisposed
)

func NewLabel(cache *Cache, ff font.Face) *Label {
	fontID := cache.internFontFace(ff)
	return &Label{
		cache:  cache,
		fontID: fontID,
		flags:  labelFlagVisible,
	}
}

// GetColorScale is used to retrieve the current color scale value of the label's text.
// Use SetColorScale to change it.
func (l *Label) GetColorScale() ColorScale {
	return l.colorScale
}

// SetColorScale assigns a new ColorScale to this label's text.
// Use GetColorScale to retrieve the current color scale.
func (l *Label) SetColorScale(cs ColorScale) {
	if l.colorScale == cs {
		return
	}
	l.colorScale = cs
	l.ebitenColorScale = l.colorScale.toEbitenColorScale()
}

// GetAlpha is a shorthand for GetColorScale().A expression.
// It's mostly provided for a symmetry with SetAlpha.
func (l *Label) GetAlpha() float32 { return l.colorScale.A }

// SetAlpha is a convenient way to change the alpha value of the ColorScale.
func (l *Label) SetAlpha(a float32) {
	if l.colorScale.A == a {
		return
	}
	l.colorScale.A = a
	l.ebitenColorScale = l.colorScale.toEbitenColorScale()
}

func (l *Label) Dispose() {
	l.flags |= labelFlagDisposed
}

func (l *Label) IsDisposed() bool {
	return l.flags&labelFlagDisposed != 0
}

func (l *Label) GetSize() (w, h int) {
	return int(l.width), int(l.height)
}

func (l *Label) SetSize(w, h int) {
	l.width = uint16(w)
	l.height = uint16(h)
}

func (l *Label) GetAlignVertical() AlignVertical {
	return AlignVertical((l.flags >> 1) & 0b11)
}

func (l *Label) SetAlignVertical(a AlignVertical) {
	l.flags &^= labelFlagAlignVerticalBit1 | labelFlagAlignVerticalBit2
	l.flags |= labelFlag(a&0b11) << 2
}

func (l *Label) GetAlignHorizontal() AlignHorizontal {
	return AlignHorizontal((l.flags >> 3) & 0b11)
}

func (l *Label) SetAlignHorizontal(a AlignHorizontal) {
	l.flags &^= labelFlagAlignHorizontalBit1 | labelFlagAlignHorizontalBit2
	l.flags |= labelFlag(a&0b11) << 4
}

func (l *Label) GetGrowVertical() GrowVertical {
	return GrowVertical((l.flags >> 5) & 0b11)
}

func (l *Label) SetGrowVertical(g GrowVertical) {
	l.flags &^= labelFlagGrowVerticalBit1 | labelFlagGrowVerticalBit2
	l.flags |= labelFlag(g&0b11) << 6
}

func (l *Label) GetGrowHorizontal() GrowHorizontal {
	return GrowHorizontal((l.flags >> 7) & 0b11)
}

func (l *Label) SetGrowHorizontal(g GrowHorizontal) {
	l.flags &^= labelFlagGrowHorizontalBit1 | labelFlagGrowHorizontalBit2
	l.flags |= labelFlag(g&0b11) << 8
}

func (l *Label) IsVisible() bool {
	return l.flags&labelFlagVisible != 0
}

// SetVisibility changes the Visible flag value.
// It can be used to show or hide the label.
// Use IsVisible to get the current flag value.
func (l *Label) SetVisibility(visible bool) {
	l.flags |= labelFlagVisible
}

func (l *Label) SetText(s string) {
	l.text = s

	fontInfo := l.cache.fontInfoList[l.fontID]

	bounds := text.BoundString(fontInfo.ff, l.text)
	l.boundsWidth = uint16(bounds.Dx())
	l.boundsHeight = uint16(bounds.Dy())
}

func (l *Label) BoundsRect() gmath.Rect {
	return l.containerRect(l.Pos.Resolve())
}

func (l *Label) Draw(screen *ebiten.Image) {
	l.DrawWithOffset(screen, gmath.Vec{})
}

func (l *Label) DrawWithOffset(screen *ebiten.Image, offset gmath.Vec) {
	if !l.IsVisible() || l.text == "" {
		return
	}

	pos := l.Pos.Resolve()

	fontInfo := l.cache.fontInfoList[l.fontID]

	// Adjust the pos, since "dot position" (baseline) is not a top-left corner.
	pos.Y += fontInfo.capHeight

	numLines := strings.Count(l.text, "\n") + 1

	containerRect := l.containerRect(pos)

	switch l.GetAlignVertical() {
	case AlignVerticalTop:
		// Do nothing.
	case AlignVerticalCenter:
		pos.Y += (containerRect.Height() - l.estimateHeight(numLines)) / 2
	case AlignVerticalBottom:
		pos.Y += containerRect.Height() - l.estimateHeight(numLines)
	}

	var drawOptions ebiten.DrawImageOptions
	drawOptions.ColorScale = l.ebitenColorScale
	drawOptions.Filter = ebiten.FilterLinear

	if l.GetAlignHorizontal() == AlignHorizontalLeft {
		drawOptions.GeoM.Translate(math.Round(pos.X), math.Round(pos.Y))
		drawOptions.GeoM.Translate(offset.X, offset.Y)
		text.DrawWithOptions(screen, l.text, fontInfo.ff, &drawOptions)
		return
	}

	textRemaining := l.text
	offsetY := 0.0
	for {
		nextLine := strings.IndexByte(textRemaining, '\n')
		lineText := textRemaining
		if nextLine != -1 {
			lineText = textRemaining[:nextLine]
			textRemaining = textRemaining[nextLine+len("\n"):]
		}
		lineBounds := text.BoundString(fontInfo.ff, lineText)
		lineBoundsWidth := float64(lineBounds.Dx())
		offsetX := 0.0
		switch l.GetAlignHorizontal() {
		case AlignHorizontalCenter:
			offsetX = (containerRect.Width() - lineBoundsWidth) / 2
		case AlignHorizontalRight:
			offsetX = containerRect.Width() - lineBoundsWidth
		}
		drawOptions.GeoM.Reset()
		drawOptions.GeoM.Translate(math.Round(pos.X+offsetX), math.Round(pos.Y+offsetY))
		drawOptions.GeoM.Translate(offset.X, offset.Y)
		text.DrawWithOptions(screen, lineText, fontInfo.ff, &drawOptions)
		if nextLine == -1 {
			break
		}
		offsetY += fontInfo.lineHeight
	}
}

func (l *Label) containerRect(pos gmath.Vec) gmath.Rect {
	var containerRect gmath.Rect

	boundsWidth := float64(l.boundsWidth)
	boundsHeight := float64(l.boundsHeight)
	fwidth := float64(l.width)
	fheight := float64(l.height)

	if l.width == 0 && l.height == 0 {
		// Auto-sized container.
		switch l.GetGrowHorizontal() {
		case GrowHorizontalRight:
			containerRect.Min.X = pos.X
			containerRect.Max.X = pos.X + boundsWidth
		case GrowHorizontalLeft:
			containerRect.Min.X = pos.X - boundsWidth
			containerRect.Max.X = pos.X
			pos.X -= boundsWidth
		case GrowHorizontalBoth:
			containerRect.Min.X = pos.X - boundsWidth/2
			containerRect.Max.X = pos.X + boundsWidth/2
			pos.X -= boundsWidth / 2
		}
		switch l.GetGrowVertical() {
		case GrowVerticalDown:
			containerRect.Min.Y = pos.Y
			containerRect.Max.Y = pos.Y + boundsHeight
		case GrowVerticalUp:
			containerRect.Min.Y = pos.Y - boundsHeight
			containerRect.Max.Y = pos.Y
			pos.Y -= boundsHeight
		case GrowVerticalBoth:
			containerRect.Min.Y = pos.Y - boundsHeight/2
			containerRect.Max.Y = pos.Y + boundsHeight/2
			pos.Y -= boundsHeight / 2
		}
	} else {
		containerRect = gmath.Rect{
			Min: pos,
			Max: pos.Add(gmath.Vec{X: fwidth, Y: fheight}),
		}
		if delta := boundsWidth - fwidth; delta > 0 {
			switch l.GetGrowHorizontal() {
			case GrowHorizontalRight:
				containerRect.Max.X += delta
			case GrowHorizontalLeft:
				containerRect.Min.X -= delta
			case GrowHorizontalBoth:
				containerRect.Min.X -= delta / 2
				containerRect.Max.X += delta / 2
			case GrowHorizontalNone:
				// Do nothing.
			}
		}
		if delta := boundsHeight - fheight; delta > 0 {
			switch l.GetGrowVertical() {
			case GrowVerticalDown:
				containerRect.Min.Y += delta
			case GrowVerticalUp:
				containerRect.Min.Y -= delta
				pos.Y -= delta
			case GrowVerticalBoth:
				containerRect.Min.Y -= delta / 2
				containerRect.Max.Y += delta / 2
				pos.Y -= delta / 2
			case GrowVerticalNone:
				// Do nothing.
			}
		}
	}

	return containerRect
}

func (l *Label) estimateHeight(numLines int) float64 {
	fontInfo := l.cache.fontInfoList[l.fontID]
	estimatedHeight := fontInfo.capHeight
	if numLines >= 2 {
		estimatedHeight += (float64(numLines) - 1) * fontInfo.lineHeight
	}
	return estimatedHeight
}
