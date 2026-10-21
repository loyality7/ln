// Package mannequin is an artist's wooden figure: a human body with real
// proportions, tapered limbs and no face. Every joint is a stage node, so a pose
// or a walk cycle is just rotating those nodes.
package mannequin

import (
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/model"
	"github.com/loyality7/ln/ln/solid"
	"github.com/loyality7/ln/ln/stage"
)

// Figure holds the joints. Rotate them (Rot.X swings a limb forward) to pose it.
// The figure stands at its Root's origin and faces -y.
type Figure struct {
	Root, Pelvis, Chest, Neck, Head *stage.Node
	Shoulder, Elbow                 [2]*stage.Node // left, right
	Hip, Knee, Ankle                [2]*stage.Node
}

// limb is a smooth tapered solid hanging from its top at z=0 down to z=-length.
// radii are sampled evenly along it; this is how a thigh gets thick at the hip,
// narrow at the knee, and a calf its swell.
func limb(length float64, radii ...float64) *model.Mesh {
	pts := []solid.Pt{{X: 0, Y: 0}, {X: radii[0] * 0.75, Y: -0.012 * length}}
	for i, r := range radii {
		pts = append(pts, solid.Pt{X: r, Y: -length * (0.02 + 0.96*float64(i)/float64(len(radii)-1))})
	}
	last := radii[len(radii)-1]
	pts = append(pts, solid.Pt{X: last * 0.7, Y: -length * 1.005}, solid.Pt{X: 0, Y: -length * 1.012})
	return solid.Lathe(solid.Smooth(pts, 5), 22)
}

// foot is a heel, an arch and toes: a low wedge with a rounded front.
func foot() *model.Mesh {
	heel := solid.Place(solid.Ellipsoid(0.040, 0.050, 0.040), geom.Translate(geom.Vector{Y: 0.035, Z: 0.04}))
	ball := solid.Place(solid.Ellipsoid(0.048, 0.090, 0.030), geom.Translate(geom.Vector{Y: -0.045, Z: 0.032}))
	toes := solid.Place(solid.Ellipsoid(0.042, 0.050, 0.020), geom.Translate(geom.Vector{Y: -0.115, Z: 0.022}))
	return solid.Shade(solid.Merge(heel, ball, toes), 60)
}

// New builds a figure of the given height in world units under parent, standing
// in a relaxed pose with a little weight on one leg.
func New(parent *stage.Node, height float64) *Figure {
	f := &Figure{}
	f.Root = parent.Group("mannequin").Uniform(height / 1.75)
	f.Pelvis = f.Root.Group("pelvis").At(0, 0, 0.93)
	f.Pelvis.Add(solid.Ellipsoid(0.165, 0.105, 0.105)).At(0, 0, 0.0)
	f.Chest = f.Pelvis.Group("chest").At(0, 0, 0.10)
	f.Chest.Add(torso()).Size(1, 0.62, 1) // waist, ribcage, shoulders; flatter front to back
	f.Neck = f.Chest.Group("neck").At(0, 0, 0.44)
	f.Neck.Add(solid.Cylinder(0.046, 0.12))
	f.Head = f.Neck.Group("head").At(0, 0.005, 0.20)
	f.Head.Add(solid.Ellipsoid(0.083, 0.098, 0.112))

	for i, side := range []float64{-1, 1} {
		f.Shoulder[i] = f.Chest.Group("shoulder").At(side*0.19, 0, 0.385)
		f.Shoulder[i].Add(solid.Sphere(0.052))
		f.Shoulder[i].Add(limb(0.30, 0.052, 0.050, 0.040))
		f.Elbow[i] = f.Shoulder[i].Group("elbow").At(0, 0, -0.30)
		f.Elbow[i].Add(solid.Sphere(0.040))
		f.Elbow[i].Add(limb(0.26, 0.040, 0.042, 0.027))
		f.Elbow[i].Add(hand()).At(0, 0, -0.27)

		f.Hip[i] = f.Pelvis.Group("hip").At(side*0.088, 0, -0.04)
		f.Hip[i].Add(limb(0.43, 0.098, 0.090, 0.072, 0.054))
		f.Knee[i] = f.Hip[i].Group("knee").At(0, 0, -0.43)
		f.Knee[i].Add(solid.Sphere(0.050))
		f.Knee[i].Add(limb(0.41, 0.054, 0.062, 0.048, 0.033))
		f.Ankle[i] = f.Knee[i].Group("ankle").At(0, 0, -0.41)
		f.Ankle[i].Add(foot()).At(0, 0, -0.04)

		f.Shoulder[i].Merge() // a whole arm is one outlined shape
		f.Hip[i].Merge()      // a whole leg too, so knees and ankles carry no seam lines
		f.Shoulder[i].Rot.Y = side * 0.07
		f.Elbow[i].Rot.X = -0.18
		f.Hip[i].Rot.Z = side * 0.06 // toes turned slightly out
	}
	// relaxed stance: weight on the right leg, left knee eased, shoulders counter-tilted
	f.Knee[0].Rot.X = 0.12
	f.Hip[0].Rot.X = -0.05
	f.Pelvis.Rot.Y = 0.04
	f.Chest.Rot.Y = -0.05
	f.Head.Rot.Z = 0.04
	f.Pelvis.Merge() // trunk and neck as one; the arms and legs keep their own outlines
	f.Head.Merge()
	return f
}

// torso is the trunk from the hips (z=0) up to the neck: narrow waist, full
// ribcage, sloping shoulders. It is round; the caller squashes it front to back.
func torso() *model.Mesh {
	return solid.Lathe(solid.Smooth([]solid.Pt{
		{X: 0, Y: -0.02}, {X: 0.145, Y: 0.0}, {X: 0.132, Y: 0.09}, {X: 0.122, Y: 0.16}, {X: 0.148, Y: 0.27},
		{X: 0.168, Y: 0.35}, {X: 0.175, Y: 0.405}, {X: 0.12, Y: 0.45}, {X: 0.052, Y: 0.475}, {X: 0, Y: 0.485},
	}, 5), 28)
}

// hand is a flat mitten with a thumb, enough to read as a hand at a distance.
func hand() *model.Mesh {
	palm := solid.Place(solid.Ellipsoid(0.030, 0.018, 0.050), geom.Translate(geom.Vector{Z: -0.02}))
	thumb := solid.Place(solid.Ellipsoid(0.012, 0.012, 0.028), geom.Translate(geom.Vector{X: 0.028, Y: -0.008, Z: -0.012}))
	return solid.Shade(solid.Merge(palm, thumb), 60)
}
