# Place a posed bioform in a scene

`assembly.NewAnimation` pairs a pose clip with a skin binding on the
same ordered rest skeleton. The caller supplies the original triangle
indices and the surface material. The rest-space paint coordinates
default to the skin's bound points. Supply a separate paint chart in the
same vertex order if the maker needs one.

```go
travel := thing.Keyed{
    Frames: []thing.Keyframe{
        {Share: 0, At: maths.V(0, 0, 0)},
        {Share: 1, At: maths.V(0, 0, 4)},
    },
    Length: 4 * time.Second,
    Ease:   "linear",
    Hold:   true,
}
figure, err := assembly.NewAnimation(assembly.AnimationInput{
    Clip: clip, Skin: skin, Faces: faces, Material: material, Root: travel,
})
if err != nil {
    return nil, err
}
sample, err := figure.At(sceneTime)
if err != nil {
    return nil, err
}
surface, err := sample.Surface()
if err != nil {
    return nil, err
}
socket, err := sample.Attachment(assembly.BoneAttachment{
    Bone: boneName,
    Rest: markerRestPlacement,
})
if err != nil {
    return nil, err
}
marker.Motion = thing.Still(socket)
return []thing.Thing{surface, marker}, nil
```

At each scene time, the root is sampled once through `thing.Motion`.
The pose clip reads the same scene time measured from `AnimationInput.Start`.
The clip holds its endpoint poses outside its range. The root motion
continues to follow its own timing and rules. In the example the root
travels linearly and holds its last position.

Between pose keys, rotations take the shortest quaternion arc, using a
normalized linear blend for nearby orientations. The translation in
`Pose.Carried` follows a straight path in figure space. Use
`WithSmoothTiming` on the clip for eased motion that settles at each key. The scene root encloses that
figure-space movement and can apply its own rotation or uniform scale.

Skin vertices deform in figure space. `Surface` puts the sampled root
on the resulting scene thing and retains the paint chart and indices.
`Attachment` uses the same bone frames as the skin. It carries an
object's rest placement with its named bone and then applies the scene
root. The returned socket is already in scene space. `Root` gives that
same scene placement for callers that also position or frame the figure.

Scene code supplies a fresh sample when the scene time changes. The
surface and marker above are fixed snapshots for one render time. The
mesh constructor can reject a deformed surface with collapsed faces.
The rest-space paint path currently needs the CPU renderer.

Placement and poses describe authored motion. There is no foot contact
or locomotion solver to keep a stance planted during travel, and no
collision, muscle, or volume conservation model in this composition.

For a maker that adds tissue or builds several parts, keep that maker's
native assembly. `primates.Human.WithClip` pairs the clip with the native
skin. At each time the maker uses one sample for the skin, the head frame
that carries the eyes, and the triangles that carry the groom. The
[studio reach](../../doc/native-human-study.md#timed-reach) demonstrates
this path in a Slayer set. Timed makers can use
`manufacture.PrepareDeformation` to retain rest data between frames.
