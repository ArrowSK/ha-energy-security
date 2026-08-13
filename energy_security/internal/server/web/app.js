(() => {
  const legacy = document.createElement('script');
  legacy.src = 'legacy.js';
  legacy.async = false;

  legacy.addEventListener('load', () => {
    const baseRender = window.render;
    if (typeof baseRender !== 'function' || typeof window.domainCard !== 'function') {
      console.error('Energy Security dashboard legacy renderer did not load');
      return;
    }

    const style = document.createElement('style');
    style.textContent = `
      .domain-grid{grid-template-columns:repeat(2,minmax(0,1fr))}
      .domain-indicators{margin-top:14px;padding-top:12px;border-top:1px solid rgba(255,255,255,.065)}
      .indicator-title{margin-bottom:9px;color:var(--muted);font-size:.70rem;font-weight:780;letter-spacing:.11em;text-transform:uppercase}
      .indicator-list{display:grid;gap:9px}
      .indicator-row{display:grid;grid-template-columns:minmax(0,1fr) auto;gap:4px 12px;align-items:center;padding:2px 0}
      .indicator-copy{min-width:0;display:grid;gap:2px}
      .indicator-copy>span{font-size:.84rem;font-weight:680}
      .indicator-copy>small{color:var(--muted);font-size:.72rem;line-height:1.35}
      .indicator-row>strong{font-size:.86rem;text-align:right;white-space:nowrap}
      .indicator-meter{grid-column:1/-1;height:4px;border-radius:20px;background:rgba(255,255,255,.06);overflow:hidden}
      .indicator-meter>i{display:block;height:100%;border-radius:inherit}
      @media(max-width:620px){
        .domain-grid{grid-template-columns:1fr}
        .indicator-copy>span,.indicator-row>strong{font-size:.9rem}
        .indicator-copy>small{font-size:.8rem}
        .indicator-title{font-size:.74rem}
      }
    `;
    document.head.appendChild(style);

    const topDomains = [
      ['electricity', 'Electricity'],
      ['gas', 'Gas'],
      ['oil', 'Oil reserves'],
      ['water', 'Hydrology'],
      ['weather', 'Weather stress']
    ];

    const fmt = (v, d = 1) => v == null ? '—' : Number(v).toLocaleString(undefined, {maximumFractionDigits: d});
    const clamp = v => Math.max(0, Math.min(100, Number(v) || 0));
    const obs = (s, key) => {
      const o = s?.observations?.[key];
      return o && o.value != null ? o : null;
    };
    const tone = score => score == null ? 'var(--muted)' : score >= 75 ? 'var(--accent)' : score >= 55 ? 'var(--warn)' : 'var(--bad)';
    const esc = value => {
      const d = document.createElement('div');
      d.textContent = String(value ?? '');
      return d.innerHTML;
    };
    const row = (label, value, detail = '', meter = null, meterTone = 'var(--accent)') => {
      const bar = meter == null ? '' : `<div class="indicator-meter" aria-hidden="true"><i style="width:${clamp(meter)}%;background:${meterTone}"></i></div>`;
      return `<div class="indicator-row"><div class="indicator-copy"><span>${esc(label)}</span>${detail ? `<small>${esc(detail)}</small>` : ''}</div><strong>${esc(value)}</strong>${bar}</div>`;
    };
    const block = (title, rows) => rows.length ? `<div class="domain-indicators"><div class="indicator-title">${esc(title)}</div><div class="indicator-list">${rows.join('')}</div></div>` : '';

    const diversity = s => {
      const mix = obs(s, 'electricity_generation_mw')?.attributes?.mix_mw || {};
      const vals = Object.values(mix).map(Number).filter(v => Number.isFinite(v) && v > 0);
      if (vals.length < 2) return null;
      const total = vals.reduce((a, v) => a + v, 0);
      if (total <= 0) return null;
      const hhi = vals.reduce((a, v) => { const p = v / total; return a + p * p; }, 0);
      return clamp((1 - hhi) / (1 - 1 / vals.length) * 100);
    };

    const electricity = s => {
      const rows = [];
      const div = diversity(s);
      if (div != null) rows.push(row('Generation diversity', `${Math.round(div)}/100`, 'normalized diversity of the current generation mix', div, tone(div)));
      const renew = obs(s, 'renewable_share_pct');
      if (renew) rows.push(row('Renewables', `${fmt(renew.value, 1)}%`, 'share of current load', renew.value));
      const load = obs(s, 'electricity_load_mw');
      const nuclear = obs(s, 'nuclear_output_mw');
      if (nuclear) {
        const detail = load && Number(load.value) > 0 ? `${fmt(Number(nuclear.value) / Number(load.value) * 100, 1)}% of current load` : 'current output';
        rows.push(row('Nuclear', `${fmt(nuclear.value, 0)} MW`, detail));
      }
      for (const [key, label] of [
        ['solar_output_mw', 'Solar'], ['wind_output_mw', 'Wind'], ['gas_power_output_mw', 'Gas-fired'],
        ['coal_power_output_mw', 'Coal / lignite'], ['biomass_output_mw', 'Biomass'], ['hydro_output_mw', 'Hydroelectric']
      ]) {
        const o = obs(s, key);
        if (o) rows.push(row(label, `${fmt(o.value, 0)} MW`, 'current generation output'));
      }
      const cross = obs(s, 'electricity_cross_border_mw');
      if (cross) {
        const v = Number(cross.value);
        rows.push(row(v < 0 ? 'Net imports' : 'Net exports', `${fmt(Math.abs(v), 0)} MW`, 'cross-border electricity trading'));
      }
      return block('Supporting indicators', rows);
    };

    const gas = s => {
      const rows = [];
      const fill = obs(s, 'gas_storage_fill_pct');
      if (fill) rows.push(row('Storage fill', `${fmt(fill.value, 1)}%`, 'working storage capacity filled', fill.value));
      const stored = obs(s, 'gas_in_storage_twh');
      const capacity = obs(s, 'gas_working_capacity_twh');
      if (stored && capacity) rows.push(row('Stored gas', `${fmt(stored.value, 1)} / ${fmt(capacity.value, 1)} TWh`, 'stored / working capacity'));
      else if (stored) rows.push(row('Stored gas', `${fmt(stored.value, 1)} TWh`, 'gas in storage'));
      const cover = obs(s, 'gas_storage_consumption_cover_pct');
      if (cover) rows.push(row('Annual-consumption cover', `${fmt(cover.value, 1)}%`, 'storage relative to annual consumption', cover.value));
      const injection = obs(s, 'gas_storage_injection_gwh_day');
      if (injection) rows.push(row('Injection', `${fmt(injection.value, 1)} GWh/day`, 'daily storage flow'));
      const withdrawal = obs(s, 'gas_storage_withdrawal_gwh_day');
      if (withdrawal) rows.push(row('Withdrawal', `${fmt(withdrawal.value, 1)} GWh/day`, 'daily storage flow'));
      const trend = obs(s, 'gas_storage_daily_trend_pct');
      if (trend) {
        const v = Number(trend.value);
        rows.push(row('Daily storage trend', `${v > 0 ? '+' : ''}${fmt(v, 1)}%`, 'latest reported daily change'));
      }
      const stock = obs(s, 'gas_national_stock_twh');
      if (stock) rows.push(row('National closing stock', `${fmt(stock.value, 1)} TWh`, 'Eurostat fallback measurement'));
      const proxy = obs(s, 'gas_stock_index_pct');
      if (proxy) rows.push(row('Stock index', `${fmt(proxy.value, 1)}%`, 'of trailing 36-month maximum; not capacity fill', proxy.value));
      return block('Storage indicators', rows);
    };

    const oil = s => {
      const rows = [];
      const emergency = obs(s, 'oil_emergency_stock_days');
      if (emergency) rows.push(row('Emergency stocks', `${fmt(emergency.value, 1)} days`, 'reported days equivalent'));
      const required = obs(s, 'oil_required_stock_days');
      if (required) rows.push(row('Minimum obligation', `${fmt(required.value, 1)} days`, 'reported statutory minimum series'));
      return block('Reserve indicators', rows);
    };

    const hydrology = s => {
      const rows = [];
      for (const [key, label, unit, detail] of [
        ['danube_budapest_level_cm', 'Danube · Budapest level', 'cm', 'river level'],
        ['danube_budapest_discharge_m3s', 'Danube · Budapest discharge', 'm³/s', 'river discharge'],
        ['danube_paks_level_cm', 'Danube · Paks level', 'cm', 'river level near Paks'],
        ['danube_paks_discharge_m3s', 'Danube · Paks discharge', 'm³/s', 'river discharge near Paks'],
        ['balaton_level_cm', 'Lake Balaton level', 'cm', 'average lake level']
      ]) {
        const o = obs(s, key);
        if (o) rows.push(row(label, `${fmt(o.value, 1)} ${unit}`, detail));
      }
      return block('Hydrological indicators', rows);
    };

    const weather = s => {
      const rows = [];
      for (const [key, label, unit, detail] of [
        ['weather_temperature_c', 'Current temperature', '°C', 'current local reading'],
        ['weather_wind_kmh', 'Current wind', 'km/h', 'current local wind speed'],
        ['forecast_max_temperature_c', '7-day maximum', '°C', 'forecast maximum temperature'],
        ['forecast_min_temperature_c', '7-day minimum', '°C', 'forecast minimum temperature'],
        ['forecast_max_wind_kmh', '7-day maximum wind', 'km/h', 'forecast maximum wind speed'],
        ['forecast_precipitation_7d_mm', '7-day precipitation', 'mm', 'forecast total precipitation']
      ]) {
        const o = obs(s, key);
        if (o) rows.push(row(label, `${fmt(o.value, 1)} ${unit}`, detail));
      }
      return block('Weather indicators', rows);
    };

    const indicatorHTML = (key, s) => ({electricity, gas, oil, water: hydrology, weather}[key]?.(s) || '');
    let firstPatchedRender = true;

    window.render = s => {
      const openMeasurements = firstPatchedRender ? new Set() : new Set(
        [...document.querySelectorAll('#observations .measurement-group[open] summary span:first-child')].map(el => el.textContent)
      );

      baseRender(s);

      const domainEl = document.getElementById('domains');
      domainEl.innerHTML = topDomains.map(([key, label]) => window.domainCard(key, label, s.domains?.[key])).join('');
      [...domainEl.querySelectorAll('.domain')].forEach((card, index) => {
        card.insertAdjacentHTML('beforeend', indicatorHTML(topDomains[index][0], s));
      });

      const sectionEyebrow = document.querySelector('#systems .section-head .eyebrow');
      if (sectionEyebrow) sectionEyebrow.textContent = 'Security domains';
      const hero = document.getElementById('heroText');
      const available = topDomains.filter(([key]) => s.domains?.[key]?.score != null).length;
      if (hero && available) hero.textContent = `${available} scored security domains are currently contributing. Supporting indicators provide context without being treated as separate security scores.`;

      const measurementGroups = [...document.querySelectorAll('#observations .measurement-group')];
      measurementGroups.forEach(group => group.removeAttribute('open'));
      if (!firstPatchedRender) {
        measurementGroups.forEach(group => {
          const label = group.querySelector('summary span:first-child')?.textContent;
          if (openMeasurements.has(label)) group.setAttribute('open', '');
        });
      }
      firstPatchedRender = false;
    };

    if (typeof window.load === 'function') window.load();
  });

  legacy.addEventListener('error', () => {
    const updated = document.getElementById('updated');
    if (updated) updated.textContent = 'Dashboard renderer failed to load';
  });

  document.head.appendChild(legacy);
})();
