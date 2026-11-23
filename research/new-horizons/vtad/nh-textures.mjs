import {NodeIO} from '@gltf-transform/core';
import {ALL_EXTENSIONS} from '@gltf-transform/extensions';
import sharp from 'sharp';
import {writeFileSync} from 'node:fs';
const io=new NodeIO().registerExtensions(ALL_EXTENSIONS);
const doc=await io.read(process.argv[2]);
for(const node of doc.getRoot().listNodes()){
 const mesh=node.getMesh();if(!mesh)continue;const M=node.getWorldMatrix();
 for(const prim of mesh.listPrimitives()){
  const tex=prim.getMaterial()?.getBaseColorTexture();const uv=prim.getAttribute('TEXCOORD_0');if(!tex||!uv)continue;
  const {data,info}=await sharp(tex.getImage()).ensureAlpha().raw().toBuffer({resolveWithObject:true});
  const P=prim.getAttribute('POSITION').getArray(), U=uv.getArray(), I=prim.getIndices()?.getArray(), n=I?I.length:P.length/3, out=[];
  for(let j=0;j<n;j+=3){let tri=[],u=0,v=0;for(let k=0;k<3;k++){let id=I?I[j+k]:j+k,x=P[3*id],y=P[3*id+1],z=P[3*id+2];tri.push([M[0]*x+M[4]*y+M[8]*z+M[12],M[1]*x+M[5]*y+M[9]*z+M[13],M[2]*x+M[6]*y+M[10]*z+M[14]]);u+=U[2*id]/3;v+=U[2*id+1]/3;}
   const x=Math.min(info.width-1,Math.max(0,Math.floor((((u%1)+1)%1)*info.width))), y=Math.min(info.height-1,Math.max(0,Math.floor((((v%1)+1)%1)*info.height))),idx=(y*info.width+x)*4;out.push({tri,rgb:[data[idx],data[idx+1],data[idx+2]]});
  }
  const name=node.getName();writeFileSync('/private/tmp/nh-uv-'+name+'.json',JSON.stringify(out));console.log(name,n/3,info.width,info.height);
 }
}
