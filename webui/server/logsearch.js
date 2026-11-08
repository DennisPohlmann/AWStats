// Searches raw web server logs for one IP/host, so the UI can show every request of a visitor.
// AWStats data files only keep monthly totals per host, so details must come from the logs.
const fs = require('fs');
const path = require('path');
const zlib = require('zlib');
const readline = require('readline');

// Reads LogFile from awstats.<site>.conf. Returns null when it is a pipe or uses date tags.
function logFileFromConf(confDir, site) {
  const conf = path.join(confDir, `awstats.${site}.conf`);
  if (!fs.existsSync(conf)) return null;
  // AWStats uses the last definition; shortcut: Include directives are not followed.
  const m = [...fs.readFileSync(conf, 'utf8').matchAll(/^LogFile\s*=\s*"?([^"\r\n]*)"?/gm)].pop();
  if (!m) return null;
  const file = m[1].trim();
  // shortcut: piped or %YYYY-style LogFile values are not resolved, set AWSTATS_LOG_FILES instead.
  if (!file || file.includes('|') || file.includes('%')) return null;
  return file;
}

// The log plus its rotated siblings: access.log, access.log.1, access.log.2.gz, access.log-20261101 ...
function logFiles(file) {
  const dir = path.dirname(file), base = path.basename(file);
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir)
    .filter(f => f === base || (f.startsWith(base + '.') || f.startsWith(base + '-')))
    .map(f => path.join(dir, f))
    .filter(f => { try { fs.accessSync(f, fs.constants.R_OK); return fs.statSync(f).isFile(); } catch { return false; } })
    .sort((a, b) => fs.statSync(a).mtimeMs - fs.statSync(b).mtimeMs); // oldest rotation first
}

// Apache/nginx "combined" (AWStats LogFormat=1). Other formats are returned as raw lines.
const COMBINED = /^(\S+) \S+ (\S+) \[([^\]]+)\] "(\S+) ([^\s"]+)(?: [^"]*)?" (\d{3}) (\S+)(?: "([^"]*)" "([^"]*)")?/;

async function searchIp(files, ip, limit = 2000) {
  const esc = ip.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const re = new RegExp(`(^|[^\\w.:])${esc}($|[^\\w.:])`);
  const hits = [];
  let total = 0;
  const skipped = [];
  for (const file of files) {
    try {
      const src = fs.createReadStream(file);
      let stream = src;
      if (file.endsWith('.gz')) { stream = src.pipe(zlib.createGunzip()); src.on('error', e => stream.destroy(e)); }
      const rl = readline.createInterface({ input: stream, crlfDelay: Infinity });
      for await (const line of rl) {
        if (!line.includes(ip) || !re.test(line)) continue;
        total++;
        const m = COMBINED.exec(line);
        hits.push(m
          ? { file: path.basename(file), time: m[3], method: m[4], url: m[5], status: +m[6], bytes: m[7] === '-' ? 0 : +m[7], referer: m[8], agent: m[9], user: m[2] }
          : { file: path.basename(file), raw: line });
        if (hits.length > 2 * limit) hits.splice(0, hits.length - limit); // keep the newest, amortized O(1)
      }
    } catch (e) {
      skipped.push(`${path.basename(file)} (${e.code || e.message})`); // unreadable or corrupt file: skip, keep searching
    }
  }
  if (hits.length > limit) hits.splice(0, hits.length - limit);
  return { hits, total, truncated: total > hits.length, skipped };
}

module.exports = { logFileFromConf, logFiles, searchIp };
