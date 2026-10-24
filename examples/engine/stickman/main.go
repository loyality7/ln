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

// Everything the stick man can do, one move after another with a caption:
//
//	go run ./examples/engine/stickman -w 520 -h 292 -fps 12 -fmt gif -out assets/stickman
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
		s.Root.Box(260, 60, 0.2).At(60, 0, -0.1).NoOutline().Ink(ink.Material{Class: ink.Paper, BlackAt: 0.55})
		for x := -6.0; x < 110; x += 1.6 {
			s.Root.Box(0.7, 0.07, 0.02).At(x, 1.4, 0.012).NoOutline().Ink(ink.Material{Class: ink.Black})
		}

		hero := rig.NewActor(s.TL, sm.New(s.Root, 1.8, 0, 1, sm.Look{}))
		show := []struct {
			name string
			move rig.Clip
		}{
			{"", sm.Idle(0.5)},
			{"WAVE", sm.Wave(1.5)},
			{"WALK", sm.Walk(2.2, 2.0)},
			{"RUN", sm.Run(1.8, 6)},
			{"JUMP", sm.Jump(0.9, 1.1, 1.6)},
			{"FRONT FLIP", sm.Flip(1.1, +1, 1.5, 2.6)},
			{"", sm.Idle(0.25)},
			{"BACK FLIP", sm.Flip(1.1, -1, 1.5, -2.2)},
			{"DANCE", sm.Dance(2.0)},
			{"GUARD", sm.Guard(0.9)},
			{"PUNCH", sm.Punch(0.9, true)},
			{"UPPERCUT", sm.Uppercut(0.9, true)},
			{"KICK", sm.Kick(1.0, true)},
			{"SPIN KICK", sm.SpinKick(1.1, true)},
			{"FLYING KICK", sm.FlyingKick(1.2, 1.6, true)},
			{"BLOCK", sm.Block(0.9, 0.2)},
			{"DUCK", sm.Duck(0.9)},
			{"EVADE", sm.Evade(0.8, 1.3)},
			{"HIT", sm.Hit(0.7, 0.8)},
			{"KNOCKOUT", sm.KO(1.4, 2.0)},
			{"", sm.Idle(0.4)},
			{"GET UP", sm.GetUp(1.0)},
			{"VICTORY", sm.Victory(1.3)},
			{"BOW", sm.Bow(1.2)},
			{"", sm.Idle(0.4)},
		}
		for _, m := range show {
			start := hero.Play(m.move)
			if m.name != "" {
				fx.Caption(s.Stage, start, m.move.Dur, m.name)
			}
		}
		comicfx.Install(s.Stage, hero)
	})
}
