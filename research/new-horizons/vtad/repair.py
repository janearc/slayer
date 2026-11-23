import pathlib,struct,collections,numpy as np
D=np.dtype([('n','<f4',3),('v','<f4',(3,3)),('a','<u2')])
for p in pathlib.Path('manufacture/new-horizons-vtad').glob('*.stl'):
 raw=p.read_bytes();n=struct.unpack_from('<I',raw,80)[0]; data=np.frombuffer(raw,dtype=D,count=n,offset=84).copy(); faces=data['v']; es=collections.defaultdict(list)
 for i,f in enumerate(faces):
  v=[tuple(a) for a in f]
  for j in range(3):
   a,b=v[j],v[(j+1)%3]; key=tuple(sorted((a,b))); es[key].append((i,1 if (a,b)==key else -1))
 # Remove extra incident faces at non-manifold edges, in ascending face order.
 drop=set()
 for rows in es.values():
  if len(rows)>2:drop.update(i for i,_ in rows[2:])
 graph=collections.defaultdict(list)
 for rows in es.values():
  rows=[r for r in rows if r[0] not in drop]
  if len(rows)==2:
   (a,sa),(b,sb)=rows;graph[a].append((b,sa==sb));graph[b].append((a,sa==sb))
 orient={};groups=[]
 for i in range(n):
  if i in drop or i in orient:continue
  orient[i]=False; stack=[i];group=[]
  while stack:
   a=stack.pop();group.append(a)
   for b,flip in graph[a]:
    want=orient[a]^flip
    if b not in orient:orient[b]=want;stack.append(b)
    elif orient[b]!=want:drop.add(b)
  groups.append(group)
 for i,flip in orient.items():
  if flip:faces[i,[1,2]]=faces[i,[2,1]]
 # Ensure each closed connected component has outward signed volume.
 for group in groups:
  members=[i for i in group if i not in drop]; S=set(members); closed=True
  for rows in es.values():
   touching=[r for r in rows if r[0] in S]
   if len(touching)==1:closed=False;break
  if closed and members:
   t=faces[members].astype(np.float64); volume=np.einsum('ij,ij->i',t[:,0],np.cross(t[:,1],t[:,2])).sum()
   if volume<0:
    for i in members:faces[i,[1,2]]=faces[i,[2,1]]
 keep=np.array([i for i in range(n) if i not in drop]);fixed=data[keep];p.write_bytes(raw[:80]+struct.pack('<I',len(fixed))+fixed.tobytes());print(p.name,n,'kept',len(fixed),'dropped',len(drop))
