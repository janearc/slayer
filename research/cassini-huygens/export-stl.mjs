import { NodeIO } from '@gltf-transform/core';
import { ALL_EXTENSIONS } from '@gltf-transform/extensions';
import draco from 'draco3dgltf';
import { writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
const io=new NodeIO().registerExtensions(ALL_EXTENSIONS).registerDependencies({'draco3d.decoder':await draco.createDecoderModule(),'draco3d.encoder':await draco.createEncoderModule()});
const doc=await io.read(process.argv[2]); const dir=process.argv[3]; mkdirSync(dir,{recursive:true});
for (const node of doc.getRoot().listNodes()) {
 const mesh=node.getMesh();if(!mesh || node.getName()==='_root') continue;const M=node.getWorldMatrix();
 for (const [i,prim] of mesh.listPrimitives().entries()) {
  const P=prim.getAttribute('POSITION').getArray(), I=prim.getIndices()?.getArray(); const count=I?I.length:P.length/3;
  const out=Buffer.alloc(84+50*count/3);out.writeUInt32LE(count/3,80);let lo=[Infinity,Infinity,Infinity],hi=[-Infinity,-Infinity,-Infinity];
  for (let j=0;j<count;j++) {const idx=I?I[j]:j,x=P[3*idx],y=P[3*idx+1],z=P[3*idx+2];const v=[M[0]*x+M[4]*y+M[8]*z+M[12],M[1]*x+M[5]*y+M[9]*z+M[13],M[2]*x+M[6]*y+M[10]*z+M[14]];for(let k=0;k<3;k++){out.writeFloatLE(v[k],84+50*Math.floor(j/3)+12+12*(j%3)+4*k);lo[k]=Math.min(lo[k],v[k]);hi[k]=Math.max(hi[k],v[k]);}}
  const f=node.getName().replace(/[^a-zA-Z0-9_-]/g,'_')+'-'+i+'.stl';writeFileSync(join(dir,f),out);console.log(f,prim.getMaterial()?.getName(),count/3,lo,hi);
 }
}
