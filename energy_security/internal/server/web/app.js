const $ = (id) => document.getElementById(id);
const domainsOrder = [
  ["electricity","Electricity"],["gas","Gas"],["oil","Oil reserves"],["water","Hydrology"],["weather","Weather stress"]
];
const measurementOrder = ["electricity","gas","oil","water","weather","other"];
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
const clampPct = v => Math.max(0,Math.min(100,Number(v)||0));
function escapeHTML(s){const d=document.createElement("div");d.textContent=String(s??"");return d.innerHTML}
function observation(s,key){
  const o=s?.observations?.[key];
  return o && o.value!==null && o.value!==undefined ? o : null;
}
function indicatorRow(label,value,detail="",meter=null,tone="var(--accent)"){
  const meterHTML=meter===null?"":`<div class="indicator-meter" aria-hidden="true"><i style="width:${clampPct(meter)}%;background:${tone}"></i></div>`;
  return `<div class="indicator-row"><div class="indicator-copy"><span>${escapeHTML(label)}</span>${detail?`<small>${escapeHTML(detail)}</small>`:""}</div><strong>${escapeHTML(value)}</strong>${meterHTML}</div>`;
}
function indicatorBlock(title,rows){
  return rows.length?`<div class="domain-indicators"><div class="indicator-title">${escapeHTML(title)}</div><div class="indicator-list">${rows.join("")}</div></div>`:"";
}
function generationDiversity(s){
  const mix=observation(s,"electricity_generation_mw")?.attributes?.mix_mw || {};
  const vals=Object.values(mix).map(Number).filter(v=>Number.isFinite(v)&&v>0);
  if(vals.length<2)return null;
  const total=vals.reduce((a,v)=>a+v,0);
  if(total<=0)return null;
  const hhi=vals.reduce((a,v)=>{const p=v/total;return a+p*p},0);
  return clampPct((1-hhi)/(1-1/vals.length)*100);
}
function electricityIndicators(s){
  const rows=[];
  const diversity=generationDiversity(s);
  if(diversity!==null)rows.push(indicatorRow("Generation diversity",`${fmtScore(diversity)}/100`,"normalized diversity of the current generation mix",diversity,scoreTone(diversity)));

  const renew=observation(s,"renewable_share_pct");
  if(renew)rows.push(indicatorRow("Renewables",`${fmtNum(renew.value,1)}%`,"share of current load",renew.value));

  const load=observation(s,"electricity_load_mw");
  const nuclear=observation(s,"nuclear_output_mw");
  if(nuclear){
    const detail=load&&Number(load.value)>0?`${fmtNum(Number(nuclear.value)/Number(load.value)*100,1)}% of current load`:"current output";
    rows.push(indicatorRow("Nuclear",`${fmtNum(nuclear.value,0)} MW`,detail));
  }

  for(const [key,label] of [
    ["solar_output_mw","Solar"],["wind_output_mw","Wind"],["gas_power_output_mw","Gas-fired"],
    ["coal_power_output_mw","Coal / lignite"],["biomass_output_mw","Biomass"],["hydro_output_mw","Hydroelectric"]
  ]){
    const o=observation(s,key);
    if(o)rows.push(indicatorRow(label,`${fmtNum(o.value,0)} MW`,"current generation output"));
  }

  const cross=observation(s,"electricity_cross_border_mw");
  if(cross){
    const v=Number(cross.value);
    rows.push(indicatorRow(v<0?"Net imports":"Net exports",`${fmtNum(Math.abs(v),0)} MW`,"cross-border electricity trading"));
  }
  return indicatorBlock("Supporting indicators",rows);
}
function gasIndicators(s){
  const rows=[];
  const fill=observation(s,"gas_storage_fill_pct");
  if(fill)rows.push(indicatorRow("Storage fill",`${fmtNum(fill.value,1)}%`,"working storage capacity filled",fill.value));

  const stored=observation(s,"gas_in_storage_twh");
  const capacity=observation(s,"gas_working_capacity_twh");
  if(stored&&capacity)rows.push(indicatorRow("Stored gas",`${fmtNum(stored.value,1)} / ${fmtNum(capacity.value,1)} TWh`,"stored / working capacity"));
  else if(stored)rows.push(indicatorRow("Stored gas",`${fmtNum(stored.value,1)} TWh`,"gas in storage"));
  else if(capacity)rows.push(indicatorRow("Working capacity",`${fmtNum(capacity.value,1)} TWh`,"reported storage capacity"));

  const cover=observation(s,"gas_storage_consumption_cover_pct");
  if(cover)rows.push(indicatorRow("Annual-consumption cover",`${fmtNum(cover.value,1)}%`,"storage relative to annual consumption",cover.value));

  const injection=observation(s,"gas_storage_injection_gwh_day");
  if(injection)rows.push(indicatorRow("Injection",`${fmtNum(injection.value,1)} GWh/day`,"daily storage flow"));
  const withdrawal=observation(s,"gas_storage_withdrawal_gwh_day");
  if(withdrawal)rows.push(indicatorRow("Withdrawal",`${fmtNum(withdrawal.value,1)} GWh/day`,"daily storage flow"));
  const trend=observation(s,"gas_storage_daily_trend_pct");
  if(trend){
    const v=Number(trend.value);
    rows.push(indicatorRow("Daily storage trend",`${v>0?"+":""}${fmtNum(v,1)}%`,"latest reported daily change"));
  }

  const stock=observation(s,"gas_national_stock_twh");
  if(stock)rows.push(indicatorRow("National closing stock",`${fmtNum(stock.value,1)} TWh`,"Eurostat fallback measurement"));
  const proxy=observation(s,"gas_stock_index_pct");
  if(proxy)rows.push(indicatorRow("Stock index",`${fmtNum(proxy.value,1)}%`,"of trailing 36-month maximum; not capacity fill",proxy.value));
  return indicatorBlock("Storage indicators",rows);
}
function oilIndicators(s){
  const rows=[];
  const actual=observation(s,"oil_emergency_stock_days");
  if(actual)rows.push(indicatorRow("Emergency stocks",`${fmtNum(actual.value,1)} days`,"reported days equivalent"));
  const required=observation(s,"oil_required_stock_days");
  if(required)rows.push(indicatorRow("Minimum obligation",`${fmtNum(required.value,1)} days`,"reported statutory minimum series"));
  return indicatorBlock("Reserve indicators",rows);
}
function waterIndicators(s){
  const rows=[];
  for(const [key,label,unit,detail] of [
    ["danube_budapest_level_cm","Danube · Budapest level","cm","river level"],
    ["danube_budapest_discharge_m3s","Danube · Budapest discharge","m³/s","river discharge"],
    ["danube_paks_level_cm","Danube · Paks level","cm","river level near Paks"],
    ["danube_paks_discharge_m3s","Danube · Paks discharge","m³/s","river discharge near Paks"],
    ["balaton_level_cm","Lake Balaton level","cm","average lake level"]
  ]){
    const o=observation(s,key);
    if(o)rows.push(indicatorRow(label,`${fmtNum(o.value,1)} ${unit}`,detail));
  }
  return indicatorBlock("Hydrological indicators",rows);
}
function weatherIndicators(s){
  const rows=[];
  for(const [key,label,unit,detail] of [
    ["weather_temperature_c","Current temperature","°C","current local reading"],
    ["weather_wind_kmh","Current wind","km/h","current local wind speed"],
    ["forecast_max_temperature_c","7-day maximum","°C","forecast maximum temperature"],
    ["forecast_min_temperature_c","7-day minimum","°C","forecast minimum temperature"],
    ["forecast_max_wind_kmh","7-day maximum wind","km/h","forecast maximum wind speed"],
    ["forecast_precipitation_7d_mm","7-day precipitation","mm","forecast total precipitation"]
  ]){
    const o=observation(s,key);
    if(o)rows.push(indicatorRow(label,`${fmtNum(o.value,1)} ${unit}`,detail));
  }
  return indicatorBlock("Weather indicators",rows);
}
function domainIndicators(key,s){
  if(key==="electricity")return electricityIndicators(s);
  if(key==="gas")return gasIndicators(s);
  if(key==="oil")return oilIndicators(s);
  if(key==="water")return waterIndicators(s);
  if(key==="weather")return weatherIndicators(s);
  return "";
}
function domainCard(key,label,d,s){
  const score=d?.score; const pct=score==null?0:clampPct(score);
  const confidence=d?.confidence ? ` · ${Math.round(d.confidence)}% confidence` : "";
  return `<article class="domain card"><div class="domain-top"><h3>${escapeHTML(label)}</h3><span class="domain-score" style="color:${scoreTone(score)}">${fmtScore(score)}</span></div><div class="bar"><i style="width:${pct}%;background:${scoreTone(score)}"></i></div><p>${escapeHTML(d?.summary || "No current measurement available.")}</p><div class="domain-status">${escapeHTML(d?.status || "Unknown")}${confidence}</div>${domainIndicators(key,s)}</article>`;
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
  const x=t=>pad+(new Date(t).getTime()-minT)/Math.max(1,maxT-minT)*(W-2*pad); const y=v=>H
ad-(Number(v)/100)*(H-2*pad);
  const line=pts.map((p,i)=>`${i?'L':'M'} ${x(p.time). toFixed(1)} ${y(p.headline).toFixed(1)}`).join(' ');
  const fill=`${line} L ${x(pts.at(-1).time). toFixed(1)} ${H-pad} L ${x(pts[0].time).toFixed(1)} ${H-pad} Z`;
  const grids=[25,50,75,100].map(v=>`<line class="grid" x1="${pad}" y1="${y(v)}" x2="${W-pad}" y2="${y(v)}"/><text x="0" y="${y(v)+3}">${v}</text>`).join('');
  $("history").innerHTML=`<svg viewBox="0 0 ${W} ${H}" preserveAspectRatio="none"><defs><linearGradient id="area" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="var(--accent)" stop-opacity=".22"/><stop offset="1" stop-color="var(--accent)" stop-opacity="0"/></linearGradient></defs>${grids}<path class="fill" d="${fill}"/><path class="line" d="${line}"/></svg>`;
}

function measurementGroup(o){
  const explicit=String(o?.attributes?.provider_group||"").toLowerCase();
  if(explicit) return explicit;
  const domain=String(o?.domain||"").toLowerCase();
  if(domain==="nuclear"||domain==="renewables")return "electricity";
  if(measurementOrder.includes(domain)) return domain;
  const key=String(o?.key||"").toLowerCase();
  if(["nuclear_output_mw","solar_output_mw","wind_output_mw","hydro_output_mw","gas_power_output_mw","coal_power_output_mw","oil_power_output_mw","biomass_output_mw","renewable_share_pct"].includes(key))return "electricity";
  for(const group of measurementOrder.slice(0,-1)){if(key.startsWith(`${group}_`)||key.includes(`_${group}_`))return group}
  return "other";
}
function groupLabel(group){
  return {electricity:"Electricity",gas:"Gas",oil:"Oil reserves",water:"Hydrology",weather:"Weather",other:"Other"}[group]||group;
}
function observationRow(o){
  const source=escapeHTML(o.source||"Unknown source");
  const value=o.value!=null?fmtNum(o.value):escapeHTML(o.text||—");
  const quality=Math.round((o.quality||0)*100);
  return `<div class="observation"><span>${escapeHTML(o.label||o.key)}</span><span class="obs-value">${value} ${escapeHTML(o.unit||"")}</span><small>${source} · ${o.stale?"stale · ":""}${age(o.observed_at)}</small><small>${quality}% quality</small></div>`;
}
function renderDiagnostics(s){
  const openGroups=new Set([...document.querySelectorAll("#observations .measurement-group[open]")].map(el=>el.dataset.group).filter(Boolean));
  const providers=Object.values(s.providers||{}).sort((a,b)=>(a.name||a.id).localeCompare(b.name||b.id));
  $("providers").innerHTML=providers.length?providers.map(p=>`<div class="provider"><span><i class="dot ${escapeHTML(p.state)}"></i>${escapeHTML(p.name||p.id)}</span><strong>${escapeHTML(p.state||"unknown")}</strong><small>Last success ${age(p.last_succesr)}</small><small>${p.latency_ms?`${p.latency_ms} ms`:""}</small>${p.last_error?`<small title="${escapeHTML(p.last_error)}">${escapeHTML(p.last_error.slice(0,120))}</small><small></small>`:""}</div>`).join(''):'<p class="muted">Provider checks have not run yet.</p>';

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
  $("observations").innerHTML=ordered.length?ordered.map(group=>{
    const rows=groups.get(group);
    return `<details class="measurement-group" data-group="${escapeHTML(group)}"${openGroups.has(group)?' open':''}><summary><span>${escapeHTML(groupLabel(group))}</span><span class="group-count">${rows.length}</span></summary><div class="measurement-items">${rows.map(observationRow).join('')}</div></details>`;
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
  const available=domainsOrder.filter(([k])=>s.domains?.[k]?.score!=null).length;
  $("heroText").textContent=available?`${available} scored security domains are currently contributing. Supporting indicators provide context without being misrepresented as separate security scores.`:"No defensible composite score can be calculated from the currently available measurements.";
  $("domains").innerHTML=domainsOrder.map(([k,l])=>domainCard(k,l,s.domains?.[k],s)).join('');
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
