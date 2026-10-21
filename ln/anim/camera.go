package anim

import (
	"math"

	"github.com/loyality7/ln/ln/geom"
	"github.com/loyality7/ln/ln/scene"
)

type Camera struct {
	Eye, Center, Up geom.Vector
	Fovy            float64
}

// Orbit circles center at the given radius and height; angle is in radians.
func Orbit(center geom.Vector, radius, height, angle, fovy float64) Camera {
	eye := center.Add(geom.Vector{X: math.Cos(angle) * radius, Y: math.Sin(angle) * radius, Z: height})
	return Camera{eye, center, geom.Vector{Z: 1}, fovy}
}

// Render draws s from this camera.
func (c Camera) Render(s *scene.Scene, width, height, step float64) geom.Paths {
	return s.Render(c.Eye, c.Center, c.Up, width, height, c.Fovy, 0.1, 1000, step)
}
