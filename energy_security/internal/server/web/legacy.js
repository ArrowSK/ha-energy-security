const $ = (id) => document.getElementById(id);
const domainsOrder = [
  ["electricity","Electricity"],["gas","Gas"],["oil","Oil reserves"],["water","Water & hydro"],
  ["weather","Weather stress"],["nuclear","Nuclear"],["renewables","Renewables"]
];
const measurementOrder = ["electricity","gas","oil","water","weather","nuclear","renewables","other"];
let lastSnapshot = null;

const fmtScore = v => (v === null || v === undefined || Number.isNaN(Number(v))) ? "—" : Math.round(Number(v));
const fmtNum = (v, d=1) => (v === null || v === undefined) ? "—" : Number(v).toLocaleString(undefined,{maximumFractionDigits:d});
const age = t => {
  if(!t) return "never";
  const dt=new Date(t);
  if(Number.isNaN(dt.getTime()) || dt.getUTCFullYear()<2000) return "never";
  const ms=Date.now()-dt.getTime(); if(ms<0) return "just now";
  const m=Math.floor(ms/60000); if(m<1)return "just now"; if(m<60)return `${m} min ago`;
  const h=Math.floor(m/60); if(h<48)return `${h} h ago`; return `${Math.floor(h/24)} d ago`;
};
const scoreTone = s => s == null ? "var(--muted)" : s>=75 ? "var(--accent)" : s>=55 ? "var(--warn)" : "var(--bad)";
function escapeHTML(s){const d=document.createElement("div");d.textContent=String(s??"");return d.innerHTML}

function domainCard(key,label,d){
  const score=d?.score; const pct=score==null?0:Math.max(0,Math.min(100,score));
  const confidence=d?.confidence ? ` · ${Math.round(d.confidence)}% confidence` : "";
  return `<article class="domain card"><div class="domain-top"><h3>${label}</h3><span class="domain-score" style="color:${scoreTone(score)}">${fmtScore(score)}</span></div><div class="bar"><i style="width:${pct}%;background:${scoreTone(score)}"></i></div><p>${escapeHTML(d?.summary || "No current measurement available.")}</p><div class="domain-status">${escapeHTML(d?.status || "Unknown")}${confidence}</div></article>`;
}

function renderMix(s){
  const gen=s.observations?.electricity_generation_mw; const mix=gen?.attributes?.mix_mw || {};
  const rows=Object.entries(mix).filter(([,v])=>Number(v)>0).sort((a,b)=>b[1]-a[1]).slice(0,8);
  const total=rows.reduce((a,[,v])=>a+Number(v),0);
  $("mix").innerHTML=rows.length?rows.map(([n,v])=>`<div class="mix-row"><span title="${escapeHTML(n)}">${escapeHTML(n.length>17?n.slice(0,16)+"…":n)}</span><div class="mix-track"><i style="width:${total?Number(v)/total*100:0}%"></i></div><span class="mix-val">${fmtNum(v,0)} MW</span></div>`).join(""):`<p class="muted">Generation mix is not available from the current source.</p>`;
  const load=s.observations?.electricity_load_mw?.value; const generation=gen?.value;
  if(load!=null&&generation!=null){
    $("loadSummary").textContent=`${fmtNum(generation,0)} MW gen · ${fmtNum(load,0)} MW load`;
  }else if(s.domains?.electricity?.score!=null){
    $("loadSummary").textContent="Reference load used";
  }else{
    $("loadSummary").textContent="Data limited";
  }
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

function measurementGroup(o){
  const explicit=String(o?.attributes?.provider_group||"").toLowerCase();
  if(explicit) return explicit;
  const domain=String(o?.domain||"").toLowerCase();
  if(measurementOrder.includes(domain)) return domain;
  const key=String(o?.key||"").toLowerCase();
  for(const group of measurementOrder.slice(0,-1)){if(key.startsWith(`${group}_`)||key.includes(`_${group}_`))return group}
  return "other";
}
function groupLabel(group){
  return {electricity:"Electricity",gas:"Gas",oil:"Oil reserves",water:"Water & hydro",weather:"Weather",nuclear:"Nuclear",renewables:"Renewables",other:"Other"}[group]||group;
}
function observationRow(o){
  const source=escapeHTML(o.source||"Unknown source");
  const value=o.value!=null?fmtNum(o.value):escapeHTML(o.text||"—");
  const quality=Math.round((o.quality||0)*100);
  return `<div class="observation"><span>${escapeHTML(o.label||o.key)}</span><span class="obs-value">${value} ${escapeHTML(o.unit||"")}</span><small>${source} · ${o.stale?"stale · ":""}${age(o.observed_at)}</small><small>${quality}% quality</small></div>`;
}
function renderDiagnostics(s){
  const providers=Object.values(s.providers||{}).sort((a,b)=>(a.name||a.id).localeCompare(b.name||b.id));
  $("providers").innerHTML=providers.length?providers.map(p=>`<div class="provider"><span><i class="dot ${escapeHTML(p.state)}"></i>${escapeHTML(p.name||p.id)}</span><strong>${escapeHTML(p.state||"unknown")}</strong><small>Last success ${age(p.last_success)}</small><small>${p.latency_ms?`${p.latency_ms} ms`:""}</small>${p.last_error?`<small title="${escapeHTML(p.last_error)}">${escapeHTML(p.last_error.slice(0,120))}</small><small></small>`:""}</div>`).join(''):'<p class="muted">Provider checks have not run yet.</p>';

  const notes=s.notes||[];
  $("notesWrap").hidden=!notes.length;
  $("notes").innerHTML=notes.map(n=>`<li>${escapeHTML(n)}</li>`).join('');

  const observations=Object.values(s.observations||{}).sort((a,b)=>String(a.label||a.key).localeCompare(String(b.label||b.key)));
  $("measurementCount").textContent=`${observations.length} measurements`;
  const groups=new Map();
  for(const o of observations){
    const group=measurementGroup(o);
    if(!groups.has(group))groups.set(group,[]);
    groups.get(group).push(o);
  }
  const ordered=[...groups.keys()].sort((a,b)=>{
    const ai=measurementOrder.indexOf(a),bi=measurementOrder.indexOf(b);
    return (ai<0?999:ai)-(bi<0?999:bi)||a.localeCompare(b);
  });
  $("observations").innerHTML=ordered.length?ordered.map((group,index)=>{
    const rows=groups.get(group);
    return `<details class="measurement-group"${index===0?' open':''}><summary><span>${escapeHTML(groupLabel(group))}</span><span class="group-count">${rows.length}</span></summary><div class="measurement-items">${rows.map(observationRow).join('')}</div></details>`;
  }).join(''):'<p class="muted">No measurements are cached.</p>';
}

function render(s){
  lastSnapshot=s;
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
  try{
    const r=await fetch('api/v1/status',{cache:'no-store'});
    if(!r.ok)throw new Error(`${r.status}`);
    const s=await r.json(); render(s); return s;
  }catch(e){
    $("updated").textContent=`Dashboard API unavailable (${e.message})`;
    return null;
  }
}
function setRefreshBusy(busy){
  const b=$("refreshBtn");
  b.disabled=busy;
  b.classList.toggle("spinning",busy);
  b.setAttribute("aria-busy",busy?"true":"false");
}
async function manualRefresh(){
  if($("refreshBtn").disabled)return;
  setRefreshBusy(true);
  const before=lastSnapshot?.updated_at||"";
  try{
    const r=await fetch('api/v1/refresh',{method:'POST'});
    if(!r.ok)throw new Error(`${r.status}`);
    for(let tries=0;tries<20;tries++){
      await new Promise(resolve=>setTimeout(resolve,2500));
      const s=await load();
      if(s?.updated_at && s.updated_at!==before)break;
    }
  }catch(e){
    $("updated").textContent=`Refresh failed (${e.message})`;
  }finally{
    setRefreshBusy(false);
  }
}

function closeMenu(){
  $("appMenu").hidden=true;
  $("menuBtn").setAttribute("aria-expanded","false");
}
$("menuBtn").addEventListener("click",(event)=>{
  event.stopPropagation();
  const opening=$("appMenu").hidden;
  $("appMenu").hidden=!opening;
  $("menuBtn").setAttribute("aria-expanded",opening?"true":"false");
});
document.addEventListener("click",(event)=>{
  if(!$("appMenu").hidden && !event.target.closest(".menu-wrap"))closeMenu();
});

async function openSetup(){
  closeMenu();
  const status=$("setupStatus");
  status.textContent="Loading configuration…"; status.className="setup-status";
  $("setupDialog").showModal();
  try{
    const r=await fetch('api/v1/config',{cache:'no-store'});
    if(!r.ok)throw new Error(`${r.status}`);
    const cfg=await r.json();
    const options=[{code:"auto",name:"HOME country (automatic)"},...(cfg.countries||[])];
    $("countryOptions").innerHTML=options.map(c=>`<option value="${escapeHTML(c.code)}">${escapeHTML(c.name)}${c.code==="auto"?"":` (${escapeHTML(c.code)})`}</option>`).join("");
    $("setupCountry").value=cfg.country||"auto";
    $("setupRefresh").value=cfg.refresh_minutes??30;
    $("setupEntities").checked=Boolean(cfg.enable_ha_entities);
    $("setupWeather").checked=Boolean(cfg.enable_weather);
    $("setupAGSI").value=""; $("setupENTSOE").value="";
    $("clearAGSI").checked=false; $("clearENTSOE").checked=false;
    $("agsiState").textContent=cfg.agsi_key_configured?"configured":"not configured";
    $("entsoeState").textContent=cfg.entsoe_token_configured?"configured":"not configured";
    status.textContent="";
  }catch(e){
    status.textContent=`Unable to load setup (${e.message}).`;
    status.className="setup-status error";
  }
}
function closeSetup(){if($("setupDialog").open)$("setupDialog").close()}
$("setupMenuBtn").addEventListener("click",openSetup);
$("setupCloseBtn").addEventListener("click",closeSetup);
$("setupCancelBtn").addEventListener("click",closeSetup);

async function waitForRestart(){
  await new Promise(resolve=>setTimeout(resolve,2200));
  for(let i=0;i<30;i++){
    try{
      const r=await fetch('healthz',{cache:'no-store'});
      if(r.ok){window.location.reload();return}
    }catch(_){}
    await new Promise(resolve=>setTimeout(resolve,2000));
  }
  $("setupStatus").textContent="Settings were saved, but the dashboard has not reconnected yet. Reopen the app from Home Assistant.";
  $("setupStatus").className="setup-status error";
  $("setupSaveBtn").disabled=false;
}
$("setupForm").addEventListener("submit",async(event)=>{
  event.preventDefault();
  const save=$("setupSaveBtn"),status=$("setupStatus");
  save.disabled=true; status.className="setup-status"; status.textContent="Saving configuration…";
  const payload={
    country:$("setupCountry").value,
    refresh_minutes:Number($("setupRefresh").value),
    enable_ha_entities:$("setupEntities").checked,
    enable_weather:$("setupWeather").checked,
    agsi_key:$("setupAGSI").value,
    entsoe_token:$("setupENTSOE").value,
    clear_agsi_key:$("clearAGSI").checked,
    clear_entsoe_token:$("clearENTSOE").checked
  };
  try{
    const r=await fetch('api/v1/config',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(payload)});
    const result=await r.json().catch(()=>({}));
    if(!r.ok)throw new Error(result.error||`${r.status}`);
    status.textContent="Saved. Restarting Energy Security Monitor…";
    status.className="setup-status success";
    waitForRestart();
  }catch(e){
    status.textContent=`Save failed: ${e.message}`;
    status.className="setup-status error";
    save.disabled=false;
  }
});

function setActiveNav(target){
  document.querySelectorAll(".nav-item").forEach(b=>b.classList.toggle("active",b.dataset.target===target));
}
document.querySelectorAll(".nav-item").forEach(button=>button.addEventListener("click",()=>{
  const target=$(button.dataset.target);
  if(target){target.scrollIntoView({behavior:"smooth",block:"start"});setActiveNav(button.dataset.target)}
}));
const navSections=[...document.querySelectorAll(".nav-section")];
const navObserver=new IntersectionObserver(entries=>{
  const visible=entries.filter(e=>e.isIntersecting).sort((a,b)=>b.intersectionRatio-a.intersectionRatio)[0];
  if(visible)setActiveNav(visible.target.id);
},{rootMargin:"-15% 0px -60% 0px",threshold:[0,.1,.25,.5]});
navSections.forEach(section=>navObserver.observe(section));

$("refreshBtn").addEventListener('click',manualRefresh);
load();
setInterval(load,60000);
