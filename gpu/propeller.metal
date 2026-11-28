// Generic compact rotor-tip optical density. Controls are authored, not a
// pressure-derived cavitation inception or individual bubble simulation.
static float propeller_at(device const float *card, float3 point) {
 device const uint *whole=reinterpret_cast<device const uint *>(card);
 if (whole[propeller_enabled]==0 || card[propeller_density]==0) return 0;
 float3 origin=float3(card[propeller_origin_x],card[propeller_origin_y],card[propeller_origin_z]);
 float3 axis=float3(card[propeller_axis_x],card[propeller_axis_y],card[propeller_axis_z]);
 float3 radial=float3(card[propeller_radial_x],card[propeller_radial_y],card[propeller_radial_z]);
 float3 crossway=float3(card[propeller_cross_x],card[propeller_cross_y],card[propeller_cross_z]);
 float3 delta=point-origin;
 float s=dot(delta,axis), active=card[propeller_active], core=card[propeller_core];
 if(s<0 || s>active)return 0;
 float x=dot(delta,radial),y=dot(delta,crossway),sum=0;
 float width=core;
 if(card[propeller_spread]>0)width+=card[propeller_spread]*max(0.0f,s-card[propeller_spread_start]);
 uint blades=whole[propeller_blades];
 for(uint k=0;k<blades;k++){
  float theta=card[propeller_omega]*(card[propeller_seconds]-s/card[propeller_slip])+6.283185307179586f*float(k)/float(blades);
  float2 centre=card[propeller_radius]*float2(cos(theta),sin(theta));
  if(card[propeller_wander]>0){
   ulong seed=(ulong(whole[propeller_seed_high])<<32)|ulong(whole[propeller_seed_low]);
   float q=(s-card[propeller_slip]*card[propeller_seconds])/card[propeller_wander_scale];
   float amount=card[propeller_wander]*wake_smooth(0.0f,card[propeller_wander_scale],s);
   centre+=amount*clamp(float2(fractal(seed,float(k)+.37f,q,.73f,2),fractal(seed,float(k)+.37f,q,17.19f,2)),-1.0f,1.0f);
  }
  float2 d=float2(x,y)-centre;
  float r2=dot(d,d);
  // 2026-11-26: lowered by its value at the 3 sigma cut (exp(-4.5)) so it
  // reaches zero there, as scene/medium's filament does
  if(r2<=9*width*width)sum+=max(exp(-.5f*r2/(width*width))-0.011108997f,0.0f);
 }
 if(card[propeller_spread]>0)sum*=(core/width)*(core/width);
 float start=wake_smooth(0.0f,2*core,s),end=1-wake_smooth(active-2*core,active,s);
 float pressure=max(0.0f,1-card[propeller_recovery]*s/card[propeller_deficit]);
 float breakup=1;
 if(sum>0 && card[propeller_fragment_contrast]>0){
  ulong seed=(ulong(whole[propeller_seed_high])<<32)|ulong(whole[propeller_seed_low]);
  float scale=card[propeller_fragment_scale];
  float packet=s-card[propeller_slip]*card[propeller_seconds];
  float pattern=fractal(seed,x/scale,y/scale,packet/scale,3);
  float mask=wake_smooth(0.0f,.5f,pattern);
  float amount=card[propeller_fragment_contrast]*wake_smooth(2*core,active/2,s);
  breakup-=amount*(1-mask);
 }
 return card[propeller_density]*sum*start*end*pressure*breakup;
}
