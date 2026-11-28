# Bioform pose clips

A pose clip samples an authored sequence on a particular skeleton.
Any maker can use it once it provides a rest hierarchy and named joints.
The rig supplies each bone's head and parent. The clip carries descendants
with their parents and turns each bone around its head.

Use the same rest skeleton for the clip and its skin binding.
`Skin.RestSkeleton()` gives a copy suitable for authoring. Each key is
a complete pose. A joint absent from a key relaxes to its rest rotation
relative to its parent at that time. To hold a joint through several keys,
carry its turn into each one. Build keys with `Pose.With` in the rig's rest
axes or with `Joints.Turned` when the rig supplies joint axes and limits.

```go
middle := skeleton.Pose{}.With(joint, skeleton.NewTurn(axis, 0.45))
clip, err := rig.NewPoseClip([]skeleton.PoseKey{
    {At: 0},
    {At: time.Second, Pose: middle},
    {At: 2 * time.Second},
})
if err != nil {
    return err
}
posed, err := skin.Pose(clip.At(elapsed))
if err != nil {
    return err
}
vertices := posed.Points()
// Use vertices with the original triangle indices and rest paint chart.
frame, found := posed.Frame(joint)
if !found {
    return fmt.Errorf("missing attachment bone %q", joint)
}
placement := frame.Place(restPlacement)
// Use placement for an object that should follow that bone.
```

`At` takes elapsed time since the start of the clip. The first key is at
zero. Keys must be strictly increasing and may use the same pose for a
hold. Sampling holds the first or last pose outside the authored times
and returns the authored pose at each key. `Duration` is the time of the
last key. Use a single key for a still.

Between keys, the clip interpolates each joint's rotation before applying
the hierarchy. This keeps a bone at the same distance from any point it
carries rigidly. Rotations, including the whole-body rotation set with
`Pose.Carried`, follow the shortest arc. Add intermediate keys to control
the direction of large turns and to author a spin. Root translation is
along a straight line in the skeleton's placement space. The default
samples each span at a constant fraction of elapsed time. Call
`clip.WithSmoothTiming()` to ease each span with zero linear and angular
velocity at its endpoints. Use it for gestures that settle at each key.
Repeatable sampling lets a caller evaluate the same clip at frame times or at multiple shutter times.

`Sample(elapsed)` supplies named frames directly when the maker has no
bound skin. `Frame.Apply` carries a point in the skeleton's rest space.
`Frame.Direction` carries a direction, such as an aim vector.
`Frame.Place` carries a rigid object placement and preserves its scale.
`Skeleton.NewSocket` attaches a placement to a named bone using an
offset from its rest head. `Socket.Place` reads the resulting placement
from a sample's named bone frames. For a landmark attached to a deformed
skin surface, use the matching
triangle in `Points` and the landmark's barycentric coordinates.

Clip timing and joint turns are authored. Interpolation has no joint
limit enforcement between keys. Contact, collision, balance, gait,
material deformation beyond skin weights, and loose attachments require
their own models. The clip is a Go API. Each animal maker supplies the
appropriate bones, keys and skin before a scene can play it.

The Slayer set reader accepts [authored pose clips](../../doc/model/bioform/pose-clips.md)
for the native human and orca. The Go clip and socket APIs work with any
maker that supplies a rest skeleton and sampled frames.
