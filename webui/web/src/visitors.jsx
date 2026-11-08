import { useEffect, useState } from 'react';
import { EChart, Card, Table, useApi, barOption, n, bytes, date, ym } from './ui.jsx';

const PAGE = 50;

export function Visitors({ site, from, to, c }) {
  const [q, setQ] = useState('');
  const [debounced, setDebounced] = useState('');
  const [offset, setOffset] = useState(0);
  const [ip, setIp] = useState(null);
  useEffect(() => { const t = setTimeout(() => { setDebounced(q.trim()); setOffset(0); }, 250); return () => clearTimeout(t); }, [q]);
  useEffect(() => setOffset(0), [site, from, to]);
  const fullHost = /^(\d{1,3}\.){3}\d{1,3}$|^[0-9a-f]*:[0-9a-f:]+$|^[a-z0-9-]+(\.[a-z0-9-]+)*\.[a-z]{2,}$/i.test(q.trim()); // log search matches whole IPs/hosts only
  const base = `/api/${encodeURIComponent(site)}`;
  const list = useApi(`${base}/list/VISITOR?from=${from}&to=${to}&q=${encodeURIComponent(debounced)}&offset=${offset}&limit=${PAGE}`);
  const total = list.data?.total || 0;

  return (
    <div className="grid">
      {ip && <IpDetail key={ip} base={base} ip={ip} c={c} onClose={() => setIp(null)} />}
      <Card wide title="Besucher (Hosts / IP-Adressen)" sub="Werte aus den AWStats-Datendateien für den gewählten Zeitraum. Klick auf eine Zeile zeigt alle Requests aus den Logs.">
        <div className="tools">
          <input type="search" placeholder="IP oder Host suchen, z. B. 84.63." value={q} onChange={e => setQ(e.target.value)} aria-label="IP oder Host suchen" />
          <button className="btn" onClick={() => setIp(q.trim())} disabled={!fullHost} title="Vollständige IP oder Hostname eingeben">Vollständige IP in Logs suchen</button>
          <span className="muted">{n(total)} Treffer</span>
        </div>
        {list.error && <p className="error">{list.error.message}</p>}
        <Table rows={list.data?.rows} keyFn={r => r.host} onRow={r => { setIp(r.host); window.scrollTo({ top: 0, behavior: 'smooth' }); }} cols={[
          { key: 'host', label: 'Host / IP' }, { key: 'pages', label: 'Seiten', num: true, fmt: n }, { key: 'hits', label: 'Hits', num: true, fmt: n },
          { key: 'bandwidth', label: 'Bandbreite', num: true, fmt: bytes }, { key: 'lastVisit', label: 'Letzter Besuch', fmt: v => date(v) }]} />
        {total > PAGE && (
          <div className="pager">
            <button className="btn" disabled={!offset} onClick={() => setOffset(o => Math.max(0, o - PAGE))}>Zurück</button>
            <span className="muted">{offset + 1} bis {Math.min(total, offset + PAGE)} von {n(total)}</span>
            <button className="btn" disabled={offset + PAGE >= total} onClick={() => setOffset(o => o + PAGE)}>Weiter</button>
          </div>
        )}
      </Card>
    </div>
  );
}

function IpDetail({ base, ip, c, onClose }) {
  const d = useApi(`${base}/ip/${encodeURIComponent(ip)}`);
  const [filter, setFilter] = useState('');
  const [max, setMax] = useState(100);
  const hits = d.data?.log.hits || [];
  const f = filter.toLowerCase();
  const newestFirst = [...hits].reverse();
  const shown = f ? newestFirst.filter(h => Object.values(h).some(v => String(v).toLowerCase().includes(f))) : newestFirst;
  const perDay = Object.entries(hits.reduce((a, h) => { const k = h.time?.slice(0, 11); if (k) a[k] = (a[k] || 0) + 1; return a; }, {}));
  const status = Object.entries(hits.reduce((a, h) => { if (h.status) a[h.status] = (a[h.status] || 0) + 1; return a; }, {}));

  return (
    <Card title={`Details: ${ip}`} sub={d.loading ? 'Durchsuche Logdateien …' : d.data?.log.files.length ? `Logdateien: ${d.data.log.files.join(', ')}` : ''} wide>
      <div className="tools"><button className="btn" onClick={onClose}>Schließen</button></div>
      {d.error && <p className="error">{d.error.message}</p>}
      {d.data && (
        <>
          <h2>Monatswerte (AWStats)</h2>
          <Table rows={d.data.months} initialSort={{ key: 'month', dir: -1 }} cols={[
            { key: 'month', label: 'Monat', fmt: ym }, { key: 'pages', label: 'Seiten', num: true, fmt: n }, { key: 'hits', label: 'Hits', num: true, fmt: n },
            { key: 'bandwidth', label: 'Bandbreite', num: true, fmt: bytes }, { key: 'lastVisit', label: 'Letzter Besuch', fmt: v => date(v) }]} />
          <h2 style={{ marginTop: 16 }}>Requests aus den Logs</h2>
          {d.data.log.note && <p className="note">{d.data.log.note}</p>}
          {d.data.log.skipped?.length > 0 && <p className="note">Nicht lesbar, übersprungen: {d.data.log.skipped.join(', ')}</p>}
          {d.data.log.truncated && <p className="note">{n(d.data.log.total)} Treffer in den Logs, angezeigt werden die neuesten {n(hits.length)}.</p>}
          {hits.length > 0 && (
            <div className="grid">
              <div><p className="sub">Requests pro Tag</p><EChart label="Requests pro Tag" option={barOption(c, { x: perDay.map(p => p[0]), y: perDay.map(p => p[1]), name: 'Requests' })} /></div>
              <div><p className="sub">Statuscodes</p><EChart label="Statuscodes" option={barOption(c, { x: status.map(s => s[0]), y: status.map(s => s[1]), name: 'Requests' })} /></div>
            </div>
          )}
          {hits.length > 0 && <div className="tools"><input type="search" placeholder="Requests filtern (URL, Status, Agent …)" value={filter} onChange={e => setFilter(e.target.value)} aria-label="Requests filtern" /><span className="muted">{n(shown.length)} von {n(hits.length)}</span></div>}
          <Table rows={shown} max={max} cols={hits[0]?.raw !== undefined ? [{ key: 'raw', label: 'Logzeile', wrap: true }] : [
            { key: 'time', label: 'Zeit' }, { key: 'method', label: 'Methode' }, { key: 'url', label: 'URL', wrap: true },
            { key: 'status', label: 'Status', render: r => <span className={`status s${String(r.status)[0]}`}>{r.status}</span> }, { key: 'bytes', label: 'Bytes', num: true, fmt: bytes },
            { key: 'referer', label: 'Referrer', wrap: true }, { key: 'agent', label: 'User-Agent', wrap: true }]} />
          {shown.length > max && <div className="pager"><span className="muted">{n(max)} von {n(shown.length)}</span><button className="btn" onClick={() => setMax(m => m + 100)}>Weitere 100 anzeigen</button></div>}
        </>
      )}
    </Card>
  );
}
