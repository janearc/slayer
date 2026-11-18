package oakland

import "github.com/janearc/puzzlemouth/yamlfile"

// gltfSchema is the strict wire boundary for the glTF subset Oakland reads.
// Unknown members are refused so unsupported glTF features cannot vanish
// while decoding into Go structs.
const gltfSchema = `{
  "$schema":"https://json-schema.org/draft/2020-12/schema",
  "type":"object","additionalProperties":false,
  "required":["asset","scenes","nodes","meshes","buffers","bufferViews","accessors"],
  "properties":{
    "asset":{"type":"object","additionalProperties":false,"required":["version"],"properties":{"version":{"const":"2.0"},"generator":{"type":"string"},"copyright":{"type":"string"}}},
    "scene":{"type":"integer","minimum":0},
    "scenes":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"properties":{"name":{"type":"string"},"nodes":{"type":"array","items":{"type":"integer","minimum":0}}}}},
    "nodes":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"properties":{"name":{"type":"string"},"mesh":{"type":"integer","minimum":0},"children":{"type":"array","items":{"type":"integer","minimum":0}},"translation":{"$ref":"#/$defs/vec3"},"rotation":{"type":"array","minItems":4,"maxItems":4,"items":{"type":"number"}},"scale":{"$ref":"#/$defs/vec3"}}}},
    "meshes":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["primitives"],"properties":{"name":{"type":"string"},"primitives":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["attributes"],"properties":{"attributes":{"type":"object","required":["POSITION"],"properties":{"POSITION":{"type":"integer","minimum":0},"NORMAL":{"type":"integer","minimum":0}},"additionalProperties":{"type":"integer","minimum":0}},"indices":{"type":"integer","minimum":0},"material":{"type":"integer","minimum":0},"mode":{"type":"integer"}}}}}}},
    "materials":{"type":"array","items":{"type":"object","additionalProperties":false,"properties":{"name":{"type":"string"}}}},
    "buffers":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["byteLength"],"properties":{"uri":{"type":"string"},"byteLength":{"type":"integer","minimum":1}}}},
    "bufferViews":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["buffer","byteLength"],"properties":{"buffer":{"type":"integer","minimum":0},"byteOffset":{"type":"integer","minimum":0},"byteLength":{"type":"integer","minimum":1},"byteStride":{"type":"integer","minimum":4,"maximum":252}}}},
    "accessors":{"type":"array","minItems":1,"items":{"type":"object","additionalProperties":false,"required":["bufferView","componentType","count","type"],"properties":{"bufferView":{"type":"integer","minimum":0},"byteOffset":{"type":"integer","minimum":0},"componentType":{"enum":[5121,5123,5125,5126]},"count":{"type":"integer","minimum":1},"type":{"enum":["SCALAR","VEC2","VEC3","VEC4","MAT2","MAT3","MAT4"]},"normalized":{"type":"boolean"},"min":{"type":"array","items":{"type":"number"}},"max":{"type":"array","items":{"type":"number"}}}}}
  },
  "$defs":{"vec3":{"type":"array","minItems":3,"maxItems":3,"items":{"type":"number"}}}
}`

// makeGltfSchema compiles the strict subset through Puzzlemouth's schema reader.
func makeGltfSchema() (*yamlfile.Schema, error) {
	return yamlfile.NewSchema("https://oakland.test/gltf-2.0-subset", []byte(gltfSchema))
}
