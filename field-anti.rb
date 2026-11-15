require "json"
require "tmpdir"
require "open3"

path=File.expand_path("gpu/field.go")
original=File.read(path)
[
 ["traceSource","float shift=world.energy/denominator;","float shift=1;"],
 ["fieldSource","marchKeep=straight.keep;","marchKeep=keep;"],
 ["fieldSource","p=kerr_arriving(x.yzw,field_direction(f,way),f[0].x,f[0].y);","p=float4(0);"],
 ["fieldSource","kerr_step_held_from(nextX,nextP,xError,pError,rate,","kerr_step(nextX,nextP,"],
 ["traceSource","if (all(colour == float3(0))) { return sum; }","if (false) {return sum;}",
  "TestGravityEmissiveVolumeDoesNotAskForLampShadows"],
 ["traceSource","if (hit.distance > far)", "if (false)",
  "TestGravitySegmentStopsAtItsEnd"],
 ["go","ops.encode(world, nil, eye, nil, at, w, h, cells...)",
  "ops.encode(world, nil, eye, lens, at, w, h, cells...)",
  "TestGravityCompleteDoesNotGenerateTheLensTwice"]
].each do |variable,before,after,test|
 test ||= "TestGravityComplete"
 Dir.mktmpdir("field-anti") do |dir|
  if variable=="go"
   abort "Go mutation anchor changed" unless original.scan(before).length==1
   changed=original.sub(before,after)
  else
   changed=original+"\nfunc init(){ #{variable}=strings.ReplaceAll(#{variable},#{before.to_json},#{after.to_json}) }\n"
  end
  file=File.join(dir,"field.go")
  File.write(file,changed)
  overlay=File.join(dir,"overlay.json")
  File.write(overlay,JSON.generate({"Replace"=>{path=>file}}))
  output,status=Open3.capture2e("go","test","-tags=draw","-count=1","-overlay=#{overlay}","./gpu","-run=^#{test}")
  unless !status.success? && output.include?("--- FAIL: #{test}") &&
    (output.include?("the gpu's light") || output.include?("pixels' depth") || output.include?("GPU ring differs from CPU") || output.include?("segment queried past its end") || output.include?("field generated lens rays more than once"))
   abort "missed #{before}:\n#{output}"
  end
  puts "rejected #{before}"
 end
end
