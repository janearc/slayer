# Authored replay fixture, not a measurement of vegetation or terrain.
# PCG128/DXSM is documented by the Go Authors in math/rand/v2/pcg.go:
# https://go.googlesource.com/go/+/refs/tags/go1.26.10/src/math/rand/v2/pcg.go
# This separate implementation uses Ruby integers for one 128-bit update,
# rather than Go's limb arithmetic. The accompanying provenance pins the
# inspected standard-library sources and the explicit fixture parameters.
require 'json'

mask64=(1<<64)-1
mask128=(1<<128)-1
multiplier=(2549297995355413924<<64)|4865540595714422341
increment=(6364136223846793005<<64)|1442695040888963407
state=(7<<64)|1
uniform=lambda do
 state=(state*multiplier+increment)&mask128
 high,low=state>>64,state&mask64
 high^=high>>32
 high=(high*0xda942042e4dd58b5)&mask64
 high^=high>>48
 high=(high*(low|1))&mask64
 (high&((1<<53)-1)).to_f/(1<<53)
end
places=[]
attempts=0
form_seeds=[28,29,30,31,28,29,30,31,28,29,30,31]
mix=lambda do |seed,index|
 word=seed^(((index+1)*0x9e3779b97f4a7c15)&mask64)
 word=((word^(word>>30))*0xbf58476d1ce4e5b9)&mask64
 word=((word^(word>>27))*0x94d049bb133111eb)&mask64
 word^(word>>31)
end
while places.size<12
 attempts+=1
 raise 'reference did not find its twelve specimens' if attempts>4096
 east=1001.0+2+146*uniform.call
 north=8999.0-2-96*uniform.call
 column=((east-1001.0)/2).round
 row=((8999.0-north)/2).round
 height=(column+1000*row)/8.0
 next unless height>2000 && height<4000
 yaw= mix.call(form_seeds.fetch(places.size),places.size)%3600
 places<<{east:east,north:north,height:height,yaw_tenths:yaw}
end
puts JSON.pretty_generate({seed:[7,1],attempts:attempts,places:places})
