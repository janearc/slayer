require "json"
require "tmpdir"
require "open3"

mutants = [
 ["gpu/volumes.go", 'given["info"] = wholes(1, 0, boxedKind, material)',
  'given["info"] = wholes(0, 0, boxedKind, material)',
  "./gpu", "TestNativeGiantDensityRemainsSmoothAtBoundedGridResolution",
  "bounded giant density developed voxel rings"],
 ["gpu/sampled.go", "nil, &spent, volumeCells...)", "nil, &spent)",
  "./gpu", "TestNativeRendererCarriesVolumeBoundToSampling",
  "renderer dropped volume-grid bound"],
 ["gpu/parts.go", "encoded.plane(space, material, clear)",
  "encoded.sphere(space, 1, material, clear)", "./gpu",
  "TestNativePlaneIsVisibleWithoutOtherGeometry",
  "analytic plane missing from GPU buffer"],
 ["render/cells/cells.go", "scene.sample(scene.world",
  "trace.Sampled(scene.world", "./render/cells",
  "TestNativeSamplerUsesSharedSceneTimingAndDevelopment",
  "chosen sampler not used"]
]
mutants.each do |file,before,after,package,test,failure|
 directory,located=Open3.capture2e("go","list","-f","{{.Dir}}",
  "./"+File.dirname(file),chdir:__dir__)
 abort directory unless located.success?
 source=File.join(directory.strip,File.basename(file))
 text=File.read(source)
 abort "mutation does not match #{file}" unless text.scan(before).length==1
 Dir.mktmpdir("slayer-native-anti") do |dir|
  changed=File.join(dir,File.basename(file))
  overlay=File.join(dir,"overlay.json")
  File.write(changed,text.sub(before,after))
  File.write(overlay,JSON.generate({"Replace"=>{source=>changed}}))
  output,status=Open3.capture2e("go","test","-tags=draw","-count=1",
   "-overlay=#{overlay}",package,"-run=^#{test}$",chdir:__dir__)
  unless !status.success? && output.include?("--- FAIL: #{test}") &&
   output.include?(failure)
   abort "#{test} missed intended counterexample\n#{output}"
  end
  puts "rejected #{test} counterexample"
 end
end
