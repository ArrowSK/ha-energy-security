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
      #signals{grid-template-columns:1fr}
      #signals>article:last-child{display:none}
      .domain-indicators{margin-top:14px;padding-top:10px;border-top:1px solid rgba(255,255,255,.065)}
      .domain-indicators>summary{list-style:none;cursor:pointer;display:flex;align-items:center;justify-content:space-between;gap:10px;padding:4px 0;color:var(--muted);font-size:.72rem;font-weight:780;letter-spacing:.1em;text-transform:uppercase}
      .domain-indicators>summary::-webkit-details-marker{display:none}
      .domain-indicators>summary:after{content:"⌄";font-size:1rem;letter-spacing:0;transition:.18s}
      .domain-indicators[open]>summary:after{transform:rotate(180deg)}
      .indicator-list{display:grid;gap:9px;padding-top:11px}
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
        .domain-indicators>summary{font-size:.76rem}
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
    const block = (domain, title, rows) => rows.length
      ? `<details class="domain-indicators" data-domain="${esc(domain)}"><summary><span>${esc(title)}</span><span>${rows.length}</span></summary><div class="indicator-list">${rows.join('')}</div></details>`
      : '';

    const diversity = s => {
      const mix = obs(s, 'electricity_generation_mw')?.attributes?.mix_mw || {};
      const vals = Object.values(mix).map(Number).filter(v => Number.isFinite(v) && v > 0);
      if (vals.length < 2) return null;
      const total = vals.reduce((a, v) => a + v, 0);
      if (total <= 0) return null;
      const hhi = vals.reduce((a, v) => {
        const p = v / total;
        return a + p * p;
      }, 0);
      return clamp((1 - hhi) / (1 - 1 / vals.length) * 100);
    };

    const electricity = s => {
      const rows = [];
      const generation = obs(s, 'electricity_generation_mw');
      const load = obs(s, 'electricity_load_mw');
      const loadValue = load && Number(load.value) > 0 ? Number(load.value) : null;
      const generationValue = generation && Number(generation.value) > 0 ? Number(generation.value) : null;
      const denominator = loadValue || generationValue;
      const shareBasis = loadValue ? 'current load' : 'current generation';

      const div = diversity(s);
      if (div != null) {
        rows.push(row(
          'Generation diversity',
          `${Math.round(div)}/100`,
          'normalized diversity of the current generation mix',
          div,
          tone(div)
        ));
      }

      const renew = obs(s, 'renewable_share_pct');
      if (renew) {
        rows.push(row(
          'Renewables',
          `${fmt(renew.value, 1)}%`,
          'share of current load',
          renew.value
        ));
      }

      const mix = generation?.attributes?.mix_mw || {};
      const components = Object.entries(mix)
        .map(([name, value]) => [name, Number(value)])
        .filter(([, value]) => Number.isFinite(value) && value > 0)
        .sort((a, b) => b[1] - a[1]);

      for (const [name, value] of components) {
        const share = denominator ? value / denominator * 100 : null;
        const detail = share == null ? 'current generation output' : `${fmt(share, 1)}% of ${shareBasis}`;
        rows.push(row(name, `${fmt(value, 0)} MW`, detail, share));
      }

      const cross = obs(s, 'electricity_cross_border_mw');
      if (cross) {
        const value = Number(cross.value);
        const share = loadValue ? Math.abs(value) / loadValue * 100 : null;
        rows.push(row(
          value < 0 ? 'Net imports' : 'Net exports',
          `${fmt(Math.abs(value), 0)} MW`,
          share == null ? 'cross-border electricity trading' : `${fmt(share, 1)}% of current load`,
          share
        ));
      }

      return block('electricity', 'Supporting indicators', rows);
    };

    const gas = s => {
      const rows = [];
      const fill = obs(s, 'gas_storage_fill_pct');
      if (fill) rows.push(row('Storage fill', `${fmt(fill.value, 1)}%`, 'working storage capacity filled', fill.value));

      const stored = obs(s, 'gas_in_storage_twh');
      const capacity = obs(s, 'gas_working_capacity_twh');
      if (stored && capacity) {
        const capacityValue = Number(capacity.value);
        const share = capacityValue > 0 ? Number(stored.value) / capacityValue * 100 : null;
        rows.push(row(
          'Stored gas',
          `${fmt(stored.value, 1)} / ${fmt(capacity.value, 1)} TWh`,
          share == null ? 'stored / working capacity' : `${fmt(share, 1)}% of working capacity`,
          share
        ));
      } else if (stored) {
        rows.push(row('Stored gas', `${fmt(stored.value, 1)} TWh`, 'gas in storage'));
      } else if (capacity) {
        rows.push(row('Working capacity', `${fmt(capacity.value, 1)} TWh`, 'reported storage capacity'));
      }

      const cover = obs(s, 'gas_storage_consumption_cover_pct');
      if (cover) rows.push(row('Annual-consumption cover', `${fmt(cover.value, 1)}%`, 'storage relative to annual consumption', cover.value));

      const injection = obs(s, 'gas_storage_injection_gwh_day');
      if (injection) rows.push(row('Injection', `${fmt(injection.value, 1)} GWh/day`, 'daily storage flow'));
      const withdrawal = obs(s, 'gas_storage_withdrawal_gwh_day');
      if (withdrawal) rows.push(row('Withdrawal', `${fmt(withdrawal.value, 1)} GWh/day`, 'daily storage flow'));
      const trend = obs(s, 'gas_storage_daily_trend_pct');
      if (trend) {
        const value = Number(trend.value);
        rows.push(row('Daily storage trend', `${value > 0 ? '+' : ''}${fmt(value, 1)}%`, 'latest reported daily change'));
      }

      const stock = obs(s, 'gas_national_stock_twh');
      if (stock) rows.push(row('National closing stock', `${fmt(stock.value, 1)} TWh`, 'Eurostat fallback measurement'));
      const proxy = obs(s, 'gas_stock_index_pct');
      if (proxy) rows.push(row('Stock index', `${fmt(proxy.value, 1)}%`, 'of trailing 36-month maximum; not capacity fill', proxy.value));

      return block('gas', 'Storage indicators', rows);
    };

    const oil = s => {
      const rows = [];
      const emergency = obs(s, 'oil_emergency_stock_days');
      if (emergency) rows.push(row('Emergency stocks', `${fmt(emergency.value, 1)} days`, 'reported days equivalent'));
      const required = obs(s, 'oil_required_stock_days');
      if (required) rows.push(row('Minimum obligation', `${fmt(required.value, 1)} days`, 'reported statutory minimum series'));
      return block('oil', 'Reserve indicators', rows);
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
      return block('water', 'Hydrological indicators', rows);
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
      return block('weather', 'Weather indicators', rows);
    };

    const indicatorHTML = (key, s) => ({electricity, gas, oil, water: hydrology, weather}[key]?.(s) || '');
    let firstPatchedRender = true;

    window.render = s => {
      const openMeasurements = firstPatchedRender
        ? new Set()
        : new Set(
            [...document.querySelectorAll('#observations .measurement-group[open] summary span:first-child')]
              .map(el => el.textContent)
          );
      const openSupports = firstPatchedRender
        ? new Set()
        : new Set(
            [...document.querySelectorAll('#domains .domain-indicators[open]')]
              .map(el => el.dataset.domain)
              .filter(Boolean)
          );

      baseRender(s);

      const domainEl = document.getElementById('domains');
      domainEl.innerHTML = topDomains.map(([key, label]) => window.domainCard(key, label, s.domains?.[key])).join('');
      [...domainEl.querySelectorAll('.domain')].forEach((card, index) => {
        const key = topDomains[index][0];
        card.insertAdjacentHTML('beforeend', indicatorHTML(key, s));
        if (openSupports.has(key)) {
          card.querySelector('.domain-indicators')?.setAttribute('open', '');
        }
      });

      const sectionEyebrow = document.querySelector('#systems .section-head .eyebrow');
      if (sectionEyebrow) sectionEyebrow.textContent = 'Security domains';

      const hero = document.getElementById('heroText');
      const available = topDomains.filter(([key]) => s.domains?.[key]?.score != null).length;
      if (hero && available) {
        hero.textContent = `${available} scored security domains are currently contributing. Supporting indicators are available on demand and are not treated as separate security scores.`;
      }

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
