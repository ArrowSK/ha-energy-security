const $ = (id) => document.getElementById(id);
const domainsOrder = [
  ["electricity","Electricity"],["gas","Gas"],["oil","Oil reserves"],["water","Water & hydro"],
  ["weather","Weather stress"],["nuclear","Nuclear"],["renewables","Renewables"]
];
const fmtScore = v => (v === null || v === undefined || Number.isNaN(Number(v))) ? "—" : Math.round(Number(v));
const fmtNum = (v, d=1) => (v === null || v === undefined) ? "—" : Number(v).toLocaleString(undefined,{maximumFractionDigits:d});
const age = t => {
  if(!t) return "unknown";
  const ms=Date.now()-new Date(t).getTime(); if(ms<0) return "just now";
  const m=Math.floor(ms/60000); if(m<1)return "just now"; if(m<60)return `${m} min ago`;
  const h=Math.floor(m/60); if(h<48)return `${h} h ago`; return `${Math.floor(h/24)} d ago`;
};
const scoreTone = s => s == null ? "var(--muted)" : s>=75 ? "var(--accent)" : s>=55 ? "var(--warn)" : "var(--bad)";

function domainCard(key,label,d){
  const score=d?.score; const pct=score==null?0:Math.max(0,Math.min(100,score));
  const confidence=d?.confidence ? ` · ${Math.round(d.confidence)}% confidence` : "";
  return `<article class="domain card"><div class="domain-top"><h3>${label}</h3><span class="domain-score" style="color:${scoreTone(score)}">${fmtScore(score)}</span></div><div class="bar"><i style="width:${pct}%;background:${scoreTone(score)}"></i></div><p>${escapeHTML(d?.summary || "No current measurement available.")}</p><div class="domain-status">${escapeHTML(d?.status || "Unknown")}${confidence}</div></article>`;
}
function escapeHTML(s){const d=document.createElement("div");d.textContent=String(s??"");return d.innerHTML}

function renderMix(s){
  const gen=s.observations?.electricity_generation_mw; const mix=gen?.attributes?.mix_mw || {};
  const rows=Object.entries(mix).filter(([,v])=>Number(v)>0).sort((a,b)=>b[1]-a[1]).slice(0,8);
  const total=rows.reduce((a,[,v])=>a+Number(v),0);
  $("mix").innerHTML=rows.length?rows.map(([n,v])=>`<div class="mix-row"><span title="${escapeHTML(n)}">${escapeHTML(n.length>17?n.slice(0,16)+"…":n)}</span><div class="mix-track"><i style="width:${total?Number(v)/total*100:0}%"></i></div><span class="mix-val">${fmtNum(v,0)} MW</span></div>`).join(""):`<p class="muted">Generation mix is not available from the current source.</p>`;
  const load=s.observations?.electricity_load_mw?.value; const generation=gen?.value;
  $("loadSummary").textContent=(load!=null&&generation!=null)?`${fmtNum(generation,0)} MW gen · ${fmtNum(load,0)} MW load`:"Data limited";
}

function renderHistory(history){
  const pts=(history||[]).filter(p=>p.headline!=null);
  if(pts.length<2){$("history").innerHTML='<p class="muted">Trend appears after at least two assessments.</p>';return}
  const W=900,H=190,pad=24; const minT=new Date(pts[0].time).getTime(),maxT=new Date(pts.at(-1).time).getTime();
  const x=t=>pad+(new Date(t).getTime()-minT)/Math.max(1,maxT-minT)*(W-2*pad); const y=v=>H-pad-(Number(v)/100)*(H-2*pad);
  const line=pts.map((p,i)=>`${i?'L':'M'} ${x(p.time).toFixed(1)} ${y(p.headline).toFixed(1)}`).join(' ');
  const fill=`${line} L ${x(pts.at(-1).time).toFixed(1)} ${H-pad} L ${x(pts[0].time).toFixed(1)} ${H-pad} Z`;
  const grids=[25,50,75,100].map(v=>`<line class="grid" x1="${pad}" y1="${y(v)}" x2="${W-pad}" y2="${y(v)}"/><text x="0" y="${y(v)+3}">${v}</text>`).join('');
  $("history").innerHTML=`<svg viewBox="0 0 ${W} ${H}" preserveAspectRatio="none"><defs><linearGradient id="area" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="var(--accent)" stop-opacity=".22"/><stop offset="1" stop-color="var(--accent)" stop-opacity="0"/></linearGradient></defs>${grids}<path class="fill" d="${fill}"/><path class="line" d="${line}"/></svg>`;
}

function renderDiagnostics(s){
  const providers=Object.values(s.providers||{}).sort((a,b)=>(a.name||a.id).localeCompare(b.name||b.id));
  $("providers").innerHTML=providers.length?providers.map(p=>`<div class="provider"><span><i class="dot ${escapeHTML(p.state)}"></i>${escapeHTML(p.name||p.id)}</span><strong>${escapeHTML(p.state||"unknown")}</strong><small>Last success ${age(p.last_success)}</small><small>${p.latency_ms?`${p.latency_ms} ms`:""}</small>${p.last_error?`<small title="${escapeHTML(p.last_error)}">${escapeHTML(p.last_error.slice(0,90))}</small><small></small>`:""}</div>`).join(''):'<p class="muted">Provider checks have not run yet.</p>';
  const observations=Object.values(s.observations||{}).sort((a,b)=>(a.domain+a.label).localeCompare(b.domain+b.label));
  $("observations").innerHTML=observations.length?observations.map(o=>`<div class="observation"><span>${escapeHTML(o.label)}</span><span class="obs-value">${o.value!=null?fmtNum(o.value):escapeHTML(o.text||"—")} ${escapeHTML(o.unit||"")}</span><small>${escapeHTML(o.source||"Unknown source")} · ${o.stale?"stale · ":""}${age(o.observed_at)}</small><small>${Math.round((o.quality||0)*100)}%</small></div>`).join(''):'<p class="muted">No measurements are cached.</p>';
  const notes=s.notes||[]; $("notesWrap").hidden=!notes.length; $("notes").innerHTML=notes.map(n=>`<li>${escapeHTML(n)}</li>`).join('');
}

function render(s){
  $("countryName").textContent=s.country_name || s.country || "Energy Security";
  $("updated").textContent=s.updated_at?`Assessment updated ${age(s.updated_at)}${s.location_name?` · ${s.location_name}`:""}`:"Waiting for the first assessment…";
  const h=s.scores?.headline; $("headline").textContent=fmtScore(h); $("ringScore").textContent=fmtScore(h); $("scoreRing").style.setProperty('--score',h??0);
  $("headline").style.color=scoreTone(h); $("status").textContent=s.scores?.status||"Data limited"; $("status").style.color=scoreTone(h);
  $("confidence").textContent=`Confidence ${fmtScore(s.scores?.confidence)}%`;
  $("currentScore").textContent=fmtScore(s.scores?.current); $("outlookScore").textContent=fmtScore(s.scores?.outlook_7d); $("strategicScore").textContent=fmtScore(s.scores?.strategic);
  const available=Object.values(s.domains||{}).filter(d=>d.score!=null).length;
  $("heroText").textContent=available?`${available} scored domains are currently contributing. Missing or stale measurements lower confidence; they are not converted to zero.`:"No defensible composite score can be calculated from the currently available measurements.";
  $("domains").innerHTML=domainsOrder.map(([k,l])=>domainCard(k,l,s.domains?.[k])).join('');
  const alerts=s.alerts||[]; $("alerts").innerHTML=alerts.length?alerts.map(a=>`<div class="alert">${escapeHTML(a)}</div>`).join(''):'<div class="alert all-clear">No active stress signal is present in the available scored domains.</div>';
  renderMix(s);renderHistory(s.history);renderDiagnostics(s);
}

async function load(){
  try{const r=await fetch('api/v1/status',{cache:'no-store'});if(!r.ok)throw new Error(`${r.status}`);render(await r.json())}
  catch(e){$("updated").textContent=`Dashboard API unavailable (${e.message})`}
}
async function manualRefresh(){
  const b=$("refreshBtn");b.disabled=true;b.textContent='Refreshing…';
  try{await fetch('api/v1/refresh',{method:'POST'});let tries=0;const timer=setInterval(async()=>{tries++;await load();if(tries>=12){clearInterval(timer);b.disabled=false;b.textContent='Refresh'}},5000)}catch(e){b.disabled=false;b.textContent='Refresh'}
}
$("refreshBtn").addEventListener('click',manualRefresh);load();setInterval(load,60000);
