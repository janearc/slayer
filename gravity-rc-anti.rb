require "json"
require "tmpdir"
require "open3"

size_guard = "w, h, err := checkedFineSize(width, height, side)\n\tif err != nil {\n\t\treturn nil, spent, err\n\t}"
[
  ["gpu/field.go", "append(encoded.inputs(), params.bytes)",
   "append(encoded.inputs()[:15], params.bytes)",
   "TestGravityRCSceneBuffers", "want 17 scene buffers"],
  ["gpu/sampled.go", 'return nil, nil, unsupported{"gravity in retained or native rendering"}',
   "return nil, nil, nil", "TestGravityRCUnsupportedModesRefuseFields",
   "retained mode silently ignored gravity"],
  ["gpu/field.go", size_guard,
   "w, h, err := checkedFineSize(width, height, side)\n\t_ = err",
   "TestGravityRCRejectsOverflowingFrameSizes", "reached encoding"]
].each do |path, before, after, test, failure|
  source = File.expand_path(path)
  original = File.read(source)
  abort "counterexample anchor changed" unless original.scan(before).length == 1
  Dir.mktmpdir("slayer-rc-anti") do |dir|
    changed = File.join(dir, File.basename(source))
    File.write(changed, original.sub(before, after))
    overlay = File.join(dir, "overlay.json")
    File.write(overlay, JSON.generate({"Replace" => {source => changed}}))
    output, status = Open3.capture2e("go", "test", "-p=1", "-tags=draw",
      "-count=1", "-overlay=#{overlay}", "./gpu", "-run=^#{test}$")
    unless !status.success? && output.include?("--- FAIL: #{test}") && output.include?(failure)
      abort "counterexample was not rejected as intended:\n#{output}"
    end
    puts "rejected #{test} counterexample"
  end
end
