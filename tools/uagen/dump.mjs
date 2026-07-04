import { defaultRegexes, strMapper, strTest, lowerize, trim } from './node_modules/ua-parser-js/src/main/ua-parser-export.mjs';
const fnName = f => f===strMapper?'strMapper': f===strTest?'strTest': f===lowerize?'lowerize': f===trim?'trim': ('UNKNOWN:'+f.toString().slice(0,60));
const isRe = x => x instanceof RegExp;
function serVal(v){
  if (v===undefined||v===null) return {k:'undef'};
  if (isRe(v)) return {k:'re', re:v.source, flags:v.flags};
  if (Array.isArray(v)) return {k:'arr', arr:v};
  if (typeof v==='object') return {k:'map', entries: serMap(v)};
  return {k:'val', v:v};
}
function serMap(m){ return Object.keys(m).map(k=>[k, serVal(m[k])]); }
function serArg(a){ return serVal(a); }
function serProp(q){
  if (typeof q === 'string') return {t:'cap', prop:q};
  if (q.length===2){
    if (typeof q[1]==='function') return {t:'fn', prop:q[0], fn:fnName(q[1]), args:[]};
    return {t:'static', prop:q[0], val:q[1]};
  }
  if (typeof q[1]==='function' && !(q[1].exec && q[1].test)){
    return {t:'fn', prop:q[0], fn:fnName(q[1]), args:q.slice(2).map(serArg)};
  }
  const spec={t:'replace', prop:q[0], re:q[1].source, flags:q[1].flags, repl:q[2], args:[]};
  if (q.length>=4) spec.fn=fnName(q[3]);
  if (q.length>4) spec.args=q.slice(4).map(serArg);
  return spec;
}
const out={};
for (const cat of ['browser','cpu','device','engine','os']){
  const t=defaultRegexes[cat]; const rules=[];
  for (let i=0;i<t.length;i+=2)
    rules.push({regexes: t[i].filter(Boolean).map(r=>({re:r.source, flags:r.flags})), props:(t[i+1]||[]).map(serProp)});
  out[cat]=rules;
}
process.stdout.write(JSON.stringify(out));
