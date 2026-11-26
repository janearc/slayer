require 'json'
require 'fileutils'

binary, destination, selected_name, selected_seed = ARGV
abort 'need producer, output directory, kind and seed' unless binary && destination && selected_name && selected_seed
FileUtils.mkdir_p(destination)
names = %w[city-apartment city-clustered-10000 city-clustered-green-10000
           city-clustered-night-10000 city-grid-10000 city-rowhouse city-tower]
abort 'unknown kind or seed' unless names.include?(selected_name) && %w[5 11].include?(selected_seed)
names = [selected_name]
limit_kib = 4 * 1024 * 1024
free_kib = 30 * 1024 * 1024
names.each do |name|
  [selected_seed.to_i].each do |seed|
    key = "#{name}-#{seed}-demo"
    result = File.join(destination, "#{key}.txt")
    next if File.exist?(result)
    available = `df -k #{destination}`.lines.last.split[3].to_i
    abort "disk floor reached before #{key}" if available < free_kib
    stdout = File.join(destination, "#{key}.pending")
    stderr = File.join(destination, "#{key}.stderr")
    env = {'REFERENCE_NAME' => name, 'REFERENCE_SEED' => seed.to_s, 'REFERENCE_LEVEL' => 'demo'}
    started = Process.clock_gettime(Process::CLOCK_MONOTONIC)
    pid = Process.spawn(env, binary, out: stdout, err: stderr)
    peak = 0
    reason = nil
    status = nil
    loop do
      waited = Process.waitpid2(pid, Process::WNOHANG)
      if waited
        status = waited[1]
        break
      end
      elapsed = Process.clock_gettime(Process::CLOCK_MONOTONIC) - started
      rss = `ps -o rss= -p #{pid}`.strip.to_i
      peak = [peak, rss].max
      available = `df -k #{destination}`.lines.last.split[3].to_i
      reason = 'wall limit' if elapsed > 120
      reason = 'RSS limit' if rss > limit_kib
      reason = 'disk floor' if available < free_kib
      if reason
        Process.kill('KILL', pid)
        _, status = Process.waitpid2(pid)
        break
      end
      sleep 0.2
    end
    seconds = Process.clock_gettime(Process::CLOCK_MONOTONIC) - started
    output = File.read(stdout)
    valid = status.success? && output.match?(/\A#{Regexp.escape(name)} #{seed} demo [0-9a-f]{64}\n\z/)
    if valid
      File.rename(stdout, result)
      File.delete(stderr) if File.zero?(stderr)
    end
    record = {name: name, seed: seed, detail: 'demo', seconds: seconds.round(3),
              sampled_peak_kib: peak, ok: valid, reason: reason || (valid ? nil : "exit #{status.exitstatus} or bad row")}
    File.open(File.join(destination, 'progress.jsonl'), 'a') { |log| log.puts(JSON.generate(record)); log.flush; log.fsync }
    puts JSON.generate(record)
    $stdout.flush
    abort 'capture stopped; incomplete case preserved' unless valid
  end
end
