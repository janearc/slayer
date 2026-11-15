// Field placement is Slayer's y-up space rotated into Celestial's z-up chart.
static float3 field_out(constant float4 *f, float3 p) {
 float3 local=float3(p.x,p.z,-p.y);
 return f[1].xyz+f[0].w*(f[2].xyz*local.x+f[3].xyz*local.y+f[4].xyz*local.z);
}
static float3 field_way(constant float4 *f, float3 p) {
 float3 local=float3(p.x,p.z,-p.y);
 return normalize(f[2].xyz*local.x+f[3].xyz*local.y+f[4].xyz*local.z);
}
static float3 field_direction(constant float4 *f,float3 d) {
 return normalize(float3(dot(d,f[2].xyz),-dot(d,f[4].xyz),dot(d,f[3].xyz)));
}
static float field_entry(constant float4 *f,float3 from,float3 way) {
 float3 at=field_in(f,from),dir=field_direction(f,way);
 float outside=dot(at,at)-f[0].z*f[0].z;
 if (outside<0) return 0;
 float closing=dot(at,dir),disc=closing*closing-outside;
 if (disc<0) return -1;
 float along=-closing-sqrt(disc);
 return along>1e-9f ? along*f[0].w : -1;
}

static Seen field_seen(World world,constant float4 *f,device const float *seed,
 float3 from,float3 way) {
 float entry=seed[14];
 Hit surface=first(world,from,way,0);
 if (entry<0 || (surface.material>=0 && surface.distance<=entry)) return shade(world,from,way);
 Through front=through(world,from,way,entry);
 float3 light=front.added,keep=front.keep;
 float3 marchKeep=keep;
 float depth=entry;
 float4 x=float4(seed[6],seed[7],seed[8],seed[9]);
 float4 p=float4(seed[10],seed[11],seed[12],seed[13]);
 float energy=seed[15];
 int legStep=0;
 float4 xError=float4(0),pError=float4(0);
 from=field_out(f,x.yzw);
 for (int i=0;i<int(f[5].z) && shows(marchKeep,world.header->tuning.y);i++) {
  float radius=kerr_radius(x.yzw,f[0].y);
  KerrRate rate=kerr_rate_at(x,p,f[0].x,f[0].y,radius);
  bool leaving=(legStep>0 && radius>f[0].z) || (dot(x.yzw,rate.x.yzw)>0 && radius>f[6].x*f[0].x);
  if (-rate.x.x>f[5].y*abs(p.x) || radius<f[5].w*f[0].x) break;
  float4 nextX=x,nextP=p;
  float3 next;
  if (leaving) {
   way=field_way(f,rate.x.yzw);
   next=from+way*max(0.0f,(f[0].z*f[6].y-radius)*f[0].w);
  } else {
   kerr_step_held_from(nextX,nextP,xError,pError,rate,f[0].x,f[0].y,f[5].x*radius/abs(rate.x.x));
   next=field_out(f,nextX.yzw);
  }
  float length=distance(next,from);
  if (!isfinite(length)) break;
  if (!(length>0) && !leaving) break;
  if (length>0) way=(next-from)/length;
  Hit hit=first_limited(world,from,way,0,length);
  bool met=hit.material>=0 && hit.distance<=length;
  float far=met ? hit.distance : length;
  world.gravity=f;world.momentum=p;world.energy=energy;
  world.cuts=leaving ? 0 : int(f[6].z);
  Through gathered=through(world,from,way,far);
  light+=keep*gathered.added; keep*=gathered.keep;marchKeep*=gathered.keep;
  if (met) {
   Seen seen=surfaced(world,hit,from+way*hit.distance,facing(hit,way));
   seen.light=light+keep*seen.light; seen.depth+=depth;
   seen.met=dimmed(seen.met,keep);
   return seen;
  }
  depth+=length; from=next; x=nextX;p=nextP;
  legStep++;
  if (leaving) {
   world.gravity=nullptr;world.cuts=0;
   float entry=field_entry(f,from,way);
   Hit surface=first(world,from,way,0);
   if (entry<0 || (surface.material>=0 && surface.distance<=entry)) {
    Seen seen=shade(world,from,way);
    seen.light=light+keep*seen.light;
    seen.weight*=keep; seen.depth+=depth;
    seen.cover=1-(1-seen.cover)*max(keep.x,max(keep.y,keep.z));
    seen.met=dimmed(seen.met,keep); seen.mirrored=dimmed(seen.mirrored,keep);
    return seen;
   }
   Through straight=through(world,from,way,entry);
   light+=keep*straight.added; keep*=straight.keep;depth+=entry;
   from+=way*entry;
   x=float4(0,field_in(f,from));
   p=kerr_arriving(x.yzw,field_direction(f,way),f[0].x,f[0].y);
   energy=p.x;
   marchKeep=straight.keep;
   legStep=0;
   xError=float4(0);pError=float4(0);
  }
 }
 return {light,1,depth,float3(0),float3(0),unpainted,unpainted,unpainted};
}

kernel void trace(constant Header *header [[buffer(0)]],
 device const Sphere *spheres [[buffer(1)]],
 device const Triangle *triangles [[buffer(2)]],
 device const Node *nodes [[buffer(3)]],
 device const Material *materials [[buffer(4)]],
 device const Lamp *lamps [[buffer(5)]],
 device const Volume *volumes [[buffer(6)]],
 device const float *baked [[buffer(7)]],
 device const Tree *trees [[buffer(8)]],
 device const Instance *instances [[buffer(9)]],
 device const Node *tops [[buffer(10)]],
 device const float4 *sky [[buffer(11)]],
 device const Bend *bends [[buffer(12)]],
 device const float *rays [[buffer(13)]],
 device const float *boxes [[buffer(14)]],
 device const Field *fields [[buffer(15)]],
 device const float *heights [[buffer(16)]],
 constant float4 *field [[buffer(17)]],
 device float *out [[buffer(18)]],
 constant uint2 &origin [[buffer(19)]],
 uint2 at [[thread_position_in_grid]],uint2 grid [[threads_per_grid]]) {
 World world={header,spheres,triangles,nodes,materials,lamps,volumes,
 baked,trees,instances,tops,sky,bends,rays,boxes,fields,heights,nullptr};
 uint2 pixel=uint2(at.x,at.y+origin.y);
 device const float *ray=rays+(pixel.y*grid.x+pixel.x)*field_seed_words;
 Seen seen=field_seen(world,field,ray,float3(ray[0],ray[1],ray[2]),float3(ray[3],ray[4],ray[5]));
 device float *sample=out+(at.y*grid.x+at.x)*per_sample;
 sample[0]=seen.light.x;sample[1]=seen.light.y;sample[2]=seen.light.z;sample[3]=seen.cover;
 sample[4]=seen.weight.x;sample[5]=seen.weight.y;sample[6]=seen.weight.z;sample[7]=seen.depth;
 sample[8]=seen.sky.x;sample[9]=seen.sky.y;sample[10]=seen.sky.z;sample[11]=0;
 if (records>0) record(sample,0,seen.met);
 if (records>1) record(sample,1,seen.mirrored);
 if (records>2) record(sample,2,seen.passed);
}
