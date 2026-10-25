# ln/ascii

A look that draws the scene as text: tone becomes a ramp of characters, and silhouettes, creases and depth edges become `| - / \`. It reads the same buffers as the ink looks. Imports `buffer`, `geom`, `light`, `scene`, plus `gg`, `golang.org/x/image/draw` and `basicfont`. Source: `ln/ascii/ascii.go`.

## Options

```go
type Options struct {
	Cols   int     // characters per row (0 = 140)
	Aspect float64 // picture width / height (0 = 16/9)
	Ramp   string  // lightest to darkest (empty = " .:-=+*#%@")
	Edges  bool    // draw outlines with | - / \
	Invert bool    // light text on a dark page, lit surfaces dense
	Floor  float64 // tones lighter than this stay blank (0 = 0.3)
	Sky    Sky     // interface{ Darkness(dir geom.Vector) float64 }; nil = paper-white sky
}
func (o Options) Rows() int // text rows that keep the aspect: round(Cols*7 / (13*Aspect))
```

A character cell is 7x13 px (`basicfont.Face7x13`). Each character samples a 4x4 block of buffer pixels (`probe = 4`).

## Rendering

| Function | Meaning |
|---|---|
| `Draw(s *scene.Scene, cam buffer.Camera, rig light.Rig, opt Options) *Art` | Gathers a buffer of `Cols*4` wide by `width/Aspect` tall and renders it |
| `Render(g *buffer.Buffers, opt Options) *Art` | Turns existing buffers into characters (use this if you already have them) |

For each cell: tone is the mean of `Dark` over hit pixels (sky pixels use `opt.Sky.Darkness` if given), mapped through `Floor` onto the ramp (flipped if `Invert`). If `Edges` is on and the cell is an edge, the ramp character is replaced by an edge character chosen perpendicular to the depth gradient (`-`, `/`, `|`, `\`). A cell counts as an edge when it contains both hit and non-hit pixels, a depth jump larger than 6% of the distance, or normals that differ (dot below 0.75).

## Art

```go
type Art struct { Cols, Rows int; Cells []rune /* row-major, row 0 at top */; Invert bool }
func (a *Art) String() string                          // the text, one line per row
func (a *Art) Image(width, height float64) image.Image // draws it; satisfies anim.Frame
```

`Image` draws at the natural size `Cols*7 x Rows*13` (ink 0.05 on white, or 0.92 on 0.04 when inverted). Pass 0, 0 to keep that size; any other size rescales bilinearly.

## Example

`stage.NewASCIILook(cols)` wraps this as `Look` with edges on:

```go
st := stage.New(960, 540, 24)
st.Look = stage.NewASCIILook(120)
st.Still("out/ascii.png", 0)
```
