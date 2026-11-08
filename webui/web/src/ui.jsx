import { useEffect, useRef, useState } from 'react';
import * as echarts from 'echarts/core';
import { LineChart, BarChart, PieChart } from 'echarts/charts';
import { GridComponent, TooltipComponent, LegendComponent, DataZoomComponent, TitleComponent } from 'echarts/components';
import { CanvasRenderer } from 'echarts/renderers';

echarts.use([LineChart, BarChart, PieChart, TitleComponent, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent, CanvasRenderer]);

// ---- formatting --------------------------------------------------------------------------------
const nf = new Intl.NumberFormat('de-DE');
export const n = v => (v == null || v === '' ? '' : nf.format(v));
export const pct = (v, total) => (total ? `${(100 * v / total).toLocaleString('de-DE', { maximumFractionDigits: 1 })} %` : '');
export function bytes(v) {
  if (!v) return '0 B';
  const u = ['B', 'KB', 'MB', 'GB', 'TB']; let i = 0;
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++; }
  return `${v.toLocaleString('de-DE', { maximumFractionDigits: i ? 1 : 0 })} ${u[i]}`;
}
// AWStats dates are numbers like 20261031222426 or strings like 20261031
export function date(v, withTime = true) {
  const s = String(v || '');
  if (s.length < 8) return s;
  const d = `${s.slice(6, 8)}.${s.slice(4, 6)}.${s.slice(0, 4)}`;
  return withTime && s.length >= 12 ? `${d} ${s.slice(8, 10)}:${s.slice(10, 12)}` : d;
}
export const ym = s => `${['Jan', 'Feb', 'Mär', 'Apr', 'Mai', 'Jun', 'Jul', 'Aug', 'Sep', 'Okt', 'Nov', 'Dez'][+s.slice(4, 6) - 1]} ${s.slice(0, 4)}`;

// ---- theme-aware ECharts wrapper ------------------------------------------------------------------
export function css() {
  const s = getComputedStyle(document.documentElement);
  const v = k => s.getPropertyValue(k).trim();
  return { ink: v('--ink'), ink2: v('--ink-2'), muted: v('--muted'), grid: v('--grid'), axis: v('--axis'), surface: v('--surface'), other: v('--other'), series: ['--s1', '--s2', '--s3', '--s4', '--s5'].map(v) };
}
const esc = echarts.format.encodeHTML; // labels come from logs (URLs, referrers): never raw HTML

export function EChart({ option, className = 'chart', onClick, label }) {
  const el = useRef(null), chart = useRef(null);
  useEffect(() => {
    chart.current = echarts.init(el.current);
    const ro = new ResizeObserver(() => chart.current?.resize());
    ro.observe(el.current);
    return () => { ro.disconnect(); chart.current.dispose(); };
  }, []);
  useEffect(() => { chart.current.setOption(option, true); }, [option]);
  useEffect(() => {
    if (!onClick) return;
    chart.current.on('click', onClick);
    return () => chart.current.off('click', onClick);
  }, [onClick]);
  return <div ref={el} className={className} role="img" aria-label={label} />;
}

const tooltipBase = c => ({
  backgroundColor: c.surface, borderColor: c.axis, textStyle: { color: c.ink, fontFamily: 'system-ui' }, confine: true,
});
const axisBase = c => ({
  axisLine: { lineStyle: { color: c.axis } }, axisTick: { show: false },
  axisLabel: { color: c.muted }, splitLine: { lineStyle: { color: c.grid } },
});

// Line chart over time. series: [{ name, data: [[x, y], ...], color?, band?: [[x, lo, hi]] }]
export function timeOption(c, { x, series, fmt = n, zoom = false }) {
  const s = [];
  series.forEach((ser, i) => {
    const color = ser.color || c.series[i];
    if (ser.band) {
      // confidence band as stacked area: invisible lower edge + visible (hi - lo)
      s.push({ name: `${ser.name} (unten)`, type: 'line', data: ser.band.map(b => b[0]), stack: 'band' + i, symbol: 'none', lineStyle: { opacity: 0 }, tooltip: { show: false }, silent: true });
      s.push({ name: `${ser.name} Band`, type: 'line', data: ser.band.map(b => b[1] - b[0]), stack: 'band' + i, symbol: 'none', lineStyle: { opacity: 0 }, areaStyle: { color, opacity: 0.15 }, tooltip: { show: false }, silent: true });
    }
    s.push({ name: ser.name, type: 'line', data: ser.data, symbol: 'circle', symbolSize: 8, showSymbol: false, connectNulls: false, lineStyle: { width: 2, color, type: ser.dashed ? 'dashed' : 'solid' }, itemStyle: { color }, areaStyle: ser.area ? { color, opacity: 0.08 } : undefined, emphasis: { focus: 'none' } });
  });
  return {
    grid: { left: 8, right: 32, top: 30, bottom: zoom ? 56 : 8, containLabel: true },
    tooltip: {
      ...tooltipBase(c), trigger: 'axis', axisPointer: { type: 'line', lineStyle: { color: c.axis } },
      formatter: ps => `<div style="color:${c.muted}">${esc(ps[0].axisValueLabel)}</div>` + ps.filter(p => p.value != null && !p.seriesName.endsWith('Band') && !p.seriesName.endsWith('(unten)'))
        .map(p => `<div><span style="display:inline-block;width:12px;height:2px;background:${p.color};vertical-align:middle;margin-right:6px"></span><b>${esc(fmt(p.value))}</b> <span style="color:${c.ink2}">${esc(p.seriesName)}</span></div>`).join(''),
    },
    legend: { show: series.length > 1, top: 0, left: 0, textStyle: { color: c.ink2 }, icon: 'path://M0,0 L12,0 L12,2 L0,2 Z', data: series.map(s => s.name) },
    xAxis: { type: 'category', data: x, boundaryGap: false, ...axisBase(c), splitLine: { show: false } },
    yAxis: { type: 'value', ...axisBase(c), axisLabel: { color: c.muted, formatter: v => fmt(v) } },
    dataZoom: zoom ? [{ type: 'inside' }, { type: 'slider', height: 22, bottom: 8, borderColor: c.grid, textStyle: { color: c.muted } }] : [],
    series: s,
  };
}

// Single-series bar chart (one color: identity is the axis category, not the hue).
export function barOption(c, { x, y, name, fmt = n, horizontal = false, color }) {
  const cat = { type: 'category', data: x, ...axisBase(c), splitLine: { show: false }, axisLabel: { color: c.muted, width: horizontal ? 160 : undefined, overflow: 'truncate' } };
  const val = { type: 'value', ...axisBase(c), axisLabel: { color: c.muted, formatter: v => fmt(v) } };
  return {
    grid: { left: 8, right: 16, top: 12, bottom: 8, containLabel: true },
    tooltip: { ...tooltipBase(c), trigger: 'item', formatter: p => `<b>${esc(fmt(p.value))}</b> <span style="color:${c.ink2}">${esc(name)}</span><div style="color:${c.muted}">${esc(p.name)}</div>` },
    xAxis: horizontal ? val : cat, yAxis: horizontal ? { ...cat, inverse: true } : val,
    series: [{ type: 'bar', name, data: y, barMaxWidth: 28, itemStyle: { color: color || c.series[0], borderRadius: horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0] }, emphasis: { itemStyle: { opacity: 0.85 } } }],
  };
}

// Donut: top 5 + "Andere" (part-to-whole stays readable with <= 6 segments).
// total: sum over the whole section when `rows` is only its top part; the remainder lands in "Andere".
export function donutOption(c, rows, { label = r => r.label || r.id, value = r => r.hits, fmt = n, total: all } = {}) {
  const sorted = rows.filter(r => value(r) > 0).sort((a, b) => value(b) - value(a));
  const top = sorted.slice(0, 5).map((r, i) => ({ name: label(r), value: value(r), itemStyle: { color: c.series[i] } }));
  const rest = all != null ? all - top.reduce((s, d) => s + d.value, 0) : sorted.slice(5).reduce((s, r) => s + value(r), 0);
  if (rest) top.push({ name: 'Andere', value: rest, itemStyle: { color: c.other } });
  const total = top.reduce((s, d) => s + d.value, 0);
  const narrow = matchMedia('(max-width: 600px)').matches; // phone: legend below the ring instead of beside it
  return {
    tooltip: { ...tooltipBase(c), trigger: 'item', formatter: p => `<b>${esc(fmt(p.value))}</b> (${p.percent.toLocaleString('de-DE')} %)<div style="color:${c.ink2}">${esc(p.name)}</div>` },
    legend: { orient: narrow ? 'horizontal' : 'vertical', ...(narrow ? { left: 0, bottom: 0 } : { right: 0, top: 'middle' }), textStyle: { color: c.ink2, width: narrow ? 130 : 128, overflow: 'truncate' }, icon: 'roundRect', itemWidth: 10, itemHeight: 10,
      formatter: name => `${pct(top.find(d => d.name === name)?.value || 0, total)}  ${name}` }, // share first: a long name is truncated, the number never
    title: { text: fmt(total), subtext: 'gesamt', left: narrow ? '49%' : '27%', top: narrow ? '30%' : '42%', textAlign: 'center', textStyle: { color: c.ink, fontSize: 18, fontWeight: 600 }, subtextStyle: { color: c.muted } },
    series: [{ type: 'pie', radius: narrow ? ['38%', '58%'] : ['48%', '72%'], center: narrow ? ['50%', '37%'] : ['28%', '50%'], data: top, label: { show: false }, itemStyle: { borderColor: c.surface, borderWidth: 2, borderRadius: 4 }, emphasis: { scaleSize: 4 } }],
  };
}

// ---- table ------------------------------------------------------------------------------------
// cols: [{ key, label, num?, fmt?, wrap? }]
export function Table({ rows, cols, onRow, initialSort, keyFn = (r, i) => i, max }) {
  const [sort, setSort] = useState(initialSort || null);
  let list = rows || [];
  if (sort) {
    const { key, dir } = sort;
    list = [...list].sort((a, b) => {
      const x = a[key], y = b[key];
      return (typeof x === 'number' && typeof y === 'number' ? x - y : String(x ?? '').localeCompare(String(y ?? ''))) * dir;
    });
  }
  if (max) list = list.slice(0, max);
  if (!list.length) return <p className="muted">Keine Daten in diesem Zeitraum.</p>;
  return (
    <div className="tablewrap">
      <table>
        <thead><tr>{cols.map(c => (
          <th key={c.key} className={c.num ? 'num' : ''} aria-sort={sort?.key === c.key ? (sort.dir > 0 ? 'ascending' : 'descending') : 'none'}
            onClick={() => setSort(s => ({ key: c.key, dir: s?.key === c.key ? -s.dir : c.num ? -1 : 1 }))}>
            {c.label}{sort?.key === c.key ? (sort.dir > 0 ? ' ▲' : ' ▼') : ''}
          </th>))}</tr></thead>
        <tbody>{list.map((r, i) => (
          <tr key={keyFn(r, i)} className={onRow ? 'click' : ''} onClick={onRow ? () => onRow(r) : undefined}
            tabIndex={onRow ? 0 : undefined} onKeyDown={onRow ? e => e.key === 'Enter' && onRow(r) : undefined}>
            {cols.map(c => <td key={c.key} className={(c.num ? 'num ' : '') + (c.wrap ? 'wrap' : '')}>{c.render ? c.render(r) : c.fmt ? c.fmt(r[c.key], r) : r[c.key]}</td>)}
          </tr>))}</tbody>
      </table>
    </div>
  );
}

export function Card({ title, sub, children, wide }) {
  return <section className="card" style={wide ? { gridColumn: '1 / -1' } : undefined}><h2>{title}</h2>{sub && <p className="sub">{sub}</p>}{children}</section>;
}

export function useApi(url) {
  const [state, setState] = useState({ data: null, error: null, loading: true });
  useEffect(() => {
    if (!url) return;
    let live = true;
    setState(s => ({ ...s, loading: true }));
    fetch(url).then(async r => { const j = await r.json(); if (!r.ok) throw new Error(j.error || r.statusText); return j; })
      .then(data => live && setState({ data, error: null, loading: false }))
      .catch(error => live && setState(s => ({ ...s, error, loading: false })));
    return () => { live = false; };
  }, [url]);
  return state;
}
