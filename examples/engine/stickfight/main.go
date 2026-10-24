package main

import (
	"github.com/loyality7/ln/components/comicfx"
	sm "github.com/loyality7/ln/components/stickman"
	. "github.com/loyality7/ln/components/studio"
	"github.com/loyality7/ln/ln/fx"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/ink"
	"github.com/loyality7/ln/ln/light"
	"github.com/loyality7/ln/ln/rig"
	"github.com/loyality7/ln/ln/stage"
)

// Two stick men fight: blocks, dodges, an uppercut, a flying kick, a spin kick,
// a double flip, a clash and a knockout. A has a ring head; B a solid head and scarf.
//
//	go run ./examples/engine/stickfight -w 520 -h 292 -fps 12 -fmt gif -out assets/stickfight
func main() {
	Run(func(s *Scene) {
		k := s.W / 960 // line weights are in pixels, so scale them with the picture
		style := ink.ToonStyle()
		style.Outline, style.BlackAt, style.Wobble = 3.4*k, 0.55, 0.45*k
		s.Stage.Look = &stage.ToonLook{Style: style, BoilFPS: 12}
		s.Atmosphere = stage.Studio{Lights: light.Rig{
			Lights:  []light.Light{light.Directional{Dir: geom.V(-0.75, -0.45, 0.5), Power: 1}},
			Ambient: 0.12, Shadows: true,
		}}

		// the floor, with marks to read the motion against
		s.Root.Box(260, 60, 0.2).At(0, 0, -0.1).NoOutline().Ink(ink.Material{Class: ink.Paper, BlackAt: 0.55})
		for x := -14.0; x < 16; x += 1.6 {
			s.Root.Box(0.7, 0.07, 0.02).At(x, 1.4, 0.012).NoOutline().Ink(ink.Material{Class: ink.Black})
		}

		A := rig.NewActor(s.TL, sm.New(s.Root, 1.8, 0, 1, sm.Look{}))
		B := rig.NewActor(s.TL, sm.New(s.Root, 1.8, 3.4, -1, sm.Look{SolidHead: true, Scarf: true}))
		beat := func(a, b rig.Lane) { rig.Beat(s.TL, a, b) }
		plays := rig.Moves
		closeTo := func(dur, gap float64) {
			rig.Close(s.TL, A, B, gap, func(d float64) rig.Clip { return sm.Dash(dur, d) })
		}
		title := func(text string, dur float64) { fx.Caption(s.Stage, s.TL.Cursor(), dur, text) }

		beat(plays(A, sm.Guard(1.0)), plays(B, sm.Guard(1.0)))
		closeTo(0.7, 1.9)
		beat(plays(A, sm.Guard(0.6)), plays(B, sm.Guard(0.6)))
		closeTo(0.45, 1.35)

		title("PUNCH AND BLOCK", 1.0)
		beat(plays(A, sm.Punch(0.9, true)), plays(B, sm.Guard(0.38), sm.Block(0.62, 0.15)))
		title("KICK AND DUCK", 1.15)
		beat(plays(A, sm.Guard(0.2), sm.Duck(0.95)), plays(B, sm.Kick(1.15, false)))
		title("UPPERCUT", 0.96)
		beat(plays(A, sm.Uppercut(0.9, true)), plays(B, sm.Guard(0.36), sm.Hit(0.6, 0.8)))

		beat(plays(A, sm.Guard(0.7)), plays(B, sm.Evade(0.8, 1.2)))
		g := rig.Gap(A, B)
		title("FLYING KICK", 1.48)
		beat(plays(A, sm.FlyingKick(1.25, g-1.0, true)), plays(B, sm.Guard(0.78), sm.Hit(0.7, 1.1)))

		beat(plays(A, sm.Guard(0.7)), plays(B, sm.Guard(0.7)))
		closeTo(0.4, 1.5)
		title("SPIN KICK", 1.2)
		beat(plays(A, sm.Guard(0.55), sm.Hit(0.65, 1.0)), plays(B, sm.SpinKick(1.1, true)))

		title("DOUBLE FLIP", 1.1)
		beat(plays(A, sm.Flip(1.1, -1, 1.4, -1.7)), plays(B, sm.Flip(1.1, +1, 1.4, 1.7)))
		closeTo(0.4, 1.3)
		title("CLASH", 1.5)
		beat(plays(A, sm.Punch(0.9, true)), plays(B, sm.Punch(0.9, true)))
		beat(plays(A, sm.Hit(0.6, 0.9)), plays(B, sm.Hit(0.6, 0.9)))
		closeTo(0.5, 1.5)

		title("KNOCKOUT", 2.05)
		beat(plays(A, sm.SpinKick(1.1, true)), plays(B, sm.Guard(0.55), sm.KO(1.5, 2.8)))
		beat(plays(A, sm.Guard(0.4), sm.Victory(1.9)), plays(B, rig.Freeze(2.3)))

		comicfx.Install(s.Stage, A, B)
	})
}
