# AWStats - Advanced Web Statistics

AWStats is a log file analyzer for web servers. `awstats.pl` reads the access
logs of Apache, nginx, IIS and most other web, proxy, streaming, FTP and mail
servers and stores the results in monthly data files. A web interface in
[`webui/`](webui/README.md) shows those data files with charts, IP search and
forecasts, for several sites and on the phone.

## Features

### Log analysis

`awstats.pl` analyzes the logs and shows:
- Number of VISITS and UNIQUE VISITORS
- Visits duration and last visits
- Authenticated users, and last authenticated visits
- Days of week and rush hours (pages, hits, KB for each day and hour)
- Domains/countries of hosts visitors (pages, hits, KB)
- Hosts list, last visits and unresolved IP addresses list
- Most viewed, entry and exit pages
- File types
- Web compression statistics (for mod_gzip or mod_deflate)
- Browsers used (pages, hits, kb for each browser)
- OS used (pages, hits, KB for each OS)
- Robot visits
- Worm attacks
- Download and continuation detection
- Search engines, keyphrases and keywords used to find your site
- HTTP errors (Page not found with last referer, etc,)
- Screen size report
- Number of times your site is "added to favourites bookmarks"
- Ratio of Browsers that support: Java, Flash, RealG2 reader, Quicktime reader, WMA reader, PDF reader
- Cluster report for load balanced servers ratio
- Other personalized reports...

It also supports:
- Can analyze all log formats
- Works from command line and from a browser as a CGI (with dynamic filters capabilities for some charts)
- Update of statistics can be made on demand from the web interface and not only from your scheduler
- Unlimited log file size, support split log files (load balancing system)
- Support 'nearly sorted' log files even for entry and exit pages
- Reverse DNS lookup before or during analysis, supports DNS cache files
- Country detection from IP location or domain name
- WhoIS links
- A lot of options/filters and plugins can be used
- Multi-named web sites supported (virtual servers)
- Cross Site Scripting Attacks protection
- Several languages
- No need of rare perl libraries
- Dynamic reports as CGI output
- Static reports in one or framed HTML or XHTML pages
- Experimental PDF export
- Look and colors can match your site design (CSS)
- Help and tooltips on HTML reported pages
- Easy to use (Just one configuration file to edit)
- Analysis database can be stored in XML format (for XSLT processing, ...)
- Free (GNU GPL) with sources (perl scripts)
- Available on all platforms

### Web interface

- Reads the data files of `awstats.pl` directly: all existing history works, no extra database.
- Overview, visitors and IPs, pages, origin, technology, robots, errors and a forecast.
- IP search across the raw logs, rotated `.gz` files included.
- Several sites, light and dark mode, phone layout.
- Runs as a container (`ghcr.io/dennispohlmann/awstats`) or with Node.js behind a reverse proxy.

Details: [webui/README.md](webui/README.md).

## Requirements

- **awstats.pl** (still included): Perl with the modules `Encode`, `JSON::XS`
  and `Try::Tiny`. Not needed for the updater below.
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
[`wwwroot/cgi-bin/awstats.model.conf`](wwwroot/cgi-bin/awstats.model.conf), for example
`/etc/awstats/awstats.example.com.conf`, and set `LogFile`, `LogFormat`,
`SiteDomain` and `DirData`. The updater supports `LogFormat` 1, 4 and
personalized formats, not 2, 3 and 5, and no mail or ftp logs (`LogType` M
and F), plugins or `ExtraSection`s.

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

- [docs/index.html](docs/index.html): setup, configuration, plugins, FAQ for `awstats.pl`
- [webui/README.md](webui/README.md): the web interface, its settings and the container
- [CHANGELOG.md](CHANGELOG.md): changes per release
- [CONTRIBUTING.md](CONTRIBUTING.md): development, commits and releases
- [SECURITY.md](SECURITY.md): reporting a vulnerability

## Files

	README.md                           This file
	LICENSE                             GNU General Public Licence
	CHANGELOG.md                        Changes per release
	docs/*                              AWStats documentation (setup/usage...)
	wwwroot/cgi-bin/awstats.pl          THE MAIN AWSTATS PROGRAM (CLI/CGI)
	wwwroot/cgi-bin/awredir.pl          A tool to track exit clicks
	wwwroot/cgi-bin/awstats.model.conf  An model configuration file
	wwwroot/cgi-bin/lang                Directory with languages files
	wwwroot/cgi-bin/lib                 Directory with awstats reference info
	wwwroot/cgi-bin/plugins             Directory with optional plugins
	wwwroot/icon/browser                Directory with browsers icons
	wwwroot/icon/clock                  Directory with clock icons
	wwwroot/icon/cpu                    Directory with cpu icons
	wwwroot/icon/flags                  Directory with country flag icons
	wwwroot/icon/os                     Directory with OS icons
	wwwroot/icon/other                  Directory with all others icons
	wwwroot/classes                     Java applet for graphapplet plugin
	wwwroot/css                         Samples of CSS files
	wwwroot/js                          Javascript sources for "Misc" feature
	tools/*                             Other provided tools
	tools/xslt/awstats.xsd              AWStats XML database schema descriptor
	tools/xslt/*                        Demo to manipulate AWStats XML database
	webui/                              Web interface for the data files (Node.js)

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
