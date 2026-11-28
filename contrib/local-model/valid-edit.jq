def in_range($minimum; $maximum):
  type == "number" and floor == . and . >= $minimum and . <= $maximum;
def point:
  type == "array" and length == 3 and all(.[]; type == "number" and . >= -50 and . <= 50);

type == "object"
and keys == ["at","height","look_at","rays","width"]
and (.width | in_range(1; 512))
and (.height | in_range(1; 512))
and (.rays | in_range(1; 8))
and (.at | point)
and (.look_at | point)
and (.at != .look_at)
