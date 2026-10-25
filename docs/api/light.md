# ln/light

Illumination behind one small interface, so a scene can mix suns, moons, lamps and ambient light without the renderer knowing which. Imports `ln/geom`, `ln/scene`. Source: `ln/light/light.go`.

## Light interface and kinds

```go
type Light interface {
	Toward(p geom.Vector) (dir geom.Vector, dist float64, power float64)
}
```

`Toward` returns the unit direction from `p` to the light, the distance to it (`geom.INF` for directional lights) and its power at `p` (0 up to 1 or more).

| Type | Fields | Behaviour (from the code) |
|---|---|---|
| `Directional` | `Dir geom.Vector` (direction from the scene toward the light), `Power` | Sun or moon: infinite distance, constant power |
| `Point` | `Pos`, `Power`, `Radius`, `Size` | Power falls as `Power / (1 + (d/Radius)^2)`: full at the lamp, half at `Radius`. Shadow rays stop `Size` short of the lamp so it does not shade itself |
| `Spot` | embeds `Point`, plus `Axis geom.Vector`, `Cone float64` | A `Point` that gives zero power outside a cone of half-angle `Cone` (radians) around `Axis`, where `Axis` points the way the light shines |

## Rig

```go
type Rig struct {
	Lights   []Light
	Ambient  float64 // light that reaches everywhere, even in shadow
	Shadows  bool    // trace shadow rays
	Exposure float64 // brightness gain; 0 means 1
}
func Sun(dir geom.Vector) Rig   // one Directional (power 1), Ambient 0.2, Shadows on
func (r Rig) Dark(s *scene.Scene, p, n geom.Vector) float64
```

`Dark` answers "how dark does a surface at `p` with normal `n` look": 0 is fully lit, 1 is black. For each light it takes `max(0, n . dir) * power`; when `Shadows` is on it casts a ray from just above the surface toward the light and skips the light if something is hit nearer than `dist`. The lit amount is `gain * (Ambient + (1 - Ambient) * lit)`, clamped to 1, and `Dark` is `1` minus that. The scene must already be compiled.

## Use

`stage.Atmosphere.Rig(...)` returns a `Rig`; `buffer.Gather` calls `Rig.Dark` for every pixel; `ink` and `ascii` read the resulting darkness.

```go
rig := light.Sun(geom.V(-0.5, -0.6, 1)) // what stage.New uses by default
```
