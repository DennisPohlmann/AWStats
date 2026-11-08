# AWStats web interface

A modern web interface for AWStats. Log analysis stays with `awstats.pl`. The
interface reads its data files (`awstatsMMYYYY.<site>.txt`) directly, so
existing history works without any migration and no extra database is needed.

```
web server logs ──► awstats.pl -update (cron, as before) ──► data files ──► Node.js server (API + web UI)
                                                                  ▲
                    raw logs (also .gz) ── only for the IP detail search ┘
```

## Features

- **Overview:** key figures, then visits, pages, hits and bandwidth per day (zoomable), per month, per weekday and per hour, plus visit duration.
- **Visitors & IPs:** search across all hosts and IPs of the period. A click shows the monthly values from AWStats and every single request from the raw logs, rotated `.gz` files included.
- **Pages:** top pages with entries and exits, file types, downloads.
- **Origin:** origin types, search engines, referring pages, search phrases and keywords.
- **Technology:** browsers (by family, table with versions), operating systems, countries/domains, screen sizes.
- **Robots** and **Errors:** HTTP codes and the URLs that returned 404, 400 and 403.
- **Forecast:** Holt-Winters with trend and weekly pattern, 14, 30 or 90 days ahead, with a 95 % band.
- Periods: current month, last 3 or 12 months, all time, single years and months. Light and dark mode, mobile layout.
- Several sites: every config with data files shows up in the site selector.

Labels (browsers, operating systems, robots, search engines, countries, HTTP
codes) come from the AWStats tables in `wwwroot/cgi-bin/lib/*.pm`, exported to
`server/labels.json` with `tools/export-labels.pl`.

## Requirements

- **Node.js 20 or newer.** The server itself has no npm dependencies; npm is only needed to build the interface.
- **AWStats** to create the data files, as before. `awstats.pl` 8.x needs the Perl modules `JSON::XS` and `Try::Tiny` (Debian/Ubuntu: `apt install libjson-xs-perl libtry-tiny-perl`).

## Running it in a container

The image contains only the web interface. Mount your existing AWStats data
directory (the `DirData` of your configs) read-only:

```sh
docker run -d --name awstats -p 127.0.0.1:3000:3000 \
  -v /var/lib/awstats:/var/lib/awstats:ro \
  ghcr.io/dennispohlmann/awstats:latest
# -> http://127.0.0.1:3000
```

or, from the repository root, `docker compose up -d` (see
`docker-compose.yml`; `AWSTATS_DATA=/path/to/data` picks another directory).
For the IP search, also mount the configs at `/etc/awstats` and the logs at
the path their `LogFile` names. The container listens on all interfaces
inside the container; the port mapping above keeps it on the host's
loopback, ready for a reverse proxy. `GET /healthz` answers `ok` while the
data directory is readable.

Images: `edge` follows `main`, releases are tagged `<version>`,
`<major>.<minor>` and `latest`.

## Running it without a container

```sh
cd webui
npm ci
npm run build            # builds the interface into web/dist
AWSTATS_DATA_DIR=/var/lib/awstats AWSTATS_CONF_DIR=/etc/awstats npm start
# -> http://127.0.0.1:3000
```

| Variable | Default | Meaning |
|---|---|---|
| `AWSTATS_DATA_DIR` | `testdata/data` | Directory with the data files (`DirData` of the AWStats config). It is only read, never written. |
| `AWSTATS_CONF_DIR` | `/etc/awstats` | Directory with `awstats.<site>.conf`; `LogFile` is read from there for the IP search. |
| `AWSTATS_LOG_FILES` | – | Optional: comma-separated log files for the IP search, when `LogFile` is a pipe or uses `%YYYY` tags. |
| `PORT` / `HOST` | `3000` / `127.0.0.1` | Listen address. |
| `AWSTATS_ALLOWED_HOSTS` | – | Comma-separated host names the API answers to, besides `localhost` and IP addresses, e.g. `stats.example.com`. Other `Host` headers get 403, which blocks DNS rebinding. |

### Behind a reverse proxy

The interface shows IP addresses and single requests, which is personal data.
The server therefore listens on `127.0.0.1` by default. Publish it only through
a reverse proxy with authentication, on its own host name (the interface loads
`/api/...` from the root path).

nginx:

```nginx
server {
    server_name stats.example.com;
    location / {
        auth_basic "Statistics";
        auth_basic_user_file /etc/nginx/htpasswd-stats;
        proxy_pass http://127.0.0.1:3000;
    }
}
```

nginx sends `127.0.0.1:3000` as `Host` here, so nothing else is needed.

HAProxy passes the client's `Host` header on, so set
`AWSTATS_ALLOWED_HOSTS=stats.example.com` (in the container with
`-e AWSTATS_ALLOWED_HOSTS=stats.example.com`):

```haproxy
userlist stats_users
    user admin password <crypt(3) hash, e.g. from: mkpasswd -m sha-512>

frontend https
    bind :443 ssl crt /etc/haproxy/certs/
    acl is_stats hdr(host) -i stats.example.com
    http-request auth realm Statistics if is_stats !{ http_auth(stats_users) }
    use_backend stats if is_stats

backend stats
    server webui 127.0.0.1:3000 check
```

A systemd unit for running it without a container:

```ini
[Service]
ExecStart=/usr/bin/node /opt/awstats-webui/server/index.js
Environment=AWSTATS_DATA_DIR=/var/lib/awstats AWSTATS_CONF_DIR=/etc/awstats
User=www-data
Restart=on-failure
```

The user needs read access to the data files and, for the IP search, to the
logs (often `root:adm`). Unreadable log files are skipped and named in the
interface.

## Development and tests

```sh
cd webui
sh testdata/build.sh ../wwwroot/cgi-bin   # test logs (Apache + nginx) and data files made with awstats.pl
npm test                                  # reader, API, IP search and forecast against those files
npm start & npm run dev                   # Vite dev server with a proxy to the API
```

## Known limits

- Only text data files are read; XML data files (`BuildHistoryFormat=xml`) are not supported.
- AWStats keeps unique visitors per month only. For periods longer than a month the interface shows the number of distinct hosts/IPs instead.
- AWStats does not store visits per hour, so the hour chart shows pages, hits or bandwidth.
- The IP detail search reads the raw logs on every request and shows the newest 2,000 matches, so it takes a while on very large logs.
- Structured fields in the IP search are only parsed for the "combined" format (Apache/nginx); other formats are shown as raw lines.
