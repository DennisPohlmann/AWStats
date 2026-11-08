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

- **Log analysis:** Perl with the modules `Encode`, `JSON::XS` and `Try::Tiny`
  (Debian/Ubuntu: `apt install libjson-xs-perl libtry-tiny-perl`), and read
  access to the web server logs. For real-time statistics as a CGI, the web
  server must be able to run Perl scripts.
- **Go updater:** Go 1.24 or newer to build it (`cmd/awstats`).
- **Web interface:** Docker, or Node.js 20 or newer.

## Quick start

1. Create a config for your site from `wwwroot/cgi-bin/awstats.model.conf`
   (or with `tools/awstats_configure.pl`), for example
   `/etc/awstats/awstats.example.com.conf`, and set `LogFile`, `LogFormat`,
   `SiteDomain` and `DirData`.
2. Update the statistics, for example hourly from cron:

   ```sh
   perl wwwroot/cgi-bin/awstats.pl -config=example.com -update
   ```

   Or with the Go updater (same configs, same data files, no Perl):

   ```sh
   go build -o awstats ./cmd/awstats
   ./awstats -config=example.com -update
   ```

   Not supported yet by the Go updater: `LogFormat` 2, 3 and 5, mail and ftp
   logs (`LogType` M and F), plugins and `ExtraSection`s.

3. Start the web interface on the data directory (`DirData`):

   ```sh
   docker run -d --name awstats -p 127.0.0.1:3000:3000 \
     -v /var/lib/awstats:/var/lib/awstats:ro \
     ghcr.io/dennispohlmann/awstats:latest
   ```

   It listens on `http://127.0.0.1:3000`. Publish it only through a reverse
   proxy with a login, because it shows IP addresses; see
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
