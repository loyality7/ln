# ln/ink

Everything that turns a scene into ink: the hatched pen-and-ink renderer, the clean manga ("toon") renderer, the vector `Drawing` they produce, contour tools, and style presets. Imports `ln/buffer`, `ln/geom`, `ln/light`, `ln/scene`. Source: `ln/ink/*.go`.

## Drawing (`drawing.go`)

A `*Drawing` is vector art in page pixels (origin bottom-left, Y up): ordered layers of fills and lines.

```go
type Line struct {
	Pts   geom.Path
	Width float64   // scaled by eye distance unless W is set
	W     []float64 // per-vertex width in px (brush pressure); wins over Width
	Hatch bool      // shading stroke: tapered, drawn before outlines in the legacy renderer
	White bool      // paper-white ink, for line art on a black ground
}
type Fill  struct { Loops []geom.Path; White bool }  // all loops filled together, even-odd rule: an inner loop cuts a hole
type Layer struct { Fills []Fill; Lines []Line }     // fills first, then lines
type Drawing struct { W, H float64; Layers []Layer /* + unexported width function */ }
```

| Function / method | Meaning |
|---|---|
| `NewDrawing(w, h float64, layers ...Layer) *Drawing` | Build a drawing yourself (logo, diagram, `components/blackhole`). Lines should carry `W`; distance scaling is off |
| `(*Drawing).Image(_, _ float64) image.Image` | Rasterises with `gg`: white paper, ink colour `(0.04, 0.04, 0.06)`, round caps. The arguments are ignored, so a `*Drawing` satisfies `anim.Frame`. Segments are bucketed by width (quarter-pixel steps) and colour for speed; widths below 0.125 px vanish |
| `(*Drawing).SVG() string` | Same content as SVG (polylines and even-odd paths) |
| `(*Drawing).Reveal(p float64) *Drawing` | The drawing as if the pen had covered fraction `p` of the total stroke length: outlines first, then hatching. Fills appear once `p >= 0.6` |

## Rendering a scene

| Function | Output |
|---|---|
| `Render(s *scene.Scene, eye, center, up geom.Vector, width, height, fovy float64, st Style) *Drawing` | The hatched look: optional sky layer (`st.Background`), hatch lines from the buffers, then outlines from `Scene.RenderDepth` |
| `RenderToon(s, eye, center, up, width, height, fovy, st Style, mats map[int]Material, overlays []Overlay) *Drawing` | The manga look: backdrop overlays, paper base, mid-tone hatching, under-black overlays, solid black regions as smooth vector fills, sky marks, over-black overlays, then per-object contours and crease lines |

Both compile the scene, fill in automatic style values, gather buffers at `Scale` times the output size, and finish with a hand-drawn wobble (`Wobble`, `WobbleSeed`, `WobbleScale`). `stage.InkLook` and `stage.ToonLook` call these.

### Style (`style.go`, `presets.go`)

`DefaultStyle()` (the "ink" look): Light `(-0.5,-0.7,1)`, three hatch families (thresholds 0.42, 0.68, 0.88), shadows on, Scale 0.5, Outline 2.6, Hatch 1.1, NearMul 1.5, FarMul 0.55, Wobble 1.3. `ToonStyle()` starts from it with Scale 1, Outline 2.4, Hatch 0.8, Wobble 0.5, BlackAt 0.8, DarkBoost 1.1, NearMul 1.2, FarMul 0.8, two hatch families (0.55, 0.8).

`Style` fields (zero means "automatic" where noted):

| Field | Meaning |
|---|---|
| `Light geom.Vector` | Direction from the scene toward the light (normalised in use) |
| `Spacing` | World distance between hatch lines (0 = eye-to-centre distance / 90) |
| `Thresholds []float64`, `Dirs []geom.Vector` | Darkness at which hatch family *i* switches on; its plane normal |
| `Shadows bool` | Cast shadow rays |
| `Scale` | Buffer resolution relative to output (0 = 0.5) |
| `Outline`, `Hatch` | Stroke widths in px |
| `NearMul`, `FarMul`, `NearDist`, `FarDist` | Width multiplier at the near and far distances (0 distances = 0.6x and 2x the eye-centre distance) |
| `Wobble`, `WobbleSeed`, `WobbleScale` | Jitter amplitude px; change the seed per frame for boiling lines; wave tightness (0 = 0.045) |
| `Background Background` | Sky; nil = plain paper |
| `Rig *light.Rig` | Lighting; nil = one sun from `Light` with ambient |
| `MinGap` | Thinnest hatch spacing in buffer pixels (0 = 3) |
| `TaperMin` | Thinnest end of a tapered stroke as a fraction (0 = 0.3; 1 = no taper) |
| `Smooth` | Rounding passes on filled shapes (0 = 2); lines use one fewer |
| `SkyBlackAt`, `SkyBandFrom`, `SkyBandStep` | Sky darkness for solid black (0 = 0.62); where pen-line bands start (0 = 0.18); band spacing in buffer rows (0 = 5) |
| `BlackAt` | Toon: shadow darkness at which paper turns solid black (0 = never) |
| `DarkBoost` | Toon: extra line weight on the shadow side |

`Preset(name string) (Style, bool)` and `PresetNames() []string` (sorted): `"ink"` (default), `"manga"` (toon), `"sketch"` (Wobble 2.2, WobbleScale 0.03, Outline 2, Hatch 1, TaperMin 0.15), `"technical"` (no wobble, no taper, Outline 1.4, Hatch 0.7, NearMul = FarMul = 1).

### Materials and overlays (`toon.go`)

```go
type Class uint8     // Paper (white; light decides), Black (always solid), White (never shaded)
type Material struct { Class Class; Tone bool; Line bool; LineW float64; BlackAt float64 }
func DefaultMaterial() Material // Paper, Tone, Line, LineW 1
```

`BlackAt` overrides the style threshold for that object (0 keeps the style value, 2 never blackens). Materials are looked up by object tag (a mirror image uses the real object's material).

```go
type Slot int      // UnderBlack, OverBlack, Backdrop
type Overlay struct { Slot Slot; Draw func(c *Ctx) Layer }
```

`UnderBlack` art is hidden where black is painted over it; `OverBlack` cuts highlights out of black; `Backdrop` goes behind everything and the figure gets a paper base over it.

`Ctx` (given to overlays and backgrounds): fields `W, H, Eye`; methods `Project(p) geom.Vector` (page pixels, Z = eye distance), `Visible(p) bool`, `OnSky(dir) (geom.Vector, bool)` (projects a direction and reports whether it shows sky inside the picture), `Focal() float64` (focal length in page pixels).

### Background (`sky.go`)

```go
type Background interface {
	Darkness(dir geom.Vector) float64  // 0 white .. 1 black along dir
	Marks(c *Ctx) Layer                // stars, sun and moon discs
}
```

Sky pixels with darkness at or above `SkyBlackAt` become solid black; those between `SkyBandFrom` and `SkyBlackAt` become horizontal pen lines.

## Contour and curve tools (`vector.go`)

| Function | Meaning |
|---|---|
| `IsoLines(f []float64, valid []bool, w, h int, levels []float64, sx, sy float64) [][]geom.Path` | Marching squares on a `w*h` field (row 0 at the bottom). Returns, for each level (levels must be ascending), the stitched curves where the field equals it. Cells with any invalid corner are skipped, and so are cells whose values differ by more than 4 (treated as a jump between separate surfaces). Points are scaled by `sx, sy` |
| `Curve(p geom.Path, perSpan int) geom.Path` | Catmull-Rom curve through the points, `perSpan` points per pair |
| `Ribbon(center geom.Path, width func(t float64) float64) geom.Path` | Closed outline of a brush stroke whose width follows `width(t)`, `t` 0..1 along the line; fill it with `Fill` |

`IsoLines` is how `components/blackhole` draws rings and spokes. Internally the package also holds `trace` (marching squares for closed outlines), `blur`, `smoothLoop` (Chaikin), `simplify` and `stitch`.

## Example

```go
lines := ink.IsoLines(radius, valid, w, h, []float64{6, 7, 8}, 1, 1)
var layer ink.Layer
for _, path := range lines[0] {
	layer.Lines = append(layer.Lines, ink.Line{Pts: path, W: widths(path), White: true})
}
dr := ink.NewDrawing(float64(w), float64(h), layer)
img := dr.Image(0, 0)
```
(`widths` is your own per-vertex width slice; see `components/blackhole/blackhole.go` for a complete use.)
