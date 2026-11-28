# Editing a Slayer scene with a local model

A small local model can translate a plain-language request into a film
and camera edit. This fixture gives it the front camera location, aim
and ball centre. The model emits a constrained JSON object. The
resolver applies the edit to a known scene and Slayer assesses and
draws a CPU preview.

Each camera action can keep a point, set an absolute position, or move
by named directions in metres. Aim can also start from the ball
centre. For example, `{"mode":"move","by":{"east":2,"up":3}}` moves
a point two metres east and three metres up. The resolver handles
coordinate signs and arithmetic. It validates the action fields and
the final values before Slayer sees the edited scene. The fixture
supplies the input scene, renderer, camera and output path.

## Run the fixture

Use `jq`, Mike Farah `yq` v4, `curl` and a Slayer binary with
`assess` and `draw`. Put Slayer on your path or set `SLAYER_BIN`
to its absolute path. From the Slayer checkout:

```sh
cd contrib/local-model
jq -c '.[0].expected' eval.json | sh assess.sh
```

Use your own local model server with OpenAI-compatible chat completions
and JSON Schema response formatting. The default endpoint is
`http://127.0.0.1:8080`. With several models on the server, set
`LOCAL_MODEL` to the desired model ID. For example, with an existing
Ollama installation and model:

```sh
export LOCAL_ENDPOINT=http://127.0.0.1:11434 LOCAL_MODEL=qwen3:0.6b
export SLAYER_ACTIONS=1 WITH_EXAMPLES=1
sh run.sh
SLAYER_CASES=confirm.json sh run.sh
SLAYER_SCENE=offset-scene.yaml SLAYER_CASES=transfer.json sh run.sh
```

Each run writes its requests, raw model responses, resolved edits,
assessment logs, previews and scores to a new local directory. The
`accepted` field requires assessment and drawing to succeed. A
`pass` also requires matching the requested values. Wall time covers
the request to the model server. Omit `SLAYER_ACTIONS=1` to compare
with a prompt that asks the model for final absolute coordinates.

## What the benchmark shows

The nine cases cover two starting scenes and hold the requests apart
from the four teaching examples. A local 523 MB Qwen3 0.6B rendered
nine of nine cases in both modes. It matched three requests with
absolute coordinates and eight with named directions. The remaining
miss aimed at the ball centre when asked to aim half a metre above
it. These are smoke tests for camera and film edits.

For further experiments, keep new scenes and requests out of the
teaching examples and score them using the same renderer. The saved
messages can also be used to compare a Jacobian lens readout on a
matching inspectable checkpoint. Compare lens-guided changes against
a baseline with the same training budget. The fixture uses a model
and server supplied by the operator.
