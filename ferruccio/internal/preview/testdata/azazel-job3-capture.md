# Owner wire capture

Original retained status capture SHA256: 837da1595c2162a1cb5109a7266a82a15be2e785ce9bcebe192ecbc0efb87485

Original submitted YAML SHA256: a128dd286a0fee5882ec335a41849333c2a9fe3f3f17ee00362cfd1881426756

Only sequence 3 and the root version/paused fields were retained. The actual
submitted YAML was decoded and compared byte-for-byte as canonical JSON with
that job's request before normalization. Other owners' jobs were discarded.

Replacements: request.agent becomes preview-owner in both fixtures. All unique
absolute path strings across both fixtures are sorted lexically; the nth path
becomes /capture/path-N, consistently in argv, inputs and fingerprint records.
In paused text, the participant label after existing becomes earlier-owner,
and the label preceding owns dispatch becomes queue-operator. Other wire
fields, values, times, hashes, state and executing flags are preserved.

The unnormalized originals remain outside Git in the retained render evidence.
No queue command, producer or publisher ran to create these fixtures.
