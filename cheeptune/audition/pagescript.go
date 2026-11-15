package audition

// pageScript is the page's behaviour: the votes, the queue, the keys.
// It asks /votes for the record, and when nothing answers keeps votes in
// local storage and says so. When the server answers with an error it says
// what the server said and does not keep votes in the browser. The
// deciding is in pageLogic.
const pageScript = "\n(function(){" + pageLogic + pageBody + "})();\n"

// pageBody is the script's part that reads and changes the page.
const pageBody = `
var KEY='tune-library-votes', server=false, notice='', state={votes:{},queue:[]};
var rows=[].slice.call(document.querySelectorAll('.tune')), current=null;
var names=rows.map(function(r){return r.dataset.name;});
function byName(n){return document.getElementById('tune-'+n);}
function say(what,r){return r.text().then(function(t){
  notice=failText(what,r.status,t);draw();});}
function load(){
  if(location.protocol==='file:'){return Promise.resolve(local());}
  return fetch('votes').then(function(r){
    server=true;
    if(r.ok){return r.json().then(function(rec){state=rec;});}
    return say('The server could not read the votes',r);
  },function(){server=false;local();}).catch(function(){server=false;local();});
}
function local(){
  try{var s=JSON.parse(localStorage.getItem(KEY));if(s&&s.votes)state=s;}catch(e){}
}
function save(){try{localStorage.setItem(KEY,JSON.stringify(state));}catch(e){}}
function cast(name,vote){
  if(server){
    return fetch('votes',{method:'POST',headers:{'Content-Type':'application/json'},
      body:JSON.stringify({name:name,vote:vote})}).then(function(r){
      if(!r.ok){return say('The vote was not saved',r);}
      return r.json().then(function(rec){state=rec;notice='';draw();});
    },function(){notice=failText('The vote was not saved',0,'');draw();});
  }
  state.queue=state.queue.filter(function(q){return q!==name;});
  if(vote==='clear'){delete state.votes[name];}
  else{state.votes[name]={vote:vote,at:new Date().toISOString()};
    if(vote==='keep'){state.queue.push(name);}}
  save();draw();return Promise.resolve();
}
function toss(r){var v=r.querySelector('video');if(!v.paused)v.pause();
  current=r;cast(r.dataset.name,'toss');}
function voteOf(n){var v=state.votes[n];return v?v.vote:'';}
function draw(){
  var t=tally(names,state),ql=document.getElementById('queue-list'),
    tl=document.getElementById('tossed-list');
  ql.innerHTML='';tl.innerHTML='';
  rows.forEach(function(r){var v=voteOf(r.dataset.name);
    r.classList.toggle('kept',v==='keep');r.classList.toggle('tossed',v==='toss');});
  state.queue.forEach(function(n){ql.appendChild(item(n,'Remove'));});
  tossedNames(state).forEach(function(n){tl.appendChild(item(n,'Undo'));});
  document.getElementById('queue-empty').hidden=t.queued>0;
  document.getElementById('tossed-count').textContent=t.tossed;
  document.getElementById('counts').textContent=countsText(t);
  document.getElementById('where').textContent=notice||(server?
    'Votes are saved in the library folder.':'Votes are kept in this browser only.');
  document.getElementById('copy').hidden=server;
}
function item(name,label){
  var li=document.createElement('li'),b=document.createElement('button');
  li.appendChild(document.createTextNode(name));b.type='button';b.textContent=label;
  b.onclick=function(){cast(name,'clear');};li.appendChild(b);return li;
}
function playOnly(r){
  rows.forEach(function(o){var v=o.querySelector('video');if(o!==r&&!v.paused)v.pause();});
}
function toggle(r){var v=r.querySelector('video');if(v.paused){playOnly(r);v.play();}else v.pause();}
function next(){
  var name=nextUnvoted(names,state.votes,current?current.dataset.name:'');
  var r=name&&byName(name);
  if(!r)return;
  current=r;r.focus();r.scrollIntoView({block:'start'});
  playOnly(r);r.querySelector('video').play();
}
rows.forEach(function(r){
  var name=r.dataset.name,v=r.querySelector('video');
  r.querySelector('.keep').onclick=function(){current=r;cast(name,'keep');};
  r.querySelector('.toss').onclick=function(){toss(r);};
  r.addEventListener('focusin',function(){current=r;});
  v.addEventListener('play',function(){playOnly(r);});
  [].forEach.call(r.querySelectorAll('.seek'),function(b){
    b.onclick=function(){playOnly(r);v.currentTime=parseFloat(b.dataset.at);v.play();};});
});
document.addEventListener('keydown',function(e){
  var tag=e.target.tagName,row=e.target.closest&&e.target.closest('.tune');
  if(!row&&current&&tag==='BODY')row=current;
  var act=keyAction(e.key,tag,e.ctrlKey||e.metaKey||e.altKey,!!row);
  if(act==='next'||act==='toggle'){e.preventDefault();}
  if(act==='next'){next();}
  else if(act==='toggle'){toggle(row);}
  else if(act==='keep'){current=row;cast(row.dataset.name,'keep');}
  else if(act==='toss'){toss(row);}
});
document.getElementById('copy').onclick=function(){
  var lines=['queue: '+state.queue.join(', ')];
  Object.keys(state.votes).sort().forEach(function(n){lines.push(state.votes[n].vote+' '+n);});
  var text=lines.join('\n'),done=function(){document.getElementById('copied').textContent='Copied.';};
  if(navigator.clipboard){navigator.clipboard.writeText(text).then(done,function(){window.prompt('Copy this',text);});}
  else{window.prompt('Copy this',text);}
};
load().then(draw);
`
