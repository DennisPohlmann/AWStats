// Generates synthetic Apache/nginx "combined" logs for testing.
const fs = require('fs');
const [,, out, days = 120, seed = 1] = process.argv;
let s = +seed; const r = () => (s = (s * 1103515245 + 12345) % 2147483648) / 2147483648;
const pick = a => a[Math.floor(r() * a.length)];
const uas = [
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36',
  'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15',
  'Mozilla/5.0 (X11; Linux x86_64; rv:131.0) Gecko/20100101 Firefox/131.0',
  'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1',
  'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36',
  'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0',
  'Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)',
  'Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)',
];
const pages = ['/', '/index.html', '/produkte/', '/produkte/a.html', '/produkte/b.html', '/kontakt.html', '/blog/', '/blog/post-1.html', '/blog/post-2.html', '/impressum.html', '/download/katalog.pdf', '/suche?q=test'];
const assets = ['/css/style.css', '/js/app.js', '/img/logo.png', '/img/hero.jpg', '/favicon.ico'];
const refs = ['-', '-', '-', 'https://www.google.com/search?q=awstats+produkte', 'https://www.google.de/search?q=katalog+download', 'https://www.bing.com/search?q=blog+post', 'https://duckduckgo.com/', 'https://news.ycombinator.com/item?id=1', 'https://example.org/links.html'];
const ips = Array.from({ length: 400 }, () => `${pick([84, 91, 178, 46, 203, 66, 192, 2])}.${Math.floor(r()*255)}.${Math.floor(r()*255)}.${Math.floor(r()*254)+1}`);
const mon = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
const p2 = n => String(n).padStart(2, '0');
const start = Date.UTC(2026, 10, 7) - days * 86400000;
const lines = [];
for (let d = 0; d < days; d++) {
  const dow = new Date(start + d * 86400000).getUTCDay();
  const visits = Math.round((60 + d * 0.6) * (dow === 0 || dow === 6 ? 0.6 : 1) * (0.8 + r() * 0.4));
  for (let v = 0; v < visits; v++) {
    const ip = pick(ips), ua = pick(uas);
    let t = start + d * 86400000 + Math.floor((r() < 0.7 ? 8 + r() * 12 : r() * 24) * 3600000);
    const n = 1 + Math.floor(r() * 5);
    for (let k = 0; k < n; k++) {
      const page = pick(pages);
      const reqs = [page, ...(page.endsWith('.pdf') ? [] : assets.filter(() => r() < 0.5))];
      for (const u of reqs) {
        const code = r() < 0.03 ? 404 : r() < 0.01 ? 500 : u === page && r() < 0.05 ? 304 : 200;
        const dt = new Date(t);
        const ts = `${p2(dt.getUTCDate())}/${mon[dt.getUTCMonth()]}/${dt.getUTCFullYear()}:${p2(dt.getUTCHours())}:${p2(dt.getUTCMinutes())}:${p2(dt.getUTCSeconds())} +0000`;
        lines.push([t, `${ip} - - [${ts}] "GET ${code === 404 ? u.replace('.html', '-alt.html') : u} HTTP/1.1" ${code} ${code === 304 ? 0 : Math.floor(500 + r() * 80000)} "${k ? 'https://www.example.com/' : pick(refs)}" "${ua}"`]);
        t += 200 + Math.floor(r() * 2000);
      }
      t += Math.floor(r() * 120000);
    }
  }
}
lines.sort((a, b) => a[0] - b[0]);
fs.writeFileSync(out, lines.map(l => l[1]).join('\n') + '\n');
console.log(out, lines.length, 'lines');
