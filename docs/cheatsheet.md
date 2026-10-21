# Cheat sheet

Everything you need to write a scene. One import:

```go
import . "github.com/loyality7/ln/components/studio"
```

A complete program:

```go
func main() {
	Run(func(s *Scene) {
		s.Look("manga")                       // ink | manga | sketch | technical | ascii
		s.TimeOf("dusk")                      // dawn morning noon afternoon golden dusk night midnight, or "17.5"
		s.Camera(V(0, -12, 2), V(0, 0, 1), 40) // where it stands, what it looks at, field of view

		s.Root.Sphere(1).At(0, 0, 1)          // a ball, 1 up
		s.Root.Box(4, 4, 0.2).At(0, 0, -0.1)  // a floor
	})
}
```

Run it:

```
go run . -still 0      # one picture (out/scene.png)
go run .               # a video, if the scene has motion (out/scene.mp4)
go run . -look sketch -sky night -w 1920 -h 1080 -out out/mine
```

## Shapes (all return a node; chain `.At .Turn .Size .Uniform`)

```go
s.Root.Box(x, y, z)          s.Root.Sphere(r)           s.Root.Cylinder(r, h)
s.Root.Cone(r, h)            s.Root.Torus(big, small)   s.Root.Capsule(r, length)
s.Root.Ellipsoid(rx, ry, rz)
node.At(x, y, z)             // position       node.Turn(rx, ry, rz)  // rotation, radians
node.Size(x, y, z)           // stretch        node.Uniform(k)        // scale
group := s.Root.Group("name").At(...)   // children move with it
group.Sphere(1)                         // add a shape inside a group
```

Anything custom: `s.Root.Add(mesh)` with a mesh from `solid`:

```go
solid.Lathe(profile, 32)        // spin a curve into a vase, bottle, wheel
solid.Extrude(polygon, height)  // any outline lifted into a solid
solid.Tube(path, radiusFn, 16, closed)   // a pipe along a curve
solid.Surface(nu, nv, func(u, v float64) Vec)   // any formula
```

## Land, water, scattering

```go
s.Terrain(func(x, y float64) float64 { return height }, Rect{x0, x1, y0, y1})
s.Water(0, 0.3)                       // mirror lake at height 0; 0.3 = ripple strength
s.GroundAt(x, y)                      // terrain height there
s.Scatter(100, Rect{...}, func(p Vec, r *Rand) {   // p stands on the terrain; r is seeded
	nature.Pine(s.Root, p, 3+r.Float64()*4, r.Int63())
})
```

## Motion (everything on `s.TL`, time in seconds, actions run one after another)

```go
s.TL.Move(node, V(4, 0, 1), 2, anim.OutBounce)   // glide to a point
s.TL.Turn(node, V(0, 0, 3.14), 1, anim.InOutSine)
s.TL.Wait(1)
s.TL.Par(func() { ...actions together... })
s.TL.Orbit(s.Cam.Eye, V(0, 0, 1), 12, 3, 0, 6.28, 8, nil)   // camera circles a point
s.TL.Do(4, func(p float64) { node.Rot.Z = p * 6.28 })       // any custom motion, p = 0..1
s.Day(6, 20, 10)                                            // sunrise to evening over 10 s
```

Easing: `anim.Linear InQuad OutQuad InOutSine OutBack OutElastic OutBounce` (and more).

## Characters and fights (`rig`, `fx`, `components/stickman`)

```go
hero := rig.NewActor(s.TL, stickman.New(s.Root, 1.8, 0, 1, stickman.Look{}))   // origin x, facing +1
hero.Play(stickman.Walk(2.8, 2.6))          // moves play one after another, blended
hero.Play(stickman.Flip(1.1, +1, 1.5, 2.8)) // direction, height, distance
hero.Play(stickman.Punch(0.9, true))        // true = comic star where it lands
rig.Beat(s.TL, rig.Moves(a, stickman.Punch(0.9, true)), rig.Moves(b, stickman.Block(0.6, 0.2))) // together
fx.Install(s.Stage, fx.DefaultFollow(), hero)   // dust, stars, speed lines, smear arcs, shake, camera
```

Moves: Idle Wave Walk Run Jump Flip Dance Fall GetUp Bow Guard Dash Punch Kick Block Duck Evade
Hit Uppercut SpinKick FlyingKick KO Victory. Make your own move by listing poses:

```go
spin := rig.Keys(1.2, rig.K(0, pose{}), rig.K(0.5, pose{"shoulderN": 170}), rig.K(1, pose{})).Spin("root.yaw", 2, 0, 1)
```

## Lights

```go
lamp := s.Root.Sphere(0.2).At(0, 0, 3)
lamp.Emit(1.5, 4)     // power, radius: the sphere now lights its surroundings
```

## Ready-made things (`components/`)

```go
nature.Pine(parent, point, height, seed)    nature.Rock(radius, seed)
nature.FBM(x, y, 4)   nature.Ridged(x, y, 5)   nature.Smoothstep(a, b, x)   // for terrain
mannequin.New(parent, height)   // a posable human figure: .Shoulder .Elbow .Hip .Knee are joints
```

## Looks and pens

`s.Look(name)` picks a preset. To tune: `style, _ := ink.Preset("manga")`, change fields
(`Outline`, `Hatch`, `Wobble`, `TaperMin`, `BlackAt`, ...), and assign it with
`s.Stage.Look = &stage.ToonLook{Style: style}`.
