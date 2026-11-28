# Region light

A CPU or software-traversal GPU still can draw a rectangle as linear light.
The session checks the rectangle against the resolved full frame and keeps
the full camera dimensions. CPU retained rows use local row numbers, and
callers must key them by the rectangle's origin and dimensions.

Native GPU traversal and unsupported draw modes return a refusal.
Development and overlays can depend on neighbouring light outside the
rectangle. See [region comparison](../render/regioncompare/session-wiring.md)
for the diagnostic command.
