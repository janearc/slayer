# Experimental GPU develop

The Metal developer applies exposure, bloom, the luminance shoulder and sRGB quantization to top-down float32 RGB frames. `DevelopWithExposure` changes exposure on an existing developer, allowing the hybrid compositor to retain its session and buffers while a cue fades the exposure. `Close` releases those resources.

The implementation limits its aggregate buffers to one GiB and rejects invalid exposure or nonfinite and excessively large exposed samples. It uses float32 and compensated arithmetic where necessary, while the CPU implementation uses float64 intermediates. Its numerical behavior, 4K throughput and output against the historical chain require measurement; it is not yet a claim of milestone acceptance.
