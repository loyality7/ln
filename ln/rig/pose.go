// Package rig animates characters without hardcoding any movement. A skeleton is
// described as data (joints, limbs), a pose is a set of named channel values, and a
// move is a Clip built from keyframe tables or oscillators. An Actor plays clips on
// the timeline, blending between them, and records events (a landing, a hit) for the
// fx package to draw.
//
// Channels are named strings:
//
//	"knee"        the angle of joint "knee" about its axis (radians; Deg converts)
//	"pelvis.z"    an offset added to the joint's position along x, y or z
//	"root.x"      forward travel of the whole skeleton; "root.z" height; "root.yaw" turn
package rig

import (
	"math"
	"strings"
)

// Pose is a value for each channel it mentions; channels it leaves out are at rest (0).
type Pose map[string]float64

// isAngle reports whether a channel is a rotation (radians) rather than a distance.
func isAngle(ch string) bool { return !strings.Contains(ch, ".") || ch == "root.yaw" }

// Deg converts the rotation channels of a pose written in degrees to radians.
// Distance channels (those with .x .y .z) are left as they are.
func Deg(p Pose) Pose {
	out := make(Pose, len(p))
	for k, v := range p {
		if isAngle(k) {
			v *= math.Pi / 180
		}
		out[k] = v
	}
	return out
}

// With is p with the channels of over replaced, for building a pose from another.
func (p Pose) With(over Pose) Pose {
	out := make(Pose, len(p)+len(over))
	for k, v := range p {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// Smooth eases 0..1 with zero slope at both ends.
func Smooth(t float64) float64 {
	t = math.Max(0, math.Min(1, t))
	return t * t * (3 - 2*t)
}

// Mix blends two poses channel by channel; t = 0 gives a, t = 1 gives b.
func (a Pose) Mix(b Pose, t float64) Pose {
	out := make(Pose, len(a))
	for k, v := range a {
		out[k] = lerp(v, b[k], t)
	}
	for k, v := range b {
		if _, ok := out[k]; !ok {
			out[k] = lerp(0, v, t)
		}
	}
	return out
}
