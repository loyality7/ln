# ln/rig

Characters and motion as data. A skeleton is a `Spec` (joints and limbs), a pose is a map of named channel values, a move is a `Clip` built from keyframes or oscillators, and an `Actor` plays clips on a `stage.Timeline`, blending between them. Nothing about any particular character lives here (`components/stickman` supplies one). Imports `anim`, `geom`, `ink`, `model`, `solid`, `stage`. Source: `ln/rig/*.go`.

## Channels and poses (`pose.go`)

`type Pose map[string]float64`. Channels not mentioned are at rest (0). Channel names:

| Channel | Meaning |
|---|---|
| `"knee"` (a joint name, no dot) | Angle of that joint about its axis, radians |
| `"pelvis.z"` (joint name + `.x`/`.y`/`.z`) | Offset added to the joint's position along that axis |
| `"root.x"` | Forward travel of the whole skeleton |
| `"root.z"` | Height of the whole skeleton |
| `"root.yaw"` | Turn of the whole skeleton (an angle, radians) |

Functions: `Deg(p Pose) Pose` converts rotation channels from degrees to radians (distance channels, which contain a dot, are left alone; `root.yaw` counts as a rotation). `(Pose).With(over)` returns a copy with channels replaced. `(Pose).Mix(b, t)` blends channel by channel (missing channels count as 0). `Smooth(t)` is `t*t*(3-2t)` with `t` clamped to 0..1.

## Skeleton (`skeleton.go`)

```go
type Axis int   // AxisY (the zero value), AxisX, AxisZ: the local axis a joint turns about
type Limb struct { Radius, Length float64; Up bool }          // capsule from the joint: down (-z), or up with Up
type Part struct { Mesh *model.Mesh; At, Turn geom.Vector; Ink *ink.Material }
type Joint struct {
	Name, Parent string
	At           geom.Vector // offset from the parent
	Axis         Axis
	Sign         float64     // -1 flips the direction a positive angle turns (0 means 1)
	Limb         *Limb
	Parts        []Part
	Track        bool        // effects such as smear arcs follow this joint's tip
	Tip          *geom.Vector // tracked point in the joint's space (default: end of the limb)
}
type Spec struct { Height float64; Ink ink.Material; Joints []Joint }
```

- `Spec.Height` is the height the dimensions were drawn at; `Spec.Ink` is the default ink for limbs and parts (the zero value means solid black, `ink.Material{Class: ink.Black}`).
- Joints must be listed parents first; a missing parent panics (`rig: joint "x" has parent "y" which is not defined before it`).

`Build(parent *stage.Node, spec Spec, height, origin, facing float64) *Skeleton` creates one stage node per joint under `parent`, uniformly scaled by `height / spec.Height`, standing at `x = origin` and facing +x (`facing` 1) or -x (-1). Limbs become capsules (`solid.Capsule(radius, length - 2*radius)`) with no outline; parts are added with their own offsets and ink.

`Skeleton` has `Root *stage.Node`, `Origin, Facing float64`, and methods:

| Method | Meaning |
|---|---|
| `Joint(name) *stage.Node` | The joint's node; panics if unknown |
| `Apply(p Pose)` | Resets every joint to rest, then applies the pose. A channel naming no joint panics (`rig: pose channel "x" names no joint`). The root is placed at `Origin + Facing*root.x`, height `root.z`, yaw `root.yaw` (plus pi when facing -1) |
| `Tips() []geom.Vector` | World positions of the tracked joints' tips (hands, feet, head), for smear arcs |

## Clips (`clip.go`)

```go
type Key struct { T float64; Pose Pose; Ease anim.Ease }   // T is progress 0..1; Ease shapes the move into this key (default Smooth)
func K(t float64, degrees Pose) Key                       // a key whose pose is in degrees
func Keys(dur float64, keys ...Key) Clip                  // poses at moments, in order
func Cycle(dur, cycles float64, base Pose, osc ...Osc) Clip   // base pose (degrees) plus oscillators over `cycles` repetitions
func Freeze(dur float64) Clip                             // hold the pose the actor is in when it starts
```

`Osc{Ch string; Amp, Phase float64; Mult float64; Wave Wave}`: value is `Amp * wave(2*pi*(Mult*cycles*progress + Phase))`. `Amp` is in degrees for rotation channels; `Phase` is in cycles (0.25 turns sine into cosine); `Mult` 0 means 1. `Wave` is `Sine` (both ways), `Half` (positive half only) or `Abs` (always positive, twice per cycle).

`Clip{Dur, Dist float64; Marks []Mark; ...}` is shaped with chainable methods that return a modified copy:

| Method | Meaning |
|---|---|
| `Arc(height, from, to)` | Adds a parabola `height*4q(1-q)` to `root.z` between two progress values (a jump) |
| `Spin(ch string, turns, from, to)` | Turns a channel through `turns` full rotations between two moments, eased with `Smooth`. After `to`, only the fractional part of `turns` remains. A flip is a spin of the pelvis, a pirouette a spin of `root.yaw` |
| `Travel(dist, from, to)` | Moves forward by `dist` (negative is back) between two moments, **linearly**; with no `Travel` it stays put. Sets `Dist` |
| `Mark(m Mark)` | Adds an effect event |

`Mark{Kind, At, Span, Secs, X, Z, AtEnd, Size}`: `Kind` is `"dust"`, `"impact"`, `"trail"` or `"speed"`; `At` is the progress when it starts; its length is `Span` (a fraction of the clip) or, if `Span` is 0, `Secs`; `X` is the forward distance from where the clip starts (from where it ends if `AtEnd`), `Z` the height; `Size` scales impact stars (0 = normal).

## Actor (`actor.go`)

```go
func NewActor(tl *stage.Timeline, skel *Skeleton) *Actor   // Blend default 0.18 s
type Actor struct { Skel *Skeleton; TL *stage.Timeline; Blend float64; Events []Event /* + position and last pose */ }
type Event struct { Kind string; T, Dur, X, Z, Size float64; Who *Actor }
```

| Method | Meaning |
|---|---|
| `X() float64` | How far the actor will have travelled after the moves scheduled so far |
| `Play(c Clip) float64` | Schedules the clip at the timeline cursor (as a `Timeline.Do`), blending from the previous pose over `Blend` seconds; records the clip's marks as `Event`s in world X (`Origin + Facing*x`); returns the start time |
| `Hold(dur) float64` | `Play(Freeze(dur))` |

Choreography:

| Function | Meaning |
|---|---|
| `Moves(a *Actor, clips ...Clip) Lane` | One actor's clips, in order |
| `Beat(tl *stage.Timeline, lanes ...Lane)` | Plays lanes at the same time; the next beat starts when the slowest lane finishes |
| `Gap(a, b *Actor) float64` | Distance between two actors facing each other |
| `Close(tl, a, b, gap float64, dash func(dist float64) Clip)` | Brings both actors together until `gap` apart, each travelling half the distance with `dash(dist)` |

## Example

Shape of real use (see `components/stickman` for the skeleton and `examples/engine/stickman`):

```go
sk := rig.Build(st.Root, spec, 1.8, 0, 1)
a := rig.NewActor(st.TL, sk)
a.Play(rig.Cycle(1.2, 2, rig.Pose{}, rig.Osc{Ch: "knee", Amp: 30}).Travel(2, 0, 1))
a.Hold(0.5)
```
(`spec` is a `rig.Spec` you define; `knee` must be a joint name in it.)
