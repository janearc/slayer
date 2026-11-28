def scene_request($context):
  "Starting scene: " + ($context | tojson) + "\nRequest: " + .request;

{
  model: $model,
  temperature: 0,
  chat_template_kwargs: {enable_thinking: false},
  seed: 37,
  max_tokens: 128,
  stream: false,
  response_format: {
    type: "json_schema",
    json_schema: {name: "slayer_edit", strict: true, schema: $format}
  },
  messages: (
    [{
      role: "system",
      content: (if $action_prompt != "" then $action_prompt else ("Edit the Slayer scene using the five allowed fields. " +
        "Output one JSON object. width and height are film dimensions in pixels; " +
        "rays means the number of rays along each side of a pixel. at and look_at are the front camera's XYZ " +
        "location and target in metres. East/west change X, north/south change Y, and above/below or higher/lower change Z. East is +X, west is -X, north is +Y, south is -Y, up is +Z, and down is -Z. " +
        "Each request supplies its own starting scene. Use only the coordinates in that request. " +
        "For a relative move, new coordinate = original coordinate + signed displacement. South by d means new Y = original Y minus d, even when Y is already negative. " +
        "For a relative aim, start at the stated reference point and adjust only the named axis of look_at. " +
        "Copy all unchanged coordinates directly from front_at or front_look_at.") end)
    }]
    + ($examples | map(
      [{role: "user", content: scene_request($example_scene)},
       {role: "assistant", content: (.expected | tojson)}]) | add // [])
    + [{role: "user", content: scene_request($scene)}]
  )
}
| if $ollama then del(.chat_template_kwargs) + {reasoning_effort: "none"} else . end
