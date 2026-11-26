struct SceneUniform {
    view_projection: mat4x4<f32>,
    model: mat4x4<f32>,
    edge: vec4<f32>,
}
@group(0) @binding(0) var<uniform> scene: SceneUniform;
struct VertexInput {
    @location(0) position: vec3<f32>,
    @location(1) normal: vec3<f32>,
    @location(2) colour: vec3<f32>,
    @location(3) model0: vec4<f32>,
    @location(4) model1: vec4<f32>,
    @location(5) model2: vec4<f32>,
    @location(6) model3: vec4<f32>,
}
struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) normal: vec3<f32>,
    @location(1) colour: vec3<f32>,
}
fn project(input: VertexInput) -> VertexOutput {
    var result: VertexOutput;
    let model = mat4x4<f32>(input.model0, input.model1, input.model2, input.model3);
    result.position = scene.view_projection * model * vec4<f32>(input.position, 1.0);
    result.normal = (model * vec4<f32>(input.normal, 0.0)).xyz;
    result.colour = input.colour;
    return result;
}
@vertex fn solid_vertex(input: VertexInput) -> VertexOutput { return project(input); }
@vertex fn edge_vertex(input: VertexInput) -> VertexOutput {
    var result = project(input);
    result.position.z = result.position.z - 0.0000002 * result.position.w;
    return result;
}
struct SolidOutput {
    @location(0) colour: vec4<f32>,
    @builtin(frag_depth) depth: f32,
}
@fragment fn solid_fragment(input: VertexOutput) -> SolidOutput {
    let key = max(dot(normalize(input.normal), normalize(vec3<f32>(-0.4, 0.8, 0.6))), 0.0);
    let linear = clamp(input.colour * (0.30 + 0.70 * key), vec3<f32>(0.0), vec3<f32>(1.0));
    var result: SolidOutput;
    result.colour = vec4<f32>(pow(linear, vec3<f32>(0.45454545)), 1.0);
    // One pixel of depth slope separates filled faces from their coplanar edges.
    let slope = max(abs(dpdx(input.position.z)), abs(dpdy(input.position.z)));
    result.depth = min(1.0, input.position.z + slope + 0.0000002);
    return result;
}
@fragment fn edge_fragment(input: VertexOutput) -> @location(0) vec4<f32> {
    return vec4<f32>(scene.edge.rgb, 1.0);
}
