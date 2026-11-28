
struct SplitWorldInput {
    @location(0) high: vec3<f32>,
    @location(1) normal: vec3<f32>,
    @location(2) colour: vec3<f32>,
    @location(3) low: vec3<f32>,
}
@vertex fn split_solid_vertex(input: SplitWorldInput) -> VertexOutput {
    var result: VertexOutput;
    let high = input.high - scene.eye_parts[0].xyz;
    let low = input.low - scene.eye_parts[1].xyz;
    result.position = scene.view_projection * vec4<f32>(high + low, 1.0);
    result.normal = input.normal;
    result.colour = input.colour;
    return result;
}
@vertex fn split_edge_vertex(input: SplitWorldInput) -> VertexOutput {
    var result: VertexOutput;
    let high = input.high - scene.eye_parts[0].xyz;
    let low = input.low - scene.eye_parts[1].xyz;
    result.position = scene.view_projection * vec4<f32>(high + low, 1.0);
    result.position.z = result.position.z - 0.0000002 * result.position.w;
    result.normal = input.normal;
    result.colour = input.colour;
    return result;
}
