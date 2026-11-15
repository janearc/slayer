# HEVC encoder

The encoder streams top-down RGBA frames to a caller-selected ffmpeg executable. It requests VideoToolbox HEVC Main, an hvc1 tag, BT.709 metadata, TV range, height padding to a multiple of 16, a 48-frame keyframe interval, and faststart. The movie is first written to a sibling temporary file and then published; callers must send frames in order and close or abort the encoder.

The encoder does not mux audio. Encoder finalization has a timeout, and pipe writes can experience backpressure. The frame count and dimensions are validated before the child is started.

The specification's assertion that 72 frames contain three 24-frame GOPs conflicts with its 48-frame GOP requirement. Joining pieces and watching them in QuickTime remain separate validation tasks. The fake-process tests do not establish hardware encoding or playback.
