def point:
  type == "array" and length == 3
  and all(.[]; type == "number" and . >= -50 and . <= 50);

def displacement:
  if type != "object"
     or (keys - ["east","west","north","south","up","down"] | length != 0)
     or (all(.[]; type == "number" and . >= 0 and . <= 50) | not)
  then error("invalid displacement")
  else [
    ((.east // 0) - (.west // 0)),
    ((.north // 0) - (.south // 0)),
    ((.up // 0) - (.down // 0))
  ]
  end;

def resolve($base; $ball; $modes):
  if type != "object" or (.mode as $mode | $modes | index($mode) == null)
  then error("invalid camera action")
  elif .mode == "keep" and keys == ["mode"] then $base
  elif .mode == "set" and keys == ["mode","xyz"] and (.xyz | point) then .xyz
  elif (.mode == "move" or .mode == "ball") and keys == ["by","mode"] then
    (if .mode == "ball" then $ball else $base end) as $origin
    | (.by | displacement) as $delta
    | [range(3) | $origin[.] + $delta[.]]
  else error("camera action has unexpected fields")
  end;

if type != "object" or keys != ["aim","camera","height","rays","width"]
then error("invalid edit fields")
else {
  width, height, rays,
  at: (.camera | resolve($scene.front_at; $scene.ball_center; ["keep","set","move"])),
  look_at: (.aim | resolve($scene.front_look_at; $scene.ball_center; ["keep","set","move","ball"]))
}
end
