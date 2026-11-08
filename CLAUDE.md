# AWStats fork - notes for Claude

Fork of eldy/awstats. Two parts:

- `wwwroot/cgi-bin/awstats.pl` (Perl, upstream): parses web server logs and
  writes the monthly data files `awstatsMMYYYY[.config].txt` into `DirData`.
- `cmd/awstats/` (Go): the replacement for the update part of `awstats.pl`.
  Reads the same configs and logs and writes the same data files; the tests
  compare its output with data files written by `awstats.pl`. Detection
  tables are `cmd/awstats/lib.json` (a dump of `wwwroot/cgi-bin/lib/*.pm`).
- `webui/` (Node.js): reads those data files and serves a JSON API and a
  React/ECharts interface. `webui/server/` has no npm dependencies; the
  interface is built with Vite into `webui/web/dist`. See `webui/README.md`.

Build and check: `cd webui && npm ci && sh testdata/build.sh ../wwwroot/cgi-bin
&& npm test && npm run build`, then in the root `go vet ./... && go test ./...`
(the Go tests use the data files `build.sh` wrote with `awstats.pl`). Container: `Dockerfile` in the root (image
`ghcr.io/dennispohlmann/awstats`), `docker-compose.yml` for the mounts.

## Rules

1. **Data files stay compatible.** The data directory and the format of the
   data files must not change in a way that breaks existing files: old files
   must stay readable by `awstats.pl` and by the web interface. A future
   parser that replaces `awstats.pl` must write the same format and is
   checked against `awstats.pl` output on the same logs.
2. **Releases only on "Release Go" (standing rule from Dennis).** Every push
   is built and tested by CI (`.github/workflows/build.yml`); a push to `main`
   publishes the `edge` image. A release is only started when Dennis
   explicitly says "Release Go": `gh workflow run build.yml --ref main -f
   release=true`. semantic-release then bumps `VERSION` and
   `webui/package.json`, writes `CHANGELOG.md`, tags `vX.Y.Z` and publishes
   the GitHub release and image. Never tag by hand or edit version files or
   `CHANGELOG.md`.
3. **Conventional Commits, one topic per commit** (`feat`, `fix`, `docs`,
   `chore`, `ci`, `build`, `test`, `refactor`; `!` or `BREAKING CHANGE:` for
   breaking changes). The type decides the release.
4. **Security is part of every change.** Review each change for security
   problems before committing it (log lines, referrers, user agents and API
   parameters are attacker-controlled; the interface shows IP addresses and
   must stay behind authentication). Before a release, audit the whole
   repository. Never commit real logs, real data files, IP addresses of real
   visitors or credentials; test data is generated (`webui/testdata/`).
5. **No references to other projects** of the maintainer in this repository.
6. Documentation and code comments in English; the interface text is German.
7. Keep upstream copyright and license notices in the Perl files (GPL).
8. **Commits are signed with Dennis' SSH key on his av-linux machine**
   (standing rule). Never commit or push from a cloud session; prepare the
   changes and let them be committed there.
9. **No `Claude-Session:` line or session link in commit messages** (standing
   rule).
