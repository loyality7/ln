package stage

import "github.com/loyality7/ln/ln/model"

// size is the radius of a mesh's bounding box, used so a lamp does not shadow itself.
type meshSize interface{ size() float64 }

func sizeOf(m *model.Mesh) float64 {
	if m == nil {
		return 0.3
	}
	b := m.BoundingBox()
	return b.Max.Sub(b.Min).Length() / 2
}
