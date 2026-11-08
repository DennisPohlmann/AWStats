# AWStats - Advanced Web Statistics

AWStats is a log file analyzer for web servers. The updater `awstats`
(written in Go, in [`cmd/awstats`](cmd/awstats)) reads the access logs of
Apache, nginx, IIS and other web servers and stores the results in monthly
data files, in the same format as the original `awstats.pl`. A web interface
in [`webui/`](webui/README.md) shows those data files with charts, IP search
and forecasts, for several sites and on the phone.

## Features

### Log analysis

- Visits, unique visitors, visit duration and last visits
- Authenticated users
- Days of week and hours (pages, hits, bytes)
- Domains/countries of hosts, hosts list
- Most viewed, entry and exit pages, file types, downloads
- Browsers, operating systems, screen sizes
- Robots and worms
- Search engines, keyphrases and keywords, referring sites
- HTTP errors with the last referrer
- Cluster report for load balanced servers
- Incremental updates: each run continues after the last line of the last run
- Configs of `awstats.pl` keep working (`awstats.<site>.conf`, `Include`, `__ENV__`)
- Logs as plain files, `.gz` files or the output of a command (`LogFile="cmd |"`)
- Reverse DNS lookup (`DNSLookup=1`)

Not supported (any more): `LogFormat` 2, 3 and 5, mail and ftp logs
(`LogType` M and F), plugins, `ExtraSection`s, and the HTML/CGI reports of
`awstats.pl` (the web interface replaces them).

### Web interface

- Reads the data files directly (also those of `awstats.pl`): all existing history works, no extra database.
- Overview, visitors and IPs, pages, origin, technology, robots, errors and a forecast.
- IP search across the raw logs, rotated `.gz` files included.
- Several sites, light and dark mode, phone layout.
- Runs as a container (`ghcr.io/dennispohlmann/awstats`) or with Node.js behind a reverse proxy.

Details: [webui/README.md](webui/README.md).

## Requirements

- **Updater:** the `awstats` binary from the release (Linux amd64/arm64), or
  Go 1.24 or newer to build it, or the container image `awstats-updater`.
  It needs read access to the web server logs.
- **Web interface:** the container image `awstats`, or Node.js 20 or newer.

## Quick start

AWStats has two parts that share one data directory (`DirData`):

- the **updater** `awstats` reads the logs and writes the data files, and
- the **web interface** reads the data files and shows them.

Each part runs either as a container or directly on the host, in any
combination. First create a config for each site from
[`conf/awstats.model.conf`](conf/awstats.model.conf), for example
`/etc/awstats/awstats.example.com.conf`, and set `LogFile`, `LogFormat`,
`SiteDomain` and `DirData` (limits of the updater: see above).

### Both in containers

```sh
AWSTATS_LOGS=/var/log/nginx docker compose up -d     # -> http://127.0.0.1:3000
```

[`docker-compose.yml`](docker-compose.yml) starts the web interface and the
updater, which updates every config in `/etc/awstats` once an hour. In the
configs, `DirData` must be `/var/lib/awstats` and `LogFile` a path under
`AWSTATS_LOGS`; the data directory must be writable for uid 1000.
`docker compose run --rm updater -config=example.com` updates one site now.

### Both without containers

The updater from a systemd timer (or cron), the web interface with Node.js:

```sh
tar xzf awstats-updater-<version>-linux-amd64.tar.gz
sudo install awstats-updater-<version>-linux-amd64/awstats /usr/local/bin/
sudo cp awstats-updater-<version>-linux-amd64/awstats-update.* /etc/systemd/system/
sudo systemctl daemon-reload && sudo systemctl enable --now awstats-update.timer
# or in crontab: 0 * * * * /usr/local/bin/awstats -all
```

`awstats -config=example.com` updates one site, `awstats -all` every config
in `/etc/awstats` (or `-configdir`). Building it yourself: `go build -o awstats
./cmd/awstats`. The web interface: unpack `awstats-web-<version>.tar.gz` to
`/opt/awstats-webui` and start `node server/index.js` (systemd unit and
settings in [webui/README.md](webui/README.md)).

### Mixed

- Updater on the host, web interface in a container: install the updater as
  above, then `docker compose up -d web`.
- Updater in a container, web interface on the host: `docker compose up -d
  updater`, and the web interface as above.

The web interface listens on `http://127.0.0.1:3000`. Publish it only through
a reverse proxy with a login, because it shows IP addresses; see
[webui/README.md](webui/README.md) for nginx and HAProxy.

## Documentation

- [conf/awstats.model.conf](conf/awstats.model.conf): all config options, with comments
- [webui/README.md](webui/README.md): the web interface, its settings and the container
- [CHANGELOG.md](CHANGELOG.md): changes per release
- [CONTRIBUTING.md](CONTRIBUTING.md): development, commits and releases
- [SECURITY.md](SECURITY.md): reporting a vulnerability

## Files

	README.md                 This file
	LICENSE                   GNU General Public Licence
	CHANGELOG.md              Changes per release
	cmd/awstats               The updater (Go): logs -> data files
	cmd/awstats/lib.json      Detection tables (browsers, OS, robots, search engines...)
	conf/awstats.model.conf   A model configuration file
	webui                     The web interface (Node.js)
	docs/awstats_changelog.txt Changes up to version 8.0 (awstats.pl)
	contrib/systemd           systemd service and timer for the updater
	Dockerfile                Container images: web interface and updater
	docker-compose.yml        Both containers with their mounts

## Credits

AWStats was created by Laurent Destailleur.

## License

Copyright (C) 2000-2025 - Laurent Destailleur - eldy@users.sourceforge.net - <https://www.dolicloud.com>

This program is free software; you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation; either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

Third-party components of the web interface: [THIRD-PARTY-LICENSES.md](THIRD-PARTY-LICENSES.md).
