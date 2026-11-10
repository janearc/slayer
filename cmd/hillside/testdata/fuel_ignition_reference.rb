require 'bigdecimal'
require 'bigdecimal/math'
require 'json'
values={central:25,adjacent:125,corner:3425}.transform_values do |numerator|
 exponent=-BigDecimal(numerator.to_s)/BigDecimal('512')
 (BigMath.exp(exponent,80)*BigDecimal('900')).to_s('F')
end
record={purpose:'Authored Gaussian ignition fixture, not a published measurement.',
 geometry:'16x8 half-metre cells, continuous peak excess900K, radius1.6m. Squared-radius ratios at selected cell centres are exact25/512,125/512,3425/512.',
 arithmetic:'Independent Ruby BigDecimal/BigMath evaluation at requested80 decimal digits.',
 excessKelvin:values}
File.write('/tmp/slayer-fuel-ignition-reference.json',JSON.pretty_generate(record)+"\n")
values.each{|name,value|puts "#{name}: #{value}"}
