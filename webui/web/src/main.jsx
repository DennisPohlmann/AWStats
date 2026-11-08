import { useEffect, useMemo, useState } from 'react';
import { createRoot } from 'react-dom/client';
import './style.css';
import { EChart, Card, Table, useApi, css, timeOption, barOption, donutOption, n, pct, bytes, date, ym } from './ui.jsx';
import { Visitors } from './visitors.jsx';

const TABS = ['Übersicht', 'Besucher & IPs', 'Seiten', 'Herkunft', 'Technik', 'Robots', 'Fehler', 'Prognose'];
const WEEKDAYS = ['Mo', 'Di', 'Mi', 'Do', 'Fr', 'Sa', 'So'];
const METRICS = { visits: 'Besuche', pages: 'Seiten', hits: 'Hits', bandwidth: 'Bandbreite' };
const fmtFor = m => (m === 'bandwidth' ? bytes : n);

const store = {
  get: k => { try { return localStorage.getItem(k); } catch { return null; } },
  set: (k, v) => { try { localStorage.setItem(k, v); } catch { /* private mode */ } },
};

function useTheme() {
  const [theme, setTheme] = useState(store.get('theme') || 'auto');
  const [, tick] = useState(0);
  useEffect(() => {
    if (theme === 'auto') delete document.documentElement.dataset.theme; else document.documentElement.dataset.theme = theme;
    store.set('theme', theme);
    tick(t => t + 1);
    const mq = matchMedia('(prefers-color-scheme: dark)');
    const on = () => tick(t => t + 1);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  }, [theme]);
  return [theme, setTheme];
}

// Period presets from the months available for a site.
function periods(months) {
  if (!months?.length) return [];
  const last = months[months.length - 1];
  const back = k => months[Math.max(0, months.length - k)];
  const years = [...new Set(months.map(m => m.slice(0, 4)))].reverse();
  // ids carry a preset name so presets covering the same months stay distinct entries
  return [
    { id: `cur:${last}-${last}`, label: `Aktueller Monat (${ym(last)})` },
    { id: `3m:${back(3)}-${last}`, label: 'Letzte 3 Monate' },
    { id: `12m:${back(12)}-${last}`, label: 'Letzte 12 Monate' },
    { id: `all:${months[0]}-${last}`, label: 'Gesamter Zeitraum' },
    ...years.map(y => ({ id: `y:${y}01-${y}12`, label: `Jahr ${y}` })),
    ...[...months].reverse().map(m => ({ id: `m:${m}-${m}`, label: ym(m) })),
  ];
}

// Site, period and tab live in the URL, so a view can be bookmarked or shared.
const url = new URLSearchParams(location.search);
function useUrlState(key, initial) {
  const [v, setV] = useState(url.get(key) || initial);
  useEffect(() => {
    const q = new URLSearchParams(location.search);
    if (v) q.set(key, v); else q.delete(key);
    history.replaceState(null, '', `${location.pathname}${q.size ? '?' + q : ''}${location.hash}`);
  }, [key, v]);
  return [v, setV];
}

function App() {
  const [theme, setTheme] = useTheme();
  const c = useMemo(css, [theme, document.documentElement.dataset.theme, matchMedia('(prefers-color-scheme: dark)').matches]);
  const sites = useApi('/api/sites');
  const siteNames = Object.keys(sites.data || {});
  const [site, setSite] = useUrlState('site', store.get('site') || '');
  const activeSite = siteNames.includes(site) ? site : siteNames[0];
  const ps = periods(sites.data?.[activeSite]);
  const [period, setPeriod] = useUrlState('period', '');
  // After a site switch keep the same kind of period (e.g. "last 3 months") if the exact one does not exist there.
  const activePeriod = (ps.find(p => p.id === period) || ps.find(p => p.id.split(':')[0] === period.split(':')[0] && !/^[ym]$/.test(period.split(':')[0])) || ps[1] || ps[0])?.id;
  const [tabParam, setTab] = useUrlState('tab', TABS[0]);
  const tab = TABS.includes(tabParam) ? tabParam : TABS[0];
  const [from, to] = (activePeriod || ':-').split(':')[1].split('-');
  const summary = useApi(activeSite && activePeriod ? `/api/${encodeURIComponent(activeSite)}/summary?from=${from}&to=${to}` : null);

  if (sites.error) return <main><p className="error">API nicht erreichbar: {sites.error.message}</p></main>;
  if (sites.data && !siteNames.length) return <main><p className="note">Keine AWStats-Datendateien gefunden. Prüfe AWSTATS_DATA_DIR.</p></main>;

  return (
    <>
      <header className="top">
        <div className="bar">
          <div className="brand">AWStats <span>NG</span></div>
          <label className="muted lbl" htmlFor="site">Site</label>
          <select id="site" value={activeSite || ''} onChange={e => { setSite(e.target.value); store.set('site', e.target.value); }}>
            {siteNames.map(s => <option key={s}>{s}</option>)}
          </select>
          <label className="muted lbl" htmlFor="period">Zeitraum</label>
          <select id="period" value={activePeriod || ''} onChange={e => setPeriod(e.target.value)}>
            {ps.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
          </select>
          <div className="spacer" />
          <select aria-label="Darstellung" value={theme} onChange={e => setTheme(e.target.value)}>
            <option value="auto">Automatisch</option><option value="light">Hell</option><option value="dark">Dunkel</option>
          </select>
        </div>
        <nav className="tabs" role="tablist">
          {TABS.map(t => <button key={t} role="tab" aria-selected={t === tab} onClick={e => { setTab(t); e.currentTarget.scrollIntoView({ block: 'nearest', inline: 'center' }); }}>{t}</button>)}
        </nav>
      </header>
      <main className={summary.loading ? 'loading' : ''}>
        {summary.error && <p className="error">{summary.error.message}</p>}
        {summary.data && (
          tab === 'Übersicht' ? <Overview d={summary.data} c={c} /> :
          tab === 'Besucher & IPs' ? <Visitors site={activeSite} from={from} to={to} d={summary.data} c={c} /> :
          tab === 'Seiten' ? <Pages d={summary.data} c={c} /> :
          tab === 'Herkunft' ? <Origin d={summary.data} c={c} /> :
          tab === 'Technik' ? <Tech d={summary.data} c={c} /> :
          tab === 'Robots' ? <Robots d={summary.data} c={c} /> :
          tab === 'Fehler' ? <Errors d={summary.data} c={c} /> :
          <Forecast site={activeSite} c={c} />
        )}
      </main>
    </>
  );
}

const rowsOf = s => (Array.isArray(s) ? s : s?.rows || []);
const totalOf = s => (Array.isArray(s) ? s.length : s?.total || 0);
const more = s => (totalOf(s) > rowsOf(s).length ? `Top ${rowsOf(s).length} von ${n(totalOf(s))}` : `${n(totalOf(s))} Einträge`);

function MetricSwitch({ value, onChange, options = METRICS }) {
  return <div className="tools" role="group" aria-label="Kennzahl">{Object.entries(options).map(([k, v]) =>
    <button key={k} className="btn" aria-pressed={value === k} onClick={() => onChange(k)}>{v}</button>)}</div>;
}

function Overview({ d, c }) {
  const [metric, setMetric] = useState('visits');
  const days = d.sections.DAY || [];
  const g = d.general;
  const sum = k => days.reduce((s, r) => s + (r[k] || 0), 0);
  const fmt = fmtFor(metric);
  const byMonth = Object.values(days.reduce((a, r) => { const k = r.date.slice(0, 6); (a[k] ||= { k, v: 0 }).v += r[metric] || 0; return a; }, {}));
  const wd = WEEKDAYS.map(() => ({ v: 0, n: 0 }));
  for (const r of days) { const i = (new Date(Date.UTC(+r.date.slice(0, 4), +r.date.slice(4, 6) - 1, +r.date.slice(6, 8))).getUTCDay() + 6) % 7; wd[i].v += r[metric] || 0; wd[i].n++; }
  const hours = d.sections.TIME || [];
  const hourMetric = metric === 'visits' ? 'pages' : metric; // AWStats stores no visits per hour
  const sessOrder = ['0s-30s', '30s-2mn', '2mn-5mn', '5mn-15mn', '15mn-30mn', '30mn-1h', '1h+'];
  const sess = [...(d.sections.SESSION || [])].sort((a, b) => sessOrder.indexOf(a.range) - sessOrder.indexOf(b.range));

  return (
    <>
      <div className="kpis">
        <Kpi label="Besuche" value={n(g.TotalVisits)} />
        <Kpi label={g.TotalUnique != null ? 'Eindeutige Besucher' : 'Verschiedene Hosts/IPs'} value={n(g.TotalUnique ?? g.DistinctHosts)} hint={g.TotalUnique == null ? 'über den ganzen Zeitraum' : undefined} />
        <Kpi label="Seiten" value={n(sum('pages'))} />
        <Kpi label="Hits" value={n(sum('hits'))} />
        <Kpi label="Bandbreite" value={bytes(sum('bandwidth'))} />
        <Kpi label="Letztes Update" value={date(g.LastUpdate, false)} hint={date(g.LastUpdate).slice(11)} />
      </div>
      <MetricSwitch value={metric} onChange={setMetric} />
      <div className="grid">
        <Card title={`${METRICS[metric]} pro Tag`} sub={`${days.length} Tage · Mausrad/Regler zum Zoomen`} wide>
          <EChart className="chart tall" label={`${METRICS[metric]} pro Tag`} option={timeOption(c, { x: days.map(r => date(r.date, false)), series: [{ name: METRICS[metric], data: days.map(r => r[metric]), area: true }], fmt, zoom: days.length > 40 })} />
        </Card>
        {byMonth.length > 1 && <Card title={`${METRICS[metric]} pro Monat`}>
          <EChart label="pro Monat" option={barOption(c, { x: byMonth.map(m => ym(m.k)), y: byMonth.map(m => m.v), name: METRICS[metric], fmt })} />
        </Card>}
        <Card title={`Ø ${METRICS[metric]} pro Wochentag`}>
          <EChart label="pro Wochentag" option={barOption(c, { x: WEEKDAYS, y: wd.map(w => (w.n ? Math.round(w.v / w.n) : 0)), name: METRICS[metric], fmt })} />
        </Card>
        <Card title={`${METRICS[hourMetric]} nach Uhrzeit`} sub={metric === 'visits' ? 'AWStats speichert Besuche nicht pro Stunde, daher Seiten' : 'Summe über den Zeitraum'}>
          <EChart label="nach Uhrzeit" option={barOption(c, { x: hours.map(h => `${String(h.hour).padStart(2, '0')}h`), y: hours.map(h => h[hourMetric]), name: METRICS[hourMetric], fmt: fmtFor(hourMetric) })} />
        </Card>
        <Card title="Besuchsdauer" sub="Anzahl Besuche je Dauer">
          <EChart label="Besuchsdauer" option={barOption(c, { x: sess.map(s => s.range.replace(/mn/g, ' min')), y: sess.map(s => s.visits), name: 'Besuche' })} />
        </Card>
      </div>
      <Card title="Tageswerte" sub="Tabelle zu den Diagrammen">
        <details><summary>Tabelle anzeigen</summary>
          <Table rows={days} initialSort={{ key: 'date', dir: -1 }} cols={[{ key: 'date', label: 'Tag', fmt: v => date(v, false) }, { key: 'visits', label: 'Besuche', num: true, fmt: n }, { key: 'pages', label: 'Seiten', num: true, fmt: n }, { key: 'hits', label: 'Hits', num: true, fmt: n }, { key: 'bandwidth', label: 'Bandbreite', num: true, fmt: bytes }]} />
        </details>
      </Card>
    </>
  );
}

function Kpi({ label, value, hint }) {
  return <div className="kpi"><div className="label">{label}</div><div className="value">{value}</div>{hint && <div className="hint">{hint}</div>}</div>;
}

// Donut + full table for one AWStats section.
function Breakdown({ title, s, c, value = 'hits', group, cols, sub }) {
  let rows = rowsOf(s);
  let donutRows = rows;
  if (group) { // e.g. browser families instead of every version
    const m = {};
    for (const r of rows) { const k = r[group]; (m[k] ||= { label: k, [value]: 0 })[value] += r[value] || 0; }
    donutRows = Object.values(m);
  }
  const total = s?.sum?.[value] ?? rows.reduce((a, r) => a + (r[value] || 0), 0); // sum over all rows, not just the top ones sent
  return (
    <Card title={title} sub={sub || more(s)}>
      {donutRows.filter(r => r[value]).length > 1
        ? <EChart label={title} option={donutOption(c, donutRows, { value: r => r[value] || 0, total })} />
        : <p className="note">{donutRows[0] ? `100 % ${donutRows[0].label || donutRows[0].id}` : 'Keine Daten in diesem Zeitraum.'}</p>}
      <details><summary>Alle Einträge</summary>
        <Table rows={rows} initialSort={{ key: value, dir: -1 }} cols={cols || [
          { key: 'label', label: 'Name', render: r => r.label || r.id }, { key: value, label: value === 'pages' ? 'Seiten' : 'Hits', num: true, fmt: n },
          { key: '_p', label: 'Anteil', num: true, render: r => pct(r[value], total) },
        ]} />
      </details>
    </Card>
  );
}

function Pages({ d, c }) {
  const s = d.sections;
  const top = rowsOf(s.SIDER).slice(0, 15);
  return (
    <div className="grid">
      <Card title="Meistbesuchte Seiten" sub={more(s.SIDER)} wide>
        <EChart className="chart tall" label="Top Seiten" option={barOption(c, { x: top.map(r => r.url), y: top.map(r => r.pages), name: 'Seitenaufrufe', horizontal: true })} />
        <Table rows={rowsOf(s.SIDER)} initialSort={{ key: 'pages', dir: -1 }} cols={[
          { key: 'url', label: 'URL', wrap: true }, { key: 'pages', label: 'Aufrufe', num: true, fmt: n }, { key: 'bandwidth', label: 'Bandbreite', num: true, fmt: bytes },
          { key: 'entry', label: 'Einstieg', num: true, fmt: n }, { key: 'exit', label: 'Ausstieg', num: true, fmt: n }]} />
      </Card>
      <Breakdown title="Dateitypen" s={s.FILETYPES} c={c} />
      <Card title="Downloads" sub={more(s.DOWNLOADS)}>
        <Table rows={rowsOf(s.DOWNLOADS)} initialSort={{ key: 'hits', dir: -1 }} cols={[{ key: 'url', label: 'Datei', wrap: true }, { key: 'hits', label: 'Hits', num: true, fmt: n }, { key: 'hits206', label: 'Teil (206)', num: true, fmt: n }, { key: 'bandwidth', label: 'Bandbreite', num: true, fmt: bytes }]} />
      </Card>
    </div>
  );
}

function Origin({ d, c }) {
  const s = d.sections;
  return (
    <div className="grid">
      <Breakdown title="Woher die Besucher kamen" s={s.ORIGIN} c={c} value="pages" sub="Seitenaufrufe nach Herkunft" />
      <Breakdown title="Suchmaschinen" s={s.SEREFERRALS} c={c} value="pages" />
      <Card title="Verweisende Seiten" sub={more(s.PAGEREFS)}>
        <Table rows={rowsOf(s.PAGEREFS)} initialSort={{ key: 'pages', dir: -1 }} cols={[{ key: 'url', label: 'Seite', wrap: true }, { key: 'pages', label: 'Seiten', num: true, fmt: n }, { key: 'hits', label: 'Hits', num: true, fmt: n }]} />
      </Card>
      <Card title="Suchbegriffe (Phrasen)" sub={more(s.SEARCHWORDS)}>
        <Table rows={rowsOf(s.SEARCHWORDS).map(r => ({ ...r, phrase: decode(r.phrase) }))} initialSort={{ key: 'count', dir: -1 }} cols={[{ key: 'phrase', label: 'Phrase', wrap: true }, { key: 'count', label: 'Suchen', num: true, fmt: n }]} />
      </Card>
      <Card title="Suchbegriffe (Wörter)" sub={more(s.KEYWORDS)}>
        <Table rows={rowsOf(s.KEYWORDS).map(r => ({ ...r, word: decode(r.word) }))} initialSort={{ key: 'count', dir: -1 }} cols={[{ key: 'word', label: 'Wort' }, { key: 'count', label: 'Suchen', num: true, fmt: n }]} />
      </Card>
    </div>
  );
}
const decode = v => { try { return decodeURIComponent(String(v).replace(/\+/g, ' ')); } catch { return String(v); } };

function Tech({ d, c }) {
  const s = d.sections;
  return (
    <div className="grid">
      <Breakdown title="Browser" s={s.BROWSER} c={c} group="group" sub="Donut nach Browser, Tabelle mit Versionen" />
      <Breakdown title="Betriebssysteme" s={s.OS} c={c} />
      <Breakdown title="Länder / Domains" s={s.DOMAIN} c={c} value="pages" sub="Ohne GeoIP-Plugin oder DNS-Lookup bleibt das meist „Unbekannt“" />
      {rowsOf(s.SCREENSIZE).length > 0 && <Breakdown title="Bildschirmgrößen" s={s.SCREENSIZE} c={c} />}
    </div>
  );
}

function Robots({ d, c }) {
  const s = d.sections;
  const top = rowsOf(s.ROBOT).slice(0, 12);
  return (
    <div className="grid">
      <Card title="Robots und Spider" sub={more(s.ROBOT)} wide>
        {top.length > 0 && <EChart label="Robots" option={barOption(c, { x: top.map(r => r.label), y: top.map(r => r.hits), name: 'Hits', horizontal: true })} />}
        <Table rows={rowsOf(s.ROBOT)} initialSort={{ key: 'hits', dir: -1 }} cols={[{ key: 'label', label: 'Robot' }, { key: 'hits', label: 'Hits', num: true, fmt: n }, { key: 'bandwidth', label: 'Bandbreite', num: true, fmt: bytes }, { key: 'robotsTxtHits', label: 'robots.txt', num: true, fmt: n }, { key: 'lastVisit', label: 'Letzter Besuch', fmt: v => date(v) }]} />
      </Card>
      {rowsOf(s.WORMS).length > 0 && <Card title="Würmer" sub={more(s.WORMS)}>
        <Table rows={rowsOf(s.WORMS)} cols={[{ key: 'label', label: 'Wurm' }, { key: 'hits', label: 'Hits', num: true, fmt: n }, { key: 'lastVisit', label: 'Letzter Besuch', fmt: v => date(v) }]} />
      </Card>}
    </div>
  );
}

function Errors({ d, c }) {
  const s = d.sections;
  const codes = rowsOf(s.ERRORS);
  const errPages = Object.keys(s).filter(k => k.startsWith('SIDER_') && rowsOf(s[k]).length);
  return (
    <div className="grid">
      <Card title="HTTP-Fehlercodes" sub={more(s.ERRORS)}>
        {codes.length > 0 && <EChart label="Fehlercodes" option={barOption(c, { x: codes.map(r => String(r.code)), y: codes.map(r => r.hits), name: 'Hits', color: getComputedStyle(document.documentElement).getPropertyValue('--critical').trim() })} />}
        <Table rows={codes} initialSort={{ key: 'hits', dir: -1 }} cols={[{ key: 'code', label: 'Code', render: r => <span className={`status s${String(r.code)[0]}`}>{r.code}</span> }, { key: 'label', label: 'Bedeutung' }, { key: 'hits', label: 'Hits', num: true, fmt: n }, { key: 'bandwidth', label: 'Bandbreite', num: true, fmt: bytes }]} />
      </Card>
      {errPages.map(k => (
        <Card key={k} title={`URLs mit Fehler ${k.slice(6)}`} sub={more(s[k])}>
          <Table rows={rowsOf(s[k])} initialSort={{ key: 'hits', dir: -1 }} cols={[{ key: 'url', label: 'URL', wrap: true }, { key: 'hits', label: 'Hits', num: true, fmt: n }, { key: 'referrer', label: 'Letzter Verweis', wrap: true }]} />
        </Card>
      ))}
    </div>
  );
}

function Forecast({ site, c }) {
  const [metric, setMetric] = useState('visits');
  const [days, setDays] = useState(30);
  const f = useApi(`/api/${encodeURIComponent(site)}/forecast?metric=${metric}&days=${days}`);
  const fmt = fmtFor(metric);
  const option = useMemo(() => {
    if (!f.data?.forecast) return null;
    const h = f.data.history, p = f.data.forecast;
    const x = [...h.map(r => r.date), ...p.dates].map(v => date(v, false));
    const pad = Array(h.length).fill(null);
    return timeOption(c, {
      x, fmt, zoom: true,
      series: [
        { name: 'Tatsächlich', data: [...h.map(r => r.value), ...Array(p.dates.length).fill(null)] },
        // starts at the last real value so both lines connect
        { name: 'Prognose', dashed: true, color: c.series[1], data: [...pad.slice(1), h[h.length - 1].value, ...p.mean.map(Math.round)],
          band: [...pad.slice(1).map(() => [null, null]), [h[h.length - 1].value, h[h.length - 1].value], ...p.lower.map((lo, i) => [lo, p.upper[i]])] },
      ],
    });
  }, [f.data, c, fmt]);
  const sumFc = f.data?.forecast?.mean.reduce((a, b) => a + b, 0);
  return (
    <>
      <MetricSwitch value={metric} onChange={setMetric} />
      <div className="tools" role="group" aria-label="Horizont">{[14, 30, 90].map(k => <button key={k} className="btn" aria-pressed={days === k} onClick={() => setDays(k)}>{k} Tage</button>)}</div>
      {f.error && <p className="error">{f.error.message}</p>}
      {f.data && !f.data.forecast && <p className="note">{f.data.note}</p>}
      {option && (
        <div className="grid">
          <div className="kpis" style={{ gridColumn: '1 / -1' }}>
            <Kpi label={`Erwartet, nächste ${days} Tage`} value={fmt(Math.round(sumFc))} hint={METRICS[metric]} />
            <Kpi label="Ø pro Tag (Prognose)" value={fmt(Math.round(sumFc / days))} />
            <Kpi label="Ø pro Tag (letzte 28 Tage)" value={fmt(Math.round(f.data.history.slice(-28).reduce((a, r) => a + r.value, 0) / Math.min(28, f.data.history.length)))} />
          </div>
          <Card title={`Prognose ${METRICS[metric]}`} sub="Holt-Winters (Trend + Wochenmuster), Band = 95 % Unsicherheit. Grundlage: alle Tageswerte dieser Site." wide>
            <EChart className="chart tall" label="Prognose" option={option} />
          </Card>
        </div>
      )}
    </>
  );
}

createRoot(document.getElementById('root')).render(<App />);
