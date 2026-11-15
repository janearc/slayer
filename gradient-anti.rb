require "json"
require "tmpdir"
require "open3"

directory,status=Open3.capture2e("go","list","-f","{{.Dir}}","./maths/noise")
abort directory unless status.success?
source=File.join(directory.strip,"noise.go")
text=File.read(source)
before="return along.firstSign*offset[along.first] +"
abort "missing gradient sign mutation" unless text.scan(before).length==1
Dir.mktmpdir("slayer-gradient-anti") do |dir|
 changed=File.join(dir,"noise.go")
 overlay=File.join(dir,"overlay.json")
 File.write(changed,text.sub(before,"return offset[along.first] +"))
 File.write(overlay,JSON.generate({"Replace"=>{source=>changed}}))
 output,result=Open3.capture2e("go","test","-count=1",
  "-overlay=#{overlay}","./maths/noise",
  "-run=^TestGradientDirectionsPreserveExactValues$")
 unless !result.success? &&
  output.include?("--- FAIL: TestGradientDirectionsPreserveExactValues") &&
  output.include?("want")
  abort "gradient assertion did not reject sign loss\n#{output}"
 end
 puts "rejected gradient sign loss"
end

before="inner[dy][dz] = mix(uint64(j+dy) + under)"
abort "missing lattice mutation" unless text.scan(before).length==1
Dir.mktmpdir("slayer-lattice-anti") do |dir|
 changed=File.join(dir,"noise.go")
 overlay=File.join(dir,"overlay.json")
 File.write(changed,text.sub(before,
  "inner[dy][dz] = mix(uint64(j) + under)"))
 File.write(overlay,JSON.generate({"Replace"=>{source=>changed}}))
 output,result=Open3.capture2e("go","test","-count=1",
  "-overlay=#{overlay}","./maths/noise",
  "-run=^TestGradientMatchesOriginalLatticeHashes$")
 unless !result.success? &&
  output.include?("--- FAIL: TestGradientMatchesOriginalLatticeHashes") &&
  output.include?("want")
  abort "lattice assertion did not reject shared corner\n#{output}"
 end
 puts "rejected incorrect shared lattice corner"
end
