package render

import (
	"github.com/loyality7/ln/ln/anim"
	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/rig"
)

// A scene file is plain JSON. Positions and sizes are in world units, with Z up. Run `ln new` for a working one,
// and see the types below for every field.
type scene struct {
	Width    float64     `json:"width"`
	Height   float64     `json:"height"`
	FPS      float64     `json:"fps"`
	Look     string      `json:"look"` // ink | manga | sketch | technical | ascii | colour
	Light    *lighting   `json:"light"`
	Gravity  *gravity    `json:"gravity"` // bend every camera ray around a mass: the scene seen through lensing
	Style    *style      `json:"style"`
	Sky      *[3]float64 `json:"sky"` // colour look: the sky colour
	Camera   camera      `json:"camera"`
	Nodes    []node      `json:"nodes"`
	Actors   []actor     `json:"actors"`   // characters built from joints
	Follow   *follow     `json:"follow"`   // the camera follows the actors
	Captions []caption   `json:"captions"` // text on the picture
	Timeline []action    `json:"timeline"`
}

// lighting is the sun and the ambient light.
type lighting struct {
	Sun      *[3]float64 `json:"sun"`      // direction toward the sun
	Power    float64     `json:"power"`    // sun strength (default 1)
	Ambient  *float64    `json:"ambient"`  // light that reaches everywhere (default 0.2)
	Shadows  *bool       `json:"shadows"`  // cast shadows (default true)
	Exposure float64     `json:"exposure"` // brightness gain (default 1)
	Lights   []lightSpec `json:"lights"`   // more lights; with no "sun", only these light the scene
}

// gravity bends light around a point mass (see ln/physics). Anything that comes too close is swallowed. It is best
// seen with the colour or ascii look; lines drawn straight from shapes (the ink looks, the plotter) do not bend.
type gravity struct {
	Mass    float64     `json:"mass"`    // in world units: the horizon is at 2 x mass
	Center  *[3]float64 `json:"center"`  // default the origin
	Capture float64     `json:"capture"` // rays that fall inside this radius vanish (default 2 x mass)
	Escape  float64     `json:"escape"`  // rays that get past this radius, heading out, are free (default 80 x mass)
}

// lightSpec is one light: point [pos, power, radius], spot [pos, axis, cone, power, radius], directional [dir, power].
type lightSpec struct {
	Type   string      `json:"type"` // point | spot | directional
	Pos    *[3]float64 `json:"pos"`
	Dir    *[3]float64 `json:"dir"`  // directional: toward the light
	Axis   *[3]float64 `json:"axis"` // spot: the way it shines
	Cone   float64     `json:"cone"` // spot: half-angle in radians
	Power  float64     `json:"power"`
	Radius float64     `json:"radius"` // point and spot: half power at this distance
	Size   float64     `json:"size"`   // shadow rays stop this far short, so a lamp does not shade itself
}

// style tunes the ink looks; anything left out keeps the look's own value.
type style struct {
	Outline float64  `json:"outline"` // outline width in pixels
	Hatch   float64  `json:"hatch"`   // hatch line width in pixels
	Wobble  *float64 `json:"wobble"`  // hand-drawn jitter in pixels (0 = ruler straight)
	Boil    *float64 `json:"boil"`    // how often the jitter changes, per second (0 = frozen)
	Detail  float64  `json:"detail"`  // resolution of the shading, relative to the picture (0.5 = half)
	Columns int      `json:"columns"` // ascii look: characters per row
	Ramp    string   `json:"ramp"`    // ascii look: characters from light to dark
	Invert  bool     `json:"invert"`  // ascii look: light text on a dark page
	Floor   float64  `json:"floor"`   // ascii look: tones lighter than this stay blank
	NoEdges bool     `json:"noEdges"` // ascii look: no outline characters
	Step    float64  `json:"step"`    // plotter look: how finely lines are tested for being hidden
}

type camera struct {
	Eye    *[3]float64 `json:"eye"`
	Target *[3]float64 `json:"target"`
	Fov    float64     `json:"fov"` // degrees
}

// node is a thing in the scene. shape is box, sphere, ellipsoid, cylinder, cone, torus, capsule, group, mesh,
// lathe, extrude, tube, mirror, annulus, heightfield or csg. args are its sizes: box [x y z], sphere [r], ellipsoid [rx ry rz],
// cylinder [r h], cone [r h], torus [bigR smallR], capsule [r length], mirror [height half-size],
// extrude [h], tube [radius], mesh [scale], annulus [inner outer] (a flat ring or disc), heightfield [sizeX sizeY height]
// (the "file" is a grayscale picture: white is high).
type node struct {
	Name     string       `json:"name"`
	Shape    string       `json:"shape"`
	Args     []float64    `json:"args"`
	At       *[3]float64  `json:"at"`
	Turn     *[3]float64  `json:"turn"` // radians about x, y, z
	Scale    *[3]float64  `json:"scale"`
	Paint    *[3]float64  `json:"paint"` // colour 0..1, for the colour look
	NoLine   bool         `json:"noOutline"`
	Hidden   bool         `json:"hidden"`
	Lamp     *lamp        `json:"lamp"`    // make the node a light
	File     string       `json:"file"`    // mesh: an .obj or .stl file, fitted into a unit cube
	Profile  [][2]float64 `json:"profile"` // lathe: [distance from the axis, height] points, spun around Z
	Polygon  [][2]float64 `json:"polygon"` // extrude: the outline, lifted by args[0]
	Path     [][3]float64 `json:"path"`    // tube: the centre line
	Sides    int          `json:"sides"`   // lathe and tube: number of sides
	Smooth   int          `json:"smooth"`  // lathe: round the profile, with this many points per span
	Closed   bool         `json:"closed"`  // tube: join the ends into a loop
	Op       string       `json:"op"`      // csg: difference or intersection
	Parts    []part       `json:"parts"`   // csg: the pieces; the first minus the rest
	Children []node       `json:"children"`
}

// lamp makes a node shine: full power at the node, half at radius.
type lamp struct {
	Power  float64 `json:"power"`
	Radius float64 `json:"radius"`
}

// part is one piece of a csg shape: cube [x y z], sphere [r], cylinder [r z0 z1] or cone [r h], placed at at.
type part struct {
	Shape string      `json:"shape"`
	Args  []float64   `json:"args"`
	At    *[3]float64 `json:"at"`
}

// actor is a skeleton: joints with capsule limbs, posed by name. A pose names a joint to turn it (degrees), or
// "joint.x|y|z" to shift it, or root.x / root.z / root.yaw to move the whole body.
type actor struct {
	Name   string  `json:"name"`
	Height float64 `json:"height"` // how tall it stands (the spec's own height is scaled to this)
	Drawn  float64 `json:"drawn"`  // the height the joint measures were drawn at (default: height)
	Origin float64 `json:"origin"` // where it stands along x
	Facing float64 `json:"facing"` // 1 faces +x, -1 faces -x
	Axis   string  `json:"axis"`   // the axis joints turn about unless they say: y (default), x or z
	Joints []joint `json:"joints"`
}

type joint struct {
	Name   string      `json:"name"`
	Parent string      `json:"parent"`
	At     [3]float64  `json:"at"` // offset from the parent
	Axis   string      `json:"axis"`
	Sign   float64     `json:"sign"` // -1 flips the way a positive angle turns
	Limb   *limb       `json:"limb"`
	Parts  []jointPart `json:"parts"`
	Track  bool        `json:"track"`
}

type limb struct {
	Radius float64 `json:"radius"`
	Length float64 `json:"length"`
	Up     bool    `json:"up"` // the limb points up instead of down
}

// jointPart is an extra solid on a joint: a head, a fist, a foot.
type jointPart struct {
	Shape string     `json:"shape"` // sphere [r], box [x y z], ellipsoid [rx ry rz], capsule [r length]
	Args  []float64  `json:"args"`
	At    [3]float64 `json:"at"`
}

type follow struct {
	Offset *[3]float64        `json:"offset"` // the eye's place relative to the point followed
	Lead   float64            `json:"lead"`
	Height float64            `json:"height"`
	Zoom   float64            `json:"zoom"` // pull back as actors separate
	Shake  map[string]float64 `json:"shake"`
}

type caption struct {
	At   float64 `json:"at"`
	Dur  float64 `json:"dur"`
	Text string  `json:"text"`
}

// action is one step of the timeline: exactly one of move, turn, scale, follow, orbit, wait, fov, show, hide, par,
// seq, or an actor's play, cycle or hold.
type action struct {
	Move   string       `json:"move"`
	Turn   string       `json:"turn"`
	Scale  string       `json:"scale"`
	Follow string       `json:"follow"` // a node glides along path, a smooth curve through the points
	Path   [][3]float64 `json:"path"`
	Orbit  *orbit       `json:"orbit"` // the camera swings around a point
	Wait   *float64     `json:"wait"`
	Fov    *float64     `json:"fov"`
	Show   string       `json:"show"`
	Hide   string       `json:"hide"`
	Par    []action     `json:"par"` // these run together
	Seq    []action     `json:"seq"` // these run one after another (inside a par, one lane)
	To     *[3]float64  `json:"to"`
	Dur    float64      `json:"dur"`
	Ease   string       `json:"ease"`

	Play   string     `json:"play"`   // an actor plays keyed poses
	Keys   []poseKey  `json:"keys"`   // with play: poses at moments of the move (t from 0 to 1, in degrees)
	Cycle  string     `json:"cycle"`  // an actor repeats a move: a base pose plus swinging joints
	Cycles float64    `json:"cycles"` // with cycle: how many repeats
	Base   rig.Pose   `json:"base"`   // with cycle: the pose it swings around, degrees
	Swing  []swing    `json:"swing"`
	Hold   string     `json:"hold"` // an actor keeps its pose for dur
	Travel *travel    `json:"travel"`
	Arc    *arc       `json:"arc"`  // a jump: the body lifts and returns
	Spin   *spin      `json:"spin"` // a flip or turn: a joint turns through whole rotations
	Marks  []markJSON `json:"marks"`
}

type poseKey struct {
	T    float64  `json:"t"`
	Pose rig.Pose `json:"pose"`
	Ease string   `json:"ease"`
}

type swing struct {
	Channel string  `json:"channel"`
	Amp     float64 `json:"amp"`   // degrees (or distance for a .x/.y/.z channel)
	Phase   float64 `json:"phase"` // in cycles; 0.25 turns sine into cosine
	Mult    float64 `json:"mult"`  // 2 swings twice as fast
	Wave    string  `json:"wave"`  // sine (default), half or abs
}

type travel struct{ Dist, From, To float64 }
type arc struct{ Height, From, To float64 }
type spin struct {
	Channel         string
	Turns, From, To float64
}
type markJSON struct {
	Kind  string  `json:"kind"`
	At    float64 `json:"at"`
	Span  float64 `json:"span"`
	Secs  float64 `json:"secs"`
	X     float64 `json:"x"`
	Z     float64 `json:"z"`
	AtEnd bool    `json:"atEnd"`
	Size  float64 `json:"size"`
}

type orbit struct {
	Center [3]float64 `json:"center"`
	Radius float64    `json:"radius"`
	Height float64    `json:"height"`
	From   float64    `json:"from"`
	To     float64    `json:"to"`
}

var eases = map[string]anim.Ease{
	"": nil, "linear": anim.Linear, "inQuad": anim.InQuad, "outQuad": anim.OutQuad, "inCubic": anim.InCubic, "outCubic": anim.OutCubic,
	"inOutSine": anim.InOutSine, "inOutQuad": anim.InOutQuad, "inOutCubic": anim.InOutCubic, "outBack": anim.OutBack,
	"outElastic": anim.OutElastic, "outBounce": anim.OutBounce,
}

func vec(a *[3]float64) geom.Vector { return geom.V(a[0], a[1], a[2]) }
