// Exports the label tables of the AWStats detection tables (cmd/awstats/lib.json) for the web interface.
// Usage (in webui/): node tools/export-labels.js > server/labels.json
const fs = require('fs');
const path = require('path');
const lib = JSON.parse(fs.readFileSync(path.join(__dirname, '../../cmd/awstats/lib.json'), 'utf8'));
const tables = {
  browsers: '%BrowsersHashIDLib', domains: '%DomainsHashIDLib', http: '%httpcodelib', os: '%OSHashLib',
  robots: '%RobotsHashIDLib', searchEngines: '%SearchEnginesHashLib', worms: '%WormsHashLib',
};
// Written by hand with sorted keys: a JS object would put keys like "300" before "2xx".
const obj = (h, val) => '{' + Object.keys(h).sort().map((k) => JSON.stringify(k) + ':' + val(h[k])).join(',') + '}';
// Strips tags until nothing changes, so nested tags like "<<b>script>" cannot leave a tag behind.
const stripTags = (s) => {
  let prev;
  do {
    prev = s;
    s = s.replace(/<[^>]*>/g, '');
  } while (s !== prev);
  return s;
};
const clean = (s) => JSON.stringify(stripTags(s).replace(/&nbsp;/g, ' '));
process.stdout.write(obj(tables, (t) => obj(lib[t], clean)));
