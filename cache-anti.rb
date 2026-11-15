require "json"
require "tmpdir"
require "open3"

path=File.expand_path("gpu/field_cache.go")
original=File.read(path)
[
 ["if cache.ready && cache.key == key {","if cache.ready {"],
 ["if cache.ready && cache.key == key {","if false && cache.ready && cache.key == key {"],
 ["key.lensKey = lens.RayKey()",'key.lensKey = "same"'],
 ["eye: input.eye, space: input.space,","eye: maths.Subspace{}, space: input.space,"]
].each do |before,after|
 abort "cache mutation anchor changed" unless original.scan(before).length==1
 Dir.mktmpdir("field-cache-anti") do |dir|
  file=File.join(dir,"field_cache.go")
  File.write(file,original.sub(before,after))
  overlay=File.join(dir,"overlay.json")
  File.write(overlay,JSON.generate({"Replace"=>{path=>file}}))
  output,status=Open3.capture2e("go","test","-tags=draw","-count=1","-overlay=#{overlay}","./gpu","-run=^TestGravityCompleteSeedCacheTracksCameraState$")
  unless !status.success? && output.include?("--- FAIL: TestGravityCompleteSeedCacheTracksCameraState") && output.include?("seed cache state")
   abort "missed #{before}:\n#{output}"
  end
  puts "rejected #{before}"
 end
end
