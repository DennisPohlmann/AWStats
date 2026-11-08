// Reader for AWStats text data files (awstatsMMYYYY[.config].txt), as written by awstats.pl 7.x/8.x.
const fs = require('fs');
const path = require('path');

// Column names per section, taken from the writer in awstats.pl (the comment lines in the file are not
// always accurate, e.g. OS has Hits and Pages). Unknown sections keep generic column names.
const SCHEMA = {
  GENERAL: null, // key/value, handled separately
  MISC: ['id', 'pages', 'hits', 'bandwidth'],
  TIME: ['hour', 'pages', 'hits', 'bandwidth', 'nvPages', 'nvHits', 'nvBandwidth'],
  VISITOR: ['host', 'pages', 'hits', 'bandwidth', 'lastVisit', 'startLastVisit', 'lastPage'],
  DAY: ['date', 'pages', 'hits', 'bandwidth', 'visits'],
  DOMAIN: ['id', 'pages', 'hits', 'bandwidth'],
  CLUSTER: ['id', 'pages', 'hits', 'bandwidth'],
  LOGIN: ['id', 'pages', 'hits', 'bandwidth', 'lastVisit'],
  ROBOT: ['id', 'hits', 'bandwidth', 'lastVisit', 'robotsTxtHits'],
  WORMS: ['id', 'hits', 'bandwidth', 'lastVisit'],
  EMAILSENDER: ['id', 'hits', 'bandwidth', 'lastVisit'],
  EMAILRECEIVER: ['id', 'hits', 'bandwidth', 'lastVisit'],
  SESSION: ['range', 'visits'],
  SIDER: ['url', 'pages', 'bandwidth', 'entry', 'exit'],
  FILETYPES: ['id', 'hits', 'bandwidth', 'bwWithoutCompression', 'bwAfterCompression'],
  DOWNLOADS: ['url', 'hits', 'hits206', 'bandwidth'],
  OS: ['id', 'hits', 'pages'],
  BROWSER: ['id', 'hits', 'pages'],
  SCREENSIZE: ['id', 'hits'],
  UNKNOWNREFERER: ['id', 'lastVisit'],
  UNKNOWNREFERERBROWSER: ['id', 'lastVisit'],
  ORIGIN: ['id', 'pages', 'hits'],
  SEREFERRALS: ['id', 'pages', 'hits'],
  PAGEREFS: ['url', 'pages', 'hits'],
  SEARCHWORDS: ['phrase', 'count'],
  KEYWORDS: ['word', 'count'],
  ERRORS: ['code', 'hits', 'bandwidth'],
  FILESIZE: ['range', 'count'],
  REQUESTTIME: ['range', 'count'],
};
const SIDER_ERR = ['url', 'hits', 'referrer']; // SIDER_404, SIDER_400, SIDER_403

const num = v => (v !== undefined && /^-?\d+(\.\d+)?$/.test(v) ? Number(v) : v);

function parse(text) {
  const out = { version: null, general: {}, sections: {} };
  const lines = text.split(/\r?\n/);
  const m = /^AWSTATS DATA FILE (\S+)/.exec(lines[0] || '');
  if (!m) throw new Error('not an AWStats text data file (XML data files are not supported)');
  out.version = m[1];
  let name = null, cols = null, rows = null;
  for (const line of lines) {
    if (!line || line[0] === '#') continue;
    if (name === null) {
      const b = /^BEGIN_(\S+)/.exec(line);
      if (!b) continue;
      name = b[1];
      cols = SCHEMA[name] !== undefined ? SCHEMA[name] : name.startsWith('SIDER_') ? SIDER_ERR : null;
      rows = [];
      continue;
    }
    if (line === `END_${name}`) {
      if (name === 'GENERAL') for (const [k, ...v] of rows) out.general[k] = v.length > 1 ? v.map(num) : num(v[0]);
      else if (name !== 'MAP') out.sections[name] = rows;
      name = null;
      continue;
    }
    const f = line.split(' ').filter(Boolean); // fields are space separated; values never contain spaces
    if (name === 'GENERAL' || name === 'MAP') { rows.push(f); continue; }
    if (!cols) { rows.push(f.map(num)); continue; }
    const r = {};
    cols.forEach((c, i) => { if (f[i] !== undefined) r[c] = i === 0 ? f[i] : num(f[i]); });
    rows.push(r);
  }
  return out;
}

// awstatsMMYYYY.config.txt  (or awstatsMMYYYY.txt for the default config)
const FILE_RE = /^awstats(\d{2})(\d{4})(?:\.(.+))?\.txt$/;

// Lists { site: [ 'YYYYMM', ... ] } found in a data directory.
function listSites(dir) {
  const sites = {};
  for (const f of fs.readdirSync(dir)) {
    const m = FILE_RE.exec(f);
    if (!m) continue;
    const site = m[3] || 'default';
    (sites[site] ||= []).push(m[2] + m[1]);
  }
  for (const s in sites) sites[s].sort();
  return sites;
}

const fileFor = (dir, site, ym) =>
  path.join(dir, `awstats${ym.slice(4)}${ym.slice(0, 4)}${site === 'default' ? '' : '.' + site}.txt`);

// Cache parsed files; re-read when mtime changes (awstats.pl -update rewrites them).
const cache = new Map();
function load(dir, site, ym) {
  const file = fileFor(dir, site, ym);
  const { mtimeMs } = fs.statSync(file);
  const hit = cache.get(file);
  if (hit && hit.mtimeMs === mtimeMs) return hit.data;
  const data = parse(fs.readFileSync(file, 'utf8'));
  cache.set(file, { mtimeMs, data });
  return data;
}

module.exports = { parse, listSites, load, fileFor };
