"""Group repaired NASA texture meshes by a triangle-centre texture sample.

This is a coarse material approximation, not texture/UV preservation. Input
samples come from nh-textures.mjs after applying original GLB node transforms.
Triangle keys ignore orientation so a winding repair does not change a match.
"""
import collections,json,pathlib,struct,numpy as np
root=pathlib.Path('manufacture/new-horizons-vtad')
dtype=np.dtype([('n','<f4',3),('v','<f4',(3,3)),('a','<u2')])
def key(v):
 return tuple(sorted(tuple(float(x) for x in row) for row in np.asarray(v,dtype='<f4')))
def group(rgb):
 r,g,b=rgb; mean=(r+g+b)/3
 if r>1.22*b and r>70:return 'warm'
 if mean<55:return 'black'
 if mean<105:return 'dark'
 if mean<170:return 'silver'
 return 'white'
for name in ('tex_01','tex_02'):
 data=json.loads(pathlib.Path('/private/tmp/nh-uv-'+name+'.json').read_text());lookup=collections.defaultdict(list)
 for row in data:lookup[key(row['tri'])].append(group(row['rgb']))
 p=root/(name+'-0.stl');raw=p.read_bytes();n=struct.unpack_from('<I',raw,80)[0];arr=np.frombuffer(raw,dtype=dtype,count=n,offset=84)
 bins=collections.defaultdict(list);unmatched=0
 for i,face in enumerate(arr):
  k=key(face['v']); candidates=lookup[k]
  if not candidates: unmatched+=1;continue
  bins[candidates.pop(0)].append(i)
 if unmatched:raise RuntimeError(f'{name}: {unmatched} unmatched triangles')
 for label,indices in bins.items():
  out=arr[np.array(indices,dtype=int)];dest=root/f'{name}-{label}.stl';dest.write_bytes(raw[:80]+struct.pack('<I',len(out))+out.tobytes())
  print(dest,len(out))
 p.unlink()
