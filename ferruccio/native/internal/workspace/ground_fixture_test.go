package workspace

const groundEditFixture = `$schema: https://github.com/janearc/slayer/schemas/ecosystem-v2.json
scene:
  name: Ground editing
  setting: ground
  light: moon
  timeofday: 2200
  terrain: {from: fixture, file: fixture-plain.tif, west: 1001, north: 8999, across: 150, down: 100, step: 2}
  film: {width: 24, height: 16, rays: 1, exposure: 100000}
  cameras:
    view:
      orientation: {point_to: stone, alpha: -25}
      location: {loc_relative: {object: stone, standoff_x: 20, standoff_y: 0, standoff_z: 10}}
  set:
    things:
      stone: {kind: yurt, seed: 4, east: 1050, north: 8990, heading: {value: 30, unit: deg}}
`
