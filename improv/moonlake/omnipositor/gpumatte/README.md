# Experimental GPU matte

The Metal matte computes the moon reference's maximum and thresholded channel means, the optional water-mask maximum, the per-pixel sea blend, and the optional foreground addition. RGB input and output use the bottom-up PFM row order used by the CPU matte. The moon reference may have dimensions different from the output frame.

The implementation uses float32 GPU reductions and blending, unlike the CPU's float64 accumulations. It rejects invalid geometry, nonfinite values, zero water masks, negative masks and positive subnormal mask samples rather than silently claiming CPU-equivalent behavior on those inputs. Its one-GiB allocation limit covers its Metal buffers, not total process memory.

The API currently creates a Metal session and buffers for each call. Historical-frame equivalence, 4K performance, cross-device behavior, and persistent GPU buffers remain unverified.
