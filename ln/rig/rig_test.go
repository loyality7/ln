package rig

import (
	"math"
	"testing"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/stage"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// a two-joint arm: an upper arm hanging from "shoulder" and a forearm from "elbow"
func armSpec() Spec {
	return Spec{Height: 1, Joints: []Joint{
		{Name: "shoulder", At: geom.V(0, 0, 1), Sign: -1, Limb: &Limb{Radius: 0.03, Length: 0.3}},
		{Name: "elbow", Parent: "shoulder", At: geom.V(0, 0, -0.3), Sign: -1, Limb: &Limb{Radius: 0.03, Length: 0.3}, Track: true},
	}}
}

func newArm(origin, facing float64) (*stage.Stage, *Skeleton) {
	st := stage.New(64, 64, 10)
	return st, Build(st.Root, armSpec(), 1, origin, facing)
}

func TestDegAndMix(t *testing.T) {
	p := Deg(Pose{"elbow": 90, "root.yaw": 180, "pelvis.z": 0.5})
	if !near(p["elbow"], math.Pi/2) || !near(p["root.yaw"], math.Pi) || p["pelvis.z"] != 0.5 {
		t.Fatalf("Deg = %v (angles to radians, distances untouched)", p)
	}
	m := Pose{"a": 2}.Mix(Pose{"a": 4, "b": 10}, 0.5)
	if m["a"] != 3 || m["b"] != 5 {
		t.Fatalf("Mix = %v: a missing channel is at rest (0)", m)
	}
}

func TestApplySignAxisAndReset(t *testing.T) {
	_, sk := newArm(0, 1)
	sk.Apply(Pose{"elbow": 0.5, "shoulder.z": 0.2})
	if !near(sk.Joint("elbow").Rot.Y, -0.5) { // Sign -1: positive swings forward
		t.Fatalf("elbow rot %v", sk.Joint("elbow").Rot)
	}
	if !near(sk.Joint("shoulder").Pos.Z, 1.2) {
		t.Fatalf("shoulder pos %v", sk.Joint("shoulder").Pos)
	}
	sk.Apply(Pose{})
	if sk.Joint("elbow").Rot.Y != 0 || !near(sk.Joint("shoulder").Pos.Z, 1) {
		t.Fatal("channels left out of a pose must return to rest")
	}
}

func TestFacingMirrorsTravel(t *testing.T) {
	_, a := newArm(0, 1)
	_, b := newArm(5, -1)
	a.Apply(Pose{"root.x": 2})
	b.Apply(Pose{"root.x": 2})
	if !near(a.Root.Pos.X, 2) || !near(b.Root.Pos.X, 3) {
		t.Fatalf("forward is +x for a (%v) and -x for b (%v)", a.Root.Pos.X, b.Root.Pos.X)
	}
	if !near(b.Root.Rot.Z, math.Pi) {
		t.Fatal("a figure facing -x is turned half a revolution")
	}
}

func TestKeysCycleAndModifiers(t *testing.T) {
	k := Keys(1, K(0, Pose{"elbow": 0}), K(1, Pose{"elbow": 90})).Travel(4, 0.5, 1)
	if p := k.pose(0.5, 0); !near(p["elbow"], math.Pi/4) || !near(p["root.x"], 0) {
		t.Fatalf("halfway: %v", p)
	}
	if p := k.pose(1, 10); !near(p["root.x"], 14) {
		t.Fatalf("travel is added to where the move starts: %v", p)
	}
	c := Cycle(1, 1, Pose{}, Osc{Ch: "elbow", Amp: 90})
	if p := c.pose(0.25, 0); !near(p["elbow"], math.Pi/2) {
		t.Fatalf("quarter cycle of a 90 degree sine: %v", p)
	}
	j := Keys(1, K(0, Pose{}), K(1, Pose{})).Arc(2, 0.25, 0.75).Spin("pelvis", 1, 0.25, 0.75)
	if p := j.pose(0.5, 0); !near(p["root.z"], 2) || !near(p["pelvis"], math.Pi) {
		t.Fatalf("peak of a jump with a flip: %v", p)
	}
	if p := j.pose(1, 0); !near(p["root.z"], 0) || !near(p["pelvis"], 0) {
		t.Fatalf("a whole flip lands where it started: %v", p)
	}
}

func TestActorPlaysInSequenceAndMarksEvents(t *testing.T) {
	st, sk := newArm(0, 1)
	a := NewActor(st.TL, sk)
	a.Play(Keys(1, K(0, Pose{}), K(1, Pose{})).Travel(2, 0, 1))
	start := a.Play(Keys(2, K(0, Pose{}), K(1, Pose{})).Travel(1, 0, 1).
		Mark(Mark{Kind: "impact", At: 0.5, Secs: 0.3, X: 0.5}).
		Mark(Mark{Kind: "dust", At: 1, Secs: 0.6, AtEnd: true}))
	if start != 1 || a.X() != 3 || st.TL.Duration() != 3 {
		t.Fatalf("second move starts at %v, actor at x=%v, film %vs", start, a.X(), st.TL.Duration())
	}
	if len(a.Events) != 2 || !near(a.Events[0].T, 2) || !near(a.Events[0].X, 2.5) || !near(a.Events[1].X, 3) {
		t.Fatalf("events %+v", a.Events)
	}
	st.Shot(2) // evaluate the timeline half way through the second move
	if !near(sk.Root.Pos.X, 2.5) {
		t.Fatalf("root at %v, want 2.5", sk.Root.Pos.X)
	}
}

func TestBeatRunsLanesTogetherAndFreezeHolds(t *testing.T) {
	st := stage.New(64, 64, 10)
	a := NewActor(st.TL, Build(st.Root, armSpec(), 1, 0, 1))
	b := NewActor(st.TL, Build(st.Root, armSpec(), 1, 5, -1))
	Beat(st.TL, Moves(a, Keys(1, K(0, Pose{}), K(1, Pose{"elbow": 90}))), Moves(b, Keys(0.5, K(0, Pose{}), K(1, Pose{})), Keys(1, K(0, Pose{}), K(1, Pose{}))))
	if st.TL.Duration() != 1.5 {
		t.Fatalf("the beat lasts as long as its slowest lane: %v", st.TL.Duration())
	}
	a.Play(Freeze(1))
	st.Shot(2)
	if !near(a.Skel.Joint("elbow").Rot.Y, -math.Pi/2) {
		t.Fatalf("a frozen actor keeps its last pose: %v", a.Skel.Joint("elbow").Rot)
	}
	if g := Gap(a, b); !near(g, 5) {
		t.Fatalf("gap %v", g)
	}
}

func TestTipsFollowTrackedJoints(t *testing.T) {
	_, sk := newArm(0, 1)
	sk.Apply(Pose{})
	tips := sk.Tips()
	if len(tips) != 1 || !near(tips[0].Z, 0.4) { // shoulder at 1, forearm 0.3 + upper arm 0.3 down
		t.Fatalf("tips %v", tips)
	}
}
