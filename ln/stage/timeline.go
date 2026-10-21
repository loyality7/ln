package stage

import (
	"math"

	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
)

type Ease = anim.Ease

type prop int

const (
	propPos prop = iota
	propRot
	propScl
)

type trackKey struct {
	n *Node
	p prop
}

type vtrack struct {
	anim.VecTrack
	last  geom.Vector // value after the last key: where the next move starts
	lastT float64
}

type effect struct {
	start, dur float64
	fn         func(p float64)
}

// Timeline scripts motion in seconds. Actions run one after another from a
// cursor; wrap them in Par to run together or At to place them absolutely.
// Sampling is stateless, so any time can be rendered in any order.
type Timeline struct {
	cursor     float64
	end        float64
	par        *parCtx
	tracks     map[trackKey]*vtrack
	order      []trackKey
	vis        map[*Node]anim.Track
	floats     map[*float64]anim.Track
	floatOrder []*float64
	effects    []effect
	cam        *Camera

	// Now is the time the timeline was last applied. Effects and overlays can read it
	// to know when in the film they are being drawn.
	Now float64
}

type parCtx struct{ start, end float64 }

func newTimeline(cam *Camera) *Timeline {
	return &Timeline{tracks: map[trackKey]*vtrack{}, vis: map[*Node]anim.Track{}, floats: map[*float64]anim.Track{}, cam: cam}
}

// Duration is the end of the last action.
func (tl *Timeline) Duration() float64 { return tl.end }

// Cursor is where the next action will start.
func (tl *Timeline) Cursor() float64 { return tl.cursor }

func (tl *Timeline) begin(dur float64) float64 {
	var start float64
	if tl.par != nil {
		start = tl.par.start
		tl.par.end = math.Max(tl.par.end, start+dur)
	} else {
		start = tl.cursor
		tl.cursor += dur
	}
	tl.end = math.Max(tl.end, start+dur)
	return start
}

// Wait pauses the sequence.
func (tl *Timeline) Wait(dur float64) { tl.begin(dur) }

// Par runs everything scheduled inside fn at the same time; the sequence
// continues when the longest one finishes.
func (tl *Timeline) Par(fn func()) {
	saved := tl.par
	ctx := &parCtx{start: tl.cursor, end: tl.cursor}
	if saved != nil {
		ctx.start = saved.start
		ctx.end = saved.start
	}
	tl.par = ctx
	fn()
	tl.par = saved
	if saved != nil {
		saved.end = math.Max(saved.end, ctx.end)
	} else {
		tl.cursor = ctx.end
	}
}

// Seq runs fn's actions one after another. Inside a Par it is one lane that
// runs beside the others.
func (tl *Timeline) Seq(fn func()) {
	saved, savedCursor := tl.par, tl.cursor
	if saved != nil {
		tl.cursor = saved.start
	}
	tl.par = nil
	fn()
	laneEnd := tl.cursor
	tl.par = saved
	if saved != nil {
		saved.end = math.Max(saved.end, laneEnd)
		tl.cursor = savedCursor
	}
}

// At runs fn with the cursor set to absolute time t, then restores the cursor.
func (tl *Timeline) At(t float64, fn func()) {
	saved, savedPar := tl.cursor, tl.par
	tl.cursor, tl.par = t, nil
	fn()
	tl.cursor, tl.par = saved, savedPar
}

func (tl *Timeline) track(n *Node, p prop) *vtrack {
	k := trackKey{n, p}
	if t, ok := tl.tracks[k]; ok {
		return t
	}
	cur := map[prop]geom.Vector{propPos: n.Pos, propRot: n.Rot, propScl: n.Scl}[p]
	t := &vtrack{last: cur}
	t.X = anim.Track{{T: 0, V: cur.X}}
	t.Y = anim.Track{{T: 0, V: cur.Y}}
	t.Z = anim.Track{{T: 0, V: cur.Z}}
	tl.tracks[k] = t
	tl.order = append(tl.order, k)
	return t
}

// tween moves a property from where it last was to target over dur seconds.
func (tl *Timeline) tween(n *Node, p prop, target geom.Vector, dur float64, e Ease) {
	start := tl.begin(dur)
	t := tl.track(n, p)
	if start > t.lastT { // hold the old value until this move begins
		t.X = append(t.X, anim.Key{T: start, V: t.last.X})
		t.Y = append(t.Y, anim.Key{T: start, V: t.last.Y})
		t.Z = append(t.Z, anim.Key{T: start, V: t.last.Z})
	}
	end := start + dur
	t.X = append(t.X, anim.Key{T: end, V: target.X, Ease: e})
	t.Y = append(t.Y, anim.Key{T: end, V: target.Y, Ease: e})
	t.Z = append(t.Z, anim.Key{T: end, V: target.Z, Ease: e})
	t.last, t.lastT = target, end
}

// Move glides n to pos (relative to its parent).
func (tl *Timeline) Move(n *Node, pos geom.Vector, dur float64, e Ease) {
	tl.tween(n, propPos, pos, dur, e)
}

// Turn rotates n to the euler angles rot (radians).
func (tl *Timeline) Turn(n *Node, rot geom.Vector, dur float64, e Ease) {
	tl.tween(n, propRot, rot, dur, e)
}

// Scale resizes n.
func (tl *Timeline) Scale(n *Node, scl geom.Vector, dur float64, e Ease) {
	tl.tween(n, propScl, scl, dur, e)
}

// Follow moves n along f(u), u from 0 to 1, over dur seconds.
func (tl *Timeline) Follow(n *Node, f func(u float64) geom.Vector, dur float64, e Ease) {
	start := tl.begin(dur)
	t := tl.track(n, propPos)
	steps := int(math.Max(2, math.Ceil(dur*48)))
	if start > t.lastT {
		t.X = append(t.X, anim.Key{T: start, V: t.last.X})
		t.Y = append(t.Y, anim.Key{T: start, V: t.last.Y})
		t.Z = append(t.Z, anim.Key{T: start, V: t.last.Z})
	}
	for i := 0; i <= steps; i++ {
		u := float64(i) / float64(steps)
		if e != nil {
			u = e(u)
		}
		p := f(u)
		T := start + dur*float64(i)/float64(steps)
		t.X = append(t.X, anim.Key{T: T, V: p.X})
		t.Y = append(t.Y, anim.Key{T: T, V: p.Y})
		t.Z = append(t.Z, anim.Key{T: T, V: p.Z})
		t.last, t.lastT = p, T
	}
}

// Show and Hide switch a node (and its children) on or off at the cursor.
func (tl *Timeline) Show(n *Node) { tl.setVis(n, 1) }
func (tl *Timeline) Hide(n *Node) { tl.setVis(n, 0) }

func (tl *Timeline) setVis(n *Node, v float64) {
	start := tl.begin(0)
	tr, ok := tl.vis[n]
	if !ok {
		tr = anim.Track{{T: 0, V: n.vis}}
	}
	prev := tr[len(tr)-1].V
	tl.vis[n] = append(tr, anim.Key{T: start, V: prev}, anim.Key{T: start, V: v})
}

// Float animates any number you point at: the camera's field of view, the hour
// of day, a light level, a spring stiffness. This is how scalars join the timeline.
func (tl *Timeline) Float(p *float64, to, dur float64, e Ease) {
	start := tl.begin(dur)
	tr, ok := tl.floats[p]
	if !ok {
		tr = anim.Track{{T: 0, V: *p}}
		tl.floatOrder = append(tl.floatOrder, p)
	}
	prev := tr[len(tr)-1]
	if start > prev.T {
		tr = append(tr, anim.Key{T: start, V: prev.V})
	}
	tl.floats[p] = append(tr, anim.Key{T: start + dur, V: to, Ease: e})
}

// Fov animates the camera's vertical field of view (degrees).
func (tl *Timeline) Fov(deg, dur float64, e Ease) { tl.Float(&tl.cam.Fovy, deg, dur, e) }

// Do runs fn(p) every frame while the action is active, p running 0 to 1 (and
// staying 1 afterwards). fn should set node properties as a pure function of
// p; this is the hook for any custom motion: walk cycles, wobble, physics.
func (tl *Timeline) Do(dur float64, fn func(p float64)) {
	start := tl.begin(dur)
	tl.effects = append(tl.effects, effect{start, dur, fn})
}

// apply sets every animated property to its value at time t seconds.
func (tl *Timeline) apply(t float64) {
	tl.Now = t
	for _, k := range tl.order {
		v := tl.tracks[k].At(t)
		switch k.p {
		case propPos:
			k.n.Pos = v
		case propRot:
			k.n.Rot = v
		case propScl:
			k.n.Scl = v
		}
	}
	for n, tr := range tl.vis {
		n.vis = tr.At(t)
	}
	for _, p := range tl.floatOrder {
		*p = tl.floats[p].At(t)
	}
	for _, e := range tl.effects {
		if t < e.start {
			continue
		}
		p := 1.0
		if e.dur > 0 {
			p = anim.Clamp((t - e.start) / e.dur)
		}
		e.fn(p)
	}
}
