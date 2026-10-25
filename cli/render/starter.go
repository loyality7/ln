package render

// Starter is a working scene file, written by `ln new`.
const Starter = `{
  "width": 960,
  "height": 540,
  "fps": 24,
  "look": "ink",
  "light": {"sun": [-0.5, -0.6, 1], "ambient": 0.2, "shadows": true},
  "style": {"outline": 2.6, "wobble": 1.3},
  "camera": {"eye": [7, -9, 4.5], "target": [0, 0, 1], "fov": 36},
  "nodes": [
    {"name": "floor", "shape": "box", "args": [14, 14, 0.2], "at": [0, 0, -0.1], "paint": [0.82, 0.8, 0.74]},
    {"name": "cube", "shape": "box", "args": [1.4, 1.4, 1.4], "at": [-3, -1.3, 0.7], "paint": [0.85, 0.3, 0.2]},
    {"name": "ball", "shape": "sphere", "args": [0.8], "at": [3, 1.3, 0.8], "paint": [0.25, 0.5, 0.85]},
    {"name": "vase", "shape": "lathe", "profile": [[0, 0], [0.6, 0], [0.9, 0.8], [0.5, 1.6], [0.6, 2], [0, 2]], "smooth": 6, "at": [0, 3, 0], "paint": [0.9, 0.75, 0.25]},
    {"name": "lamp", "shape": "sphere", "args": [0.15], "at": [0, -3, 3], "lamp": {"power": 1.2, "radius": 8}, "paint": [1, 0.95, 0.7]}
  ],
  "timeline": [
    {"par": [
      {"move": "cube", "to": [3, -1.3, 0.7], "dur": 3, "ease": "inOutSine"},
      {"move": "ball", "to": [-3, 1.3, 0.8], "dur": 3, "ease": "inOutSine"},
      {"turn": "cube", "to": [0, 0, 3.14], "dur": 3, "ease": "inOutSine"},
      {"orbit": {"center": [0, 0, 1], "radius": 11, "height": 4.5, "from": -0.9, "to": 0.6}, "dur": 3}
    ]},
    {"wait": 0.5}
  ]
}
`
