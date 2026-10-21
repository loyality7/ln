package stickman

import "github.com/loyality7/ln/ln/rig"

// Poses are written in degrees. Channels: shoulderN/F, elbowN/F, hipN/F, kneeN/F,
// spine, head, pelvis (leans or flips the whole body), pelvis.z (drops the hips),
// root.z (height off the ground). Anything left out is at rest.

type pose = rig.Pose

var (
	rest = pose{"elbowN": 8, "elbowF": 8}

	guard = pose{
		"hipN": -18, "hipF": 28, "kneeN": 36, "kneeF": 42, "pelvis.z": -0.10, "spine": 8,
		"shoulderN": 72, "shoulderF": 56, "elbowN": 112, "elbowF": 116,
	}
)

// crouch is a squat; amount 0 is standing, 1 is deep.
func crouch(amount float64) pose {
	a := amount
	return pose{
		"hipN": 58 * a, "hipF": 58 * a, "kneeN": 110 * a, "kneeF": 110 * a, "spine": 26 * a,
		"pelvis.z": -0.30 * a, "shoulderN": -40 * a, "shoulderF": -40 * a, "elbowN": 8, "elbowF": 8,
	}
}

// tuck curls into a ball; amount 1 is fully tucked.
func tuck(amount float64) pose {
	a := amount
	return pose{
		"hipN": 112 * a, "hipF": 112 * a, "kneeN": 128 * a, "kneeF": 128 * a, "spine": 38 * a,
		"shoulderN": 70 * a, "shoulderF": 70 * a, "elbowN": 115 * a, "elbowF": 115 * a,
	}
}

var (
	air = pose{
		"shoulderN": 168, "shoulderF": 168, "elbowN": 10, "elbowF": 10, "hipN": 24, "hipF": 8,
		"kneeN": 48, "kneeF": 30, "spine": -6,
	}
	lying = pose{
		"pelvis": -90, "pelvis.z": -0.88, "shoulderN": 15, "shoulderF": 25, "elbowN": 30, "elbowF": 50,
		"hipN": 20, "hipF": 8, "kneeN": 25, "kneeF": 10,
	}
)

// ---- everyday moves -------------------------------------------------------------

// Idle stands and breathes.
func Idle(dur float64) rig.Clip {
	return rig.Cycle(dur, 1.2*dur, rest,
		rig.Osc{Ch: "spine", Amp: 1.5}, rig.Osc{Ch: "pelvis.z", Amp: -0.004},
		rig.Osc{Ch: "shoulderN", Amp: 3}, rig.Osc{Ch: "shoulderF", Amp: -3})
}

// Wave raises the near arm and waves.
func Wave(dur float64) rig.Clip {
	return rig.Cycle(dur, 3.2*dur, rest.With(pose{"shoulderN": 165, "elbowN": 30, "head": 6}),
		rig.Osc{Ch: "elbowN", Amp: 32})
}

// Walk strolls dist units.
func Walk(dur, dist float64) rig.Clip {
	return rig.Cycle(dur, 0.95*dur, rest.With(pose{"elbowN": 20, "elbowF": 20, "spine": 3, "kneeN": 6, "kneeF": 6, "pelvis.z": -0.012}),
		rig.Osc{Ch: "hipN", Amp: 30}, rig.Osc{Ch: "hipF", Amp: -30},
		rig.Osc{Ch: "kneeN", Amp: 46, Phase: 0.25, Wave: rig.Half}, rig.Osc{Ch: "kneeF", Amp: 46, Phase: 0.75, Wave: rig.Half},
		rig.Osc{Ch: "shoulderN", Amp: -26}, rig.Osc{Ch: "shoulderF", Amp: 26},
		rig.Osc{Ch: "pelvis.z", Amp: 0.016, Mult: 2, Phase: 0.25},
	).Travel(dist, 0, 1)
}

// Run sprints dist units, with speed lines.
func Run(dur, dist float64) rig.Clip {
	return rig.Cycle(dur, 1.55*dur, rest.With(pose{"elbowN": 95, "elbowF": 95, "spine": 17, "kneeN": 22, "kneeF": 22, "pelvis.z": -0.06}),
		rig.Osc{Ch: "hipN", Amp: 62}, rig.Osc{Ch: "hipF", Amp: -62},
		rig.Osc{Ch: "kneeN", Amp: 92, Phase: 0.25, Wave: rig.Half}, rig.Osc{Ch: "kneeF", Amp: 92, Phase: 0.75, Wave: rig.Half},
		rig.Osc{Ch: "shoulderN", Amp: -58}, rig.Osc{Ch: "shoulderF", Amp: 58},
		rig.Osc{Ch: "pelvis.z", Amp: 0.05, Wave: rig.Abs}, rig.Osc{Ch: "root.z", Amp: 0.05, Wave: rig.Abs},
	).Travel(dist, 0, 1).
		Mark(rig.Mark{Kind: "speed", At: 0, Span: 1}).
		Mark(rig.Mark{Kind: "dust", At: 0, Secs: 0.6})
}

// Jump leaps height units up and dist forward.
func Jump(dur, height, dist float64) rig.Clip {
	return rig.Keys(dur,
		rig.K(0, rest), rig.K(0.2, crouch(1)), rig.K(0.3, air), rig.K(0.7, air),
		rig.K(0.8, crouch(0.85)), rig.K(1, rest)).
		Arc(height, 0.2, 0.8).Travel(dist, 0.2, 0.8).
		Mark(rig.Mark{Kind: "dust", At: 0.78, Secs: 0.6, AtEnd: true})
}

// Flip is a somersault: dir +1 forward, -1 backward.
func Flip(dur, dir, height, dist float64) rig.Clip {
	back := crouch(1).With(pose{"shoulderN": -55 * dir, "shoulderF": -55 * dir})
	return rig.Keys(dur,
		rig.K(0, rest), rig.K(0.17, back), rig.K(0.30, tuck(0.8)), rig.K(0.5, tuck(1)), rig.K(0.7, tuck(0.8)),
		rig.K(0.83, pose{"hipN": 30, "hipF": 30, "kneeN": 20, "kneeF": 20, "shoulderN": 150, "shoulderF": 150, "spine": 5}),
		rig.K(0.9, crouch(0.9)), rig.K(1, rest)).
		Spin("pelvis", dir, 0.17, 0.83).Arc(height, 0.17, 0.83).Travel(dist, 0.17, 0.83).
		Mark(rig.Mark{Kind: "trail", At: 0.17, Span: 0.66}).
		Mark(rig.Mark{Kind: "dust", At: 0.81, Secs: 0.6, AtEnd: true})
}

// Dance bounces, pumps its arms and spins twice.
func Dance(dur float64) rig.Clip {
	return rig.Cycle(dur, 2.4*dur, rest.With(pose{"shoulderN": 70, "shoulderF": 70, "elbowN": 60, "elbowF": 60}),
		rig.Osc{Ch: "pelvis.z", Amp: -0.06, Wave: rig.Abs},
		rig.Osc{Ch: "shoulderN", Amp: 80}, rig.Osc{Ch: "shoulderF", Amp: -80},
		rig.Osc{Ch: "elbowN", Amp: 30, Phase: 0.25}, rig.Osc{Ch: "elbowF", Amp: -30, Phase: 0.25},
		rig.Osc{Ch: "hipN", Amp: 22}, rig.Osc{Ch: "hipF", Amp: -22},
		rig.Osc{Ch: "kneeN", Amp: 30, Wave: rig.Half}, rig.Osc{Ch: "kneeF", Amp: 30, Phase: 0.5, Wave: rig.Half},
		rig.Osc{Ch: "spine", Amp: 6, Phase: 0.25}, rig.Osc{Ch: "head", Amp: 8},
	).Spin("root.yaw", 2, 0, 1)
}

// Fall topples backward and lies there.
func Fall(dur float64) rig.Clip {
	flail := pose{"pelvis": -45, "pelvis.z": -0.3, "shoulderN": 150, "shoulderF": 130, "elbowN": 30, "elbowF": 50, "hipN": 10, "kneeN": 15}
	return rig.Keys(dur, rig.K(0, rest), rig.K(0.4, flail), rig.K(1, lying)).Travel(-0.5, 0, 1)
}

// GetUp rises from the ground, pushing through a kneel.
func GetUp(dur float64) rig.Clip {
	kneel := pose{"hipN": 70, "hipF": 40, "kneeN": 115, "kneeF": 70, "spine": 20, "shoulderN": 40, "shoulderF": 40, "pelvis": -30, "pelvis.z": -0.5}
	return rig.Keys(dur, rig.K(0, lying), rig.K(0.5, kneel), rig.K(1, rest)).Travel(0.4, 0, 1)
}

// Bow bends forward, holds, and rises.
func Bow(dur float64) rig.Clip {
	bow := pose{"spine": 78, "head": 18, "hipN": 8, "hipF": 8, "shoulderN": 12, "shoulderF": 12}
	return rig.Keys(dur, rig.K(0, rest), rig.K(0.35, bow), rig.K(0.65, bow), rig.K(1, rest))
}

// ---- fighting moves -------------------------------------------------------------

// Guard holds the fighting stance, bouncing lightly.
func Guard(dur float64) rig.Clip {
	return rig.Cycle(dur, 2.3*dur, guard,
		rig.Osc{Ch: "pelvis.z", Amp: 0.025}, rig.Osc{Ch: "kneeN", Amp: 5}, rig.Osc{Ch: "kneeF", Amp: -5})
}

// Dash slides forward (negative dist: back) in the stance.
func Dash(dur, dist float64) rig.Clip {
	return rig.Cycle(dur, 1.5, guard,
		rig.Osc{Ch: "hipN", Amp: 24}, rig.Osc{Ch: "hipF", Amp: -24},
		rig.Osc{Ch: "kneeN", Amp: 20, Wave: rig.Half}, rig.Osc{Ch: "kneeF", Amp: 20, Phase: 0.5, Wave: rig.Half},
		rig.Osc{Ch: "pelvis.z", Amp: 0.03, Wave: rig.Abs},
	).Travel(dist, 0, 1)
}

// Punch throws a straight punch with a lunge; hit adds a comic star where it lands.
func Punch(dur float64, hit bool) rig.Clip {
	stance := guard.With(pose{"hipN": -24, "hipF": 34, "kneeN": 38, "kneeF": 44, "pelvis.z": -0.09})
	c := rig.Keys(dur,
		rig.K(0, guard),
		rig.K(0.3, stance.With(pose{"shoulderN": -28, "elbowN": 130, "spine": 8})),
		rig.K(0.46, stance.With(pose{"shoulderN": 90, "elbowN": 6, "spine": 22})),
		rig.K(0.56, stance.With(pose{"shoulderN": 90, "elbowN": 6, "spine": 22})),
		rig.K(1, guard)).Travel(0.35, 0, 0.5)
	if hit {
		c = c.Mark(rig.Mark{Kind: "impact", At: 0.5, Secs: 0.3, X: 1.17, Z: 1.28})
	}
	return c
}

// Kick throws a high kick.
func Kick(dur float64, hit bool) rig.Clip {
	chamber := guard.With(pose{"hipF": 70, "kneeF": 115, "hipN": -8, "kneeN": 18, "spine": -24, "shoulderN": 82, "shoulderF": -55, "elbowN": 20, "elbowF": 20, "pelvis.z": -0.04})
	extend := chamber.With(pose{"hipF": 98, "kneeF": 6})
	c := rig.Keys(dur, rig.K(0, guard), rig.K(0.28, chamber), rig.K(0.46, extend), rig.K(0.62, extend), rig.K(1, guard)).Travel(0.2, 0, 0.5)
	if hit {
		c = c.Mark(rig.Mark{Kind: "impact", At: 0.46, Secs: 0.3, X: 1.13, Z: 1.0})
	}
	return c
}

// Block crosses the arms in front of the face and rides the hit back a little.
func Block(dur, push float64) rig.Clip {
	up := guard.With(pose{"shoulderN": 118, "shoulderF": 112, "elbowN": 128, "elbowF": 132, "spine": -6, "pelvis.z": -0.16, "kneeN": 48, "kneeF": 54})
	return rig.Keys(dur, rig.K(0, guard), rig.K(0.2, up), rig.K(0.7, up), rig.K(1, guard)).Travel(-push, 0, 0.45)
}

// Duck drops low to slip under a strike.
func Duck(dur float64) rig.Clip {
	low := guard.With(pose{"pelvis.z": -0.46, "hipN": 75, "hipF": 80, "kneeN": 135, "kneeF": 140, "spine": 40, "head": -12})
	return rig.Keys(dur, rig.K(0, guard), rig.K(0.28, low), rig.K(0.62, low), rig.K(1, guard))
}

// Evade hops backward out of range.
func Evade(dur, dist float64) rig.Clip {
	hop := guard.With(pose{"hipN": 25, "hipF": 48, "kneeN": 60, "kneeF": 40, "spine": -14, "shoulderN": 70, "shoulderF": 30})
	return rig.Keys(dur, rig.K(0, guard), rig.K(0.5, hop), rig.K(1, guard)).Arc(0.35, 0, 1).Travel(-dist, 0, 1)
}

// Hit staggers back from a blow.
func Hit(dur, push float64) rig.Clip {
	jolt := guard.With(pose{"spine": -30, "head": -30, "shoulderN": -45, "shoulderF": -25, "elbowN": 30, "elbowF": 40, "hipN": 14, "hipF": -20, "kneeN": 34, "kneeF": 30, "pelvis.z": -0.12})
	return rig.Keys(dur, rig.K(0, guard), rig.K(0.22, jolt), rig.K(1, guard)).Travel(-push, 0, 0.7)
}

// Uppercut crouches and drives a fist up.
func Uppercut(dur float64, hit bool) rig.Clip {
	wind := guard.With(pose{"pelvis.z": -0.38, "hipN": 40, "hipF": 60, "kneeN": 90, "kneeF": 100, "shoulderN": 8, "elbowN": 105, "spine": 20})
	strike := guard.With(pose{"pelvis.z": 0, "kneeN": 27, "kneeF": 30, "shoulderN": 172, "elbowN": 8, "spine": -14, "root.z": 0.16})
	c := rig.Keys(dur, rig.K(0, guard), rig.K(0.3, wind), rig.K(0.48, strike), rig.K(0.62, strike), rig.K(1, guard)).Travel(0.3, 0, 0.5)
	if hit {
		c = c.Mark(rig.Mark{Kind: "impact", At: 0.46, Secs: 0.3, X: 0.8, Z: 1.55})
	}
	return c
}

// SpinKick whirls around and lashes out with a straight leg.
func SpinKick(dur float64, hit bool) rig.Clip {
	chamber := guard.With(pose{"hipF": 60, "kneeF": 40, "shoulderN": 60, "shoulderF": -50, "spine": -8})
	lash := guard.With(pose{"hipF": 96, "kneeF": 4, "hipN": -10, "kneeN": 24, "shoulderN": 88, "shoulderF": -70, "elbowN": 12, "elbowF": 12, "spine": -14, "root.z": 0.1})
	c := rig.Keys(dur, rig.K(0, guard), rig.K(0.35, chamber), rig.K(0.55, lash), rig.K(0.75, lash), rig.K(1, guard)).
		Spin("root.yaw", 1, 0, 1).Travel(0.25, 0, 0.6)
	if hit {
		c = c.Mark(rig.Mark{Kind: "impact", At: 0.52, Secs: 0.3, X: 1.175, Z: 1.05})
	}
	return c
}

// FlyingKick leaps at the opponent feet first; the hit lands near the top of the leap.
func FlyingKick(dur, dist float64, hit bool) rig.Clip {
	fly := pose{"pelvis": -32, "hipF": 100, "kneeF": 4, "hipN": -30, "kneeN": 55, "shoulderN": 120, "shoulderF": -35, "elbowN": 25, "elbowF": 25}
	c := rig.Keys(dur, rig.K(0, rest), rig.K(0.2, crouch(1)), rig.K(0.4, fly), rig.K(0.75, fly),
		rig.K(0.9, crouch(0.85)), rig.K(1, guard)).Arc(0.95, 0.2, 0.78).Travel(dist, 0.2, 0.78).
		Mark(rig.Mark{Kind: "dust", At: 0.78, Secs: 0.6, AtEnd: true})
	if hit {
		c = c.Mark(rig.Mark{Kind: "impact", At: 0.66, Secs: 0.3, X: dist*0.88 + 0.92, Z: 1.05})
	}
	return c
}

// KO is a knockout: launched backward through the air, landing flat.
func KO(dur, push float64) rig.Clip {
	flying := pose{"pelvis": -50, "pelvis.z": -0.2, "shoulderN": 150, "shoulderF": 120, "elbowN": 25, "elbowF": 45, "hipN": 30, "kneeN": 40}
	return rig.Keys(dur, rig.K(0, guard), rig.K(0.3, flying), rig.K(0.66, lying), rig.K(0.8, lying.With(pose{"pelvis": -86})), rig.K(1, lying)).
		Arc(1.0, 0, 0.62).Travel(-push, 0, 1).
		Mark(rig.Mark{Kind: "dust", At: 0.62, Secs: 0.7, AtEnd: true})
}

// Victory pumps a fist and hops.
func Victory(dur float64) rig.Clip {
	return rig.Cycle(dur, 2.2*dur, pose{"shoulderN": 172, "shoulderF": 172, "elbowN": 15, "elbowF": 15, "head": -10, "spine": -6},
		rig.Osc{Ch: "root.z", Amp: 0.22, Wave: rig.Abs}, rig.Osc{Ch: "hipN", Amp: 12, Wave: rig.Abs})
}
