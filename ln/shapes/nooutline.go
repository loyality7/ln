package shapes

import "github.com/loyality7/ln/ln/geom"

// NoOutline hides a shape's own line paths so it only shows through shading
// (hatching). Good for terrain and other large surfaces.
type NoOutline struct{ Shape }

func (NoOutline) Paths() geom.Paths { return nil }
