require "json"
require "tmpdir"
require "open3"

path = File.expand_path("gpu/gravity_test.go")
original = File.read(path)
[
 ["no gravity", '"2*mass*r*r2*overD", "0.0f"'],
 ["no spin", '"kerr_rate_at(x,p,mass,spin,kerr_radius(x.yzw,spin))", "kerr_rate_at(x,p,mass,0,kerr_radius(x.yzw,0))"'],
 ["no integration", '"x+=change.x;p+=change.p;", "x+=float4(0);p+=change.p;"'],
 ["no disc guard", '"if (x.z==0 && d<=0) return 0;", ""',
  "TestGravityMetalCriticalAndBackwardSteps", "ring-disc branch differs"]
].each do |name, replacement, test, assertion|
 test ||= "TestGravityMetalStepsMatchCelestial"
 assertion ||= "gravity step mismatch"
 Dir.mktmpdir("gravity-anti") do |dir|
  changed = original.sub('"math"', '"math"' + "\n\t\"strings\"")
  changed = changed.gsub("hole.MetalLibrary+gravityProbe",
   "strings.ReplaceAll(hole.MetalLibrary, #{replacement})+gravityProbe")
  # gofmt puts spaces around the concatenation.
  changed = changed.gsub("hole.MetalLibrary + gravityProbe",
   "strings.ReplaceAll(hole.MetalLibrary, #{replacement}) + gravityProbe")
  abort "mutation missing" if changed == original
  file = File.join(dir,"gravity_test.go")
  File.write(file,changed)
  overlay = File.join(dir,"overlay.json")
  File.write(overlay,JSON.generate({"Replace"=>{path=>file}}))
  out,status=Open3.capture2e("go","test","-tags=draw","-count=1","-overlay=#{overlay}","./gpu","-run=^#{test}$")
  unless !status.success? && out.include?("--- FAIL: #{test}") && out.include?(assertion)
   abort "missed #{name}:\n#{out}"
  end
  puts "rejected #{name}"
 end
end
