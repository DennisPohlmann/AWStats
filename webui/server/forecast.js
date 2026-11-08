// Additive Holt-Winters forecast with weekly seasonality for daily series.
// Picks smoothing parameters by grid search on one-step-ahead error.

function fit(y, a, b, g, m) {
  let level = y.slice(0, m).reduce((s, v) => s + v, 0) / m;
  let trend = m * 2 <= y.length ? (y.slice(m, 2 * m).reduce((s, v) => s + v, 0) / m - level) / m : 0;
  const season = y.slice(0, m).map(v => v - level);
  let sse = 0, n = 0;
  for (let t = m; t < y.length; t++) {
    const pred = level + trend + season[t % m];
    const err = y[t] - pred;
    sse += err * err; n++;
    const prev = level;
    level = a * (y[t] - season[t % m]) + (1 - a) * (level + trend);
    trend = b * (level - prev) + (1 - b) * trend;
    season[t % m] = g * (y[t] - level) + (1 - g) * season[t % m];
  }
  return { level, trend, season, rmse: Math.sqrt(sse / Math.max(n, 1)) };
}

// y: numbers (one per day, no gaps). Returns { mean[], lower[], upper[], params } for `h` days ahead.
function forecast(y, h, m = 7) {
  if (y.length < 2 * m) return null; // not enough history for a weekly pattern
  const grid = [0.1, 0.3, 0.5, 0.7, 0.9];
  let best = null;
  for (const a of grid) for (const b of [0.01, 0.05, 0.1, 0.2]) for (const g of grid) {
    const f = fit(y, a, b, g, m);
    if (!best || f.rmse < best.f.rmse) best = { f, a, b, g };
  }
  const { level, trend, season, rmse } = best.f;
  const mean = [], lower = [], upper = [];
  for (let k = 1; k <= h; k++) {
    const v = level + k * trend + season[(y.length + k - 1) % m];
    const band = 1.96 * rmse * Math.sqrt(1 + (k - 1) * best.a * best.a); // widens with horizon
    mean.push(Math.max(0, v)); lower.push(Math.max(0, v - band)); upper.push(Math.max(0, v + band));
  }
  return { mean, lower, upper, params: { alpha: best.a, beta: best.b, gamma: best.g, rmse } };
}

module.exports = { forecast };
