package scene

import (
	"github.com/loyality7/ln/ln/geom"
)

type ClipFilter struct {
	Matrix geom.Matrix
	Eye    geom.Vector
	Scene  *Scene
	Depth  bool // replace output Z with eye distance (used for ink line weight)
}

var ClipBox = geom.Box{geom.Vector{-1, -1, -1}, geom.Vector{1, 1, 1}}

func (f *ClipFilter) Filter(v geom.Vector) (geom.Vector, bool) {
	w := f.Matrix.MulPositionW(v)
	if !f.Scene.Visible(f.Eye, v) {
		return w, false
	}
	if !ClipBox.Contains(w) {
		return w, false
	}
	if f.Depth {
		w.Z = v.Distance(f.Eye)
	}
	return w, true
}
