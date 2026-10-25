package cli

// topics are the reference pages behind `ln help <topic>`.
var topics = map[string]string{
	"looks": `LOOKS: how a 3D scene is drawn (set "look" in the scene, or pass -look to ln render)

  ink        black pen lines on white paper
  manga      thick and thin pen lines, comic style
  sketch     loose, wobbly hand-drawn lines
  technical  clean, even engineering lines
  colour     flat colours with ink lines ("sky": background colour)
  ascii      letters instead of lines ("ramp", "invert", "floor", "noEdges")
  plotter    plain vector paths for a pen plotter (.svg output)

For video made into art, ln art has its own looks: lines, colour, glow, glow-colour.
`,
	"scene": `SCENE FILE (JSON): ln render scene.json

Top level:  width, height, fps, look, style, lighting, camera, lights, gravity,
            nodes (the things), actors (jointed figures), captions, timeline (the story)
Shapes:     box sphere ellipsoid cylinder cone torus capsule annulus group
            lathe extrude tube mirror csg heightfield (a greyscale image) mesh (.obj or .stl)
Timeline:   move turn scale follow orbit wait fov show hide, and groups that run together (par) or in order (seq)
Lights:     point, spot or directional
Gravity:    bends light and lines around a heavy point (a black hole)

Unknown fields are errors, so a typo is caught instead of ignored.
Run  ln new  to write a working scene.json and start from it.
`,
	"drawing": `DRAWING FILE (JSON): ln draw drawing.json

Top level:  mode (ink or glow), width, height, paper [r,g,b], look (glow only), items
Position:   x goes right, y goes down, in pixels. Colours are [r,g,b] from 0 to 1.

Ink items:
  line, circle, ellipse, bezier   outlines (width, color, white)
  brush      pressure-varying pen stroke through pts (style: broad, dynamic, fine)
  fill       solid shape from pts
  hatch      parallel lines in a box (spacing, angle, width)
  woven      woven pen strokes in a box (cell, lines, width)
  halftone   print dots in a box (density 0-1, pitch, angle)

Glow items (mode "glow"):
  spot       soft round light: at [x,y], radius, color, power
  streak     light between two pts (width)
  path       a line of light through pts (radius)
  also circle, ellipse and bezier as light

Glow look: exposure, bloom (list of blur sizes), strength, vignette.
Run  ln new -type drawing  for a working example.
`,
}
