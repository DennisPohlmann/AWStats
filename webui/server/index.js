// AWStats NG server: JSON API on top of AWStats data files + static web UI.
const http = require('http');
const fs = require('fs');
const path = require('path');
const { listSites, load } = require('./datafile');
const { forecast } = require('./forecast');
const { logFileFromConf, logFiles, searchIp } = require('./logsearch');
const L = require('./labels.json');

const DATA_DIR = process.env.AWSTATS_DATA_DIR || path.join(__dirname, '../testdata/data');
const CONF_DIR = process.env.AWSTATS_CONF_DIR || '/etc/awstats';
const WEB_DIR = path.join(__dirname, '../web/dist');
const PORT = +process.env.PORT || 3000;
const HOST = process.env.HOST || '127.0.0.1'; // the UI shows visitor IPs: keep it local, publish via reverse proxy with auth

// ---- labels --------------------------------------------------------------------------------
const browserKeys = Object.keys(L.browsers).sort((a, b) => b.length - a.length);
function browserLabel(id) {
  const k = browserKeys.find(k => id.startsWith(k));
  if (!k) return { label: id, group: id };
  const ver = id.slice(k.length).replace(/^_/, '');
  return { label: ver ? `${L.browsers[k]} ${ver}` : L.browsers[k], group: L.browsers[k] };
}
const ORIGIN = { From0: 'Direkt / Lesezeichen', From1: 'Unbekannt', From2: 'Suchmaschinen', From3: 'Externe Seiten', From4: 'Interne Seiten', From5: 'Newsgroups' };
const LABEL = {
  OS: id => ({ label: L.os[id] || id }),
  BROWSER: browserLabel,
  ROBOT: id => ({ label: L.robots[id] || id }),
  WORMS: id => ({ label: L.worms[id] || id }),
  SEREFERRALS: id => ({ label: L.searchEngines[id] || id.replace(/_/g, '.') }),
  DOMAIN: id => ({ label: id === 'ip' ? 'Unbekannt (nur IP)' : L.domains[id] || id }),
  ERRORS: id => ({ label: L.http[id] || '' }),
  ORIGIN: id => ({ label: ORIGIN[id] || id }),
};

// ---- aggregation over months -----------------------------------------------------------------
const MAX_KEYS = ['lastVisit'];
const LATEST_KEYS = ['startLastVisit', 'lastPage', 'referrer'];

function aggregate(months) {
  const general = { TotalVisits: 0, months: months.length, FirstTime: null, LastTime: null, LastUpdate: null };
  const sections = {};
  const index = {};
  for (const m of months) {
    const g = m.data.general;
    general.TotalVisits += g.TotalVisits || 0;
    general.FirstTime ??= g.FirstTime;
    if (g.LastTime) general.LastTime = g.LastTime;
    if (g.LastUpdate) general.LastUpdate = Array.isArray(g.LastUpdate) ? g.LastUpdate[0] : g.LastUpdate;
    if (months.length === 1) general.TotalUnique = g.TotalUnique;
    for (const [name, rows] of Object.entries(m.data.sections)) {
      const out = (sections[name] ||= []);
      if (name === 'DAY') { out.push(...rows); continue; }
      const idx = (index[name] ||= new Map());
      for (const r of rows) {
        if (Array.isArray(r)) { out.push(r); continue; } // unknown section layout
        const key = Object.values(r)[0];
        const cur = idx.get(key);
        if (!cur) { const c = { ...r }; idx.set(key, c); out.push(c); continue; }
        for (const [k, v] of Object.entries(r)) {
          if (k === Object.keys(r)[0]) continue;
          if (MAX_KEYS.includes(k)) cur[k] = Math.max(cur[k] || 0, v || 0);
          else if (LATEST_KEYS.includes(k)) cur[k] = v;
          else if (typeof v === 'number') cur[k] = (cur[k] || 0) + v;
        }
      }
    }
  }
  if (sections.DAY) sections.DAY.sort((a, b) => a.date.localeCompare(b.date));
  // Distinct hosts over the whole range (AWStats' own "unique visitors" is per month only).
  general.DistinctHosts = sections.VISITOR ? sections.VISITOR.length : 0;
  for (const [name, fn] of Object.entries(LABEL)) for (const r of sections[name] || []) Object.assign(r, fn(Object.values(r)[0]));
  return { general, sections };
}

const SORT_COL = r => r.pages ?? r.hits ?? r.count ?? r.visits ?? 0;
const UNSORTED = new Set(['DAY', 'TIME', 'SESSION', 'FILESIZE', 'REQUESTTIME', 'ORIGIN', 'MISC']);
const TOP = 100;

// Aggregates are cached per range and reused while no data file changed (load() returns the same objects).
const aggCache = new Map();
function aggregateCached(site, months) {
  const key = site + '|' + months.map(m => m.ym).join(',');
  const hit = aggCache.get(key);
  if (hit && hit.refs.length === months.length && hit.refs.every((r, i) => r === months[i].data)) return hit.agg;
  const agg = aggregate(months);
  if (aggCache.size > 50) aggCache.clear(); // shortcut: crude bound, fine for a handful of sites
  aggCache.set(key, { refs: months.map(m => m.data), agg });
  return agg;
}

function rangeMonths(site, q, sites) {
  const all = sites[site];
  const to = q.get('to') || all[all.length - 1];
  const from = q.get('from') || to;
  if (!/^\d{6}$/.test(from) || !/^\d{6}$/.test(to)) throw httpError(400, 'from/to must be YYYYMM');
  return all.filter(ym => ym >= from && ym <= to).map(ym => ({ ym, data: load(DATA_DIR, site, ym) }));
}

// ---- routes ---------------------------------------------------------------------------------
function httpError(status, message) { return Object.assign(new Error(message), { status }); }

async function api(parts, q) {
  const sites = listSites(DATA_DIR);
  if (parts[0] === 'sites') return sites;
  const [site, action, arg] = parts;
  if (!Object.hasOwn(sites, site)) throw httpError(404, `unknown site ${site}`);

  if (action === 'summary') {
    // Top lists only, so a busy site's 100k hosts do not end up in the browser at once.
    const { general, sections } = aggregateCached(site, rangeMonths(site, q, sites));
    const out = {};
    for (const [name, rows] of Object.entries(sections)) {
      if (UNSORTED.has(name)) { out[name] = rows; continue; }
      const sum = {}; // column sums over all rows, so shares stay right although only the top rows are sent
      for (const r of rows) for (const [k, v] of Object.entries(r)) if (typeof v === 'number' && !MAX_KEYS.includes(k)) sum[k] = (sum[k] || 0) + v;
      out[name] = { total: rows.length, sum, rows: [...rows].sort((a, b) => SORT_COL(b) - SORT_COL(a)).slice(0, TOP) };
    }
    return { general, sections: out };
  }

  if (action === 'list') {
    // Full, searchable, paged list of one section, e.g. /api/site/list/VISITOR?q=84.1&offset=0&limit=50
    if (!/^[A-Z_0-9]+$/.test(arg || '')) throw httpError(400, 'bad section');
    const { sections } = aggregateCached(site, rangeMonths(site, q, sites));
    const needle = (q.get('q') || '').toLowerCase();
    const sort = q.get('sort');
    let rows = sections[arg] || [];
    if (needle) rows = rows.filter(r => Object.values(r).some(v => String(v).toLowerCase().includes(needle)));
    rows = [...rows].sort(sort ? (a, b) => (+b[sort] || 0) - (+a[sort] || 0) : (a, b) => SORT_COL(b) - SORT_COL(a));
    const offset = Math.max(0, +q.get('offset') || 0), limit = Math.min(1000, +q.get('limit') || 50);
    return { total: rows.length, rows: rows.slice(offset, offset + limit) };
  }

  if (action === 'forecast') {
    const metric = q.get('metric') || 'visits';
    if (!['visits', 'pages', 'hits', 'bandwidth'].includes(metric)) throw httpError(400, 'bad metric');
    const h = Math.min(365, Math.max(1, +q.get('days') || 30));
    const days = aggregateCached(site, sites[site].map(ym => ({ ym, data: load(DATA_DIR, site, ym) }))).sections.DAY || [];
    // Fill missing days with 0 and drop the last (possibly incomplete) day.
    const series = [];
    if (days.length) {
      const byDate = new Map(days.map(d => [d.date, d[metric] || 0]));
      const d = toDate(days[0].date), end = toDate(days[days.length - 1].date);
      for (; d < end; d.setUTCDate(d.getUTCDate() + 1)) series.push({ date: fmt(d), value: byDate.get(fmt(d)) || 0 });
    }
    const f = forecast(series.map(s => s.value), h);
    if (!f) return { history: series, forecast: null, note: 'Mindestens 14 Tage Historie nötig' };
    const last = toDate(series[series.length - 1].date);
    const dates = f.mean.map((_, i) => { const d = new Date(last); d.setUTCDate(d.getUTCDate() + i + 1); return fmt(d); });
    return { history: series, forecast: { dates, ...f } };
  }

  if (action === 'ip') {
    const ip = arg || '';
    if (!/^[\w.:-]{1,255}$/.test(ip)) throw httpError(400, 'bad ip/host');
    const months = sites[site].map(ym => {
      const r = (load(DATA_DIR, site, ym).sections.VISITOR || []).find(v => v.host === ip);
      return r && { month: ym, ...r };
    }).filter(Boolean);
    let log = { hits: [], truncated: false, files: [], note: null };
    const configured = process.env.AWSTATS_LOG_FILES ? process.env.AWSTATS_LOG_FILES.split(',') : null;
    const lf = configured ? null : logFileFromConf(CONF_DIR, site);
    const files = configured || (lf ? logFiles(lf) : []);
    if (!files.length) log.note = 'Keine lesbaren Logdateien gefunden (LogFile in der AWStats-Config oder AWSTATS_LOG_FILES setzen)';
    else log = { ...(await searchIp(files, ip)), files: files.map(f => path.basename(f)), note: null };
    return { host: ip, months, log };
  }

  throw httpError(404, 'unknown endpoint');
}

const toDate = s => new Date(Date.UTC(+s.slice(0, 4), +s.slice(4, 6) - 1, +s.slice(6, 8)));
const fmt = d => d.toISOString().slice(0, 10).replace(/-/g, '');

const MIME = { '.html': 'text/html', '.js': 'text/javascript', '.css': 'text/css', '.svg': 'image/svg+xml', '.json': 'application/json', '.ico': 'image/x-icon', '.png': 'image/png' };

const server = http.createServer(async (req, res) => {
  const url = new URL(req.url, 'http://x');
  try {
    if (url.pathname.startsWith('/api/')) {
      let parts;
      try { parts = url.pathname.slice(5).split('/').map(decodeURIComponent); } catch { throw httpError(400, 'bad url encoding'); }
      const body = JSON.stringify(await api(parts, url.searchParams));
      res.writeHead(200, { 'content-type': 'application/json; charset=utf-8' });
      return res.end(body);
    }
    // static files of the built web UI, SPA fallback to index.html
    let file = path.normalize(path.join(WEB_DIR, url.pathname));
    if (!file.startsWith(WEB_DIR + path.sep) || !fs.existsSync(file) || fs.statSync(file).isDirectory()) file = path.join(WEB_DIR, 'index.html');
    if (!fs.existsSync(file)) { res.writeHead(404); return res.end('web UI not built: run "npm run build"'); }
    res.writeHead(200, { 'content-type': MIME[path.extname(file)] || 'application/octet-stream' });
    fs.createReadStream(file).pipe(res);
  } catch (e) {
    res.writeHead(e.status || 500, { 'content-type': 'application/json' });
    res.end(JSON.stringify({ error: e.status ? e.message : 'internal error' }));
    if (!e.status) console.error(e);
  }
});

if (require.main === module) server.listen(PORT, HOST, () => console.log(`AWStats NG on http://${HOST}:${PORT}  data: ${DATA_DIR}`));
module.exports = { api, aggregate, server };
