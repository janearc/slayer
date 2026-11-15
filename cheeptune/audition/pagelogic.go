package audition

// pageLogic is the part of the page's script that decides things, written
// as plain functions that touch nothing on the page: which tune is next
// unheard, what a key does given where focus is, what the counts say, what
// a failure says. They are in the page's script and a test runs them with
// node, when node is there. Nothing here reads or writes the page.
const pageLogic = `
var PLAYER_KEYS={' ':1,ArrowUp:1,ArrowDown:1,ArrowLeft:1,ArrowRight:1};
function nextUnvoted(names,votes,currentName){
  var start=names.indexOf(currentName),n=names.length;
  for(var i=1;i<=n;i++){var name=names[(start+i)%n];if(!votes[name])return name;}
  return '';
}
function tally(names,rec){
  var kept=0,tossed=0,left=0;
  Object.keys(rec.votes).forEach(function(n){
    if(rec.votes[n].vote==='keep')kept++;
    if(rec.votes[n].vote==='toss')tossed++;});
  names.forEach(function(n){if(!rec.votes[n])left++;});
  return {kept:kept,tossed:tossed,left:left,queued:rec.queue.length};
}
function countsText(t){
  return t.kept+' kept, '+t.tossed+' tossed, '+t.left+' still to hear.';
}
function tossedNames(rec){
  return Object.keys(rec.votes).filter(function(n){
    return rec.votes[n].vote==='toss';}).sort();
}
function keyAction(key,tag,held,hasRow){
  if(held)return '';
  if(tag==='VIDEO'&&PLAYER_KEYS[key])return '';
  if(key==='j'||key==='ArrowDown')return 'next';
  if(!hasRow)return '';
  if(key==='k')return 'keep';
  if(key==='t')return 'toss';
  if(key===' '&&tag!=='BUTTON')return 'toggle';
  return '';
}
function failText(what,status,text){
  if(!status)return what+'. Is the server still running?';
  return what+': '+String(text).trim();
}
`
