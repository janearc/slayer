require "json"
require "tmpdir"
require "open3"

mutants = [
 ["trace/volumes.go", "if colour == (maths.Light{}) {", "if false {",
 "./trace", "TestEmissiveVolumeDoesNotTraceZeroScatteringShadows",
 "zero-scattering volume asked lamps"],
 ["flame/sampled.go", "return densityMix(c, low, high)",
 "_ = c; _ = low; _ = high; return s.body.Density(p)",
 "./flame", "TestSampledFlameCachesDensityAndKeepsItsOpticalProperties",
 "density recomputed instead of cached"],
 ["flame/sampled.go", "return densityMix(c, low, high)",
 "_ = c; _ = low; _ = high; return 0",
 "./trace", "TestSampledFlameRetainsTerminalImageQuality",
 "sampled flame lost terminal image quality"]
]
mutants.each do |file,before,after,package,test,failure|
  directory,located=Open3.capture2e("go","list","-f","{{.Dir}}",
  "./"+File.dirname(file),chdir:__dir__)
 abort directory unless located.success?
 source=File.join(directory.strip,File.basename(file))

 text=File.read(source)
 abort "mutation no longer matches #{file}" unless text.scan(before).length==1
 Dir.mktmpdir("slayer-sampling-anti") do |dir|
  changed=File.join(dir,File.basename(file))
  overlay=File.join(dir,"overlay.json")
  File.write(changed,text.sub(before,after))
  File.write(overlay,JSON.generate({"Replace"=>{source=>changed}}))
  output,status=Open3.capture2e("go","test","-tags=draw","-count=1","-overlay=#{overlay}",
   package,"-run=^#{test}$",chdir:__dir__)
  unless !status.success? && output.include?("--- FAIL: #{test}") &&
   output.include?(failure)
   abort "#{test} did not reject intended mutation\n#{output}"
  end
  puts "rejected #{test} counterexample"
 end
end
