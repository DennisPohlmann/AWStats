# AWStats - two container images from one Dockerfile:
#
#   web      (default target, ghcr.io/dennispohlmann/awstats): the web
#            interface. It only reads the data directory (mounted read-only).
#   updater  (--target updater, ghcr.io/dennispohlmann/awstats-updater): reads
#            the configs and the logs and writes the data files.
#
#   docker build -t awstats .
#   docker build --target updater -t awstats-updater .
#
#   docker run -d --name awstats -p 127.0.0.1:3000:3000 \
#     -v /var/lib/awstats:/var/lib/awstats:ro awstats
#   docker run -d --name awstats-updater \
#     -v /etc/awstats:/etc/awstats:ro -v /var/log/nginx:/var/log/nginx:ro \
#     -v /var/lib/awstats:/var/lib/awstats awstats-updater
#
# The updater updates every /etc/awstats/awstats.<site>.conf once an hour
# (default command "-all -every=1h"); "docker run --rm awstats-updater
# -config=example.com" updates one site once. DirData in the configs must be
# /var/lib/awstats, LogFile a path mounted into the container, and the data
# directory writable for uid 1000. For the IP search in the raw logs, mount
# the configs and logs into the web container too (or set
# AWSTATS_LOG_FILES). See README.md and webui/README.md.

# Base images pinned by digest, so Dependabot can update them (a new digest
# behind the same tag is an Alpine, Node or Go patch).
FROM golang:1.26-alpine@sha256:87c65a688975c0b1d4585d204874c45ca78354670e3a5f11368c5a492303d696 AS go
ARG VERSION=dev
WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /awstats ./cmd/awstats

FROM alpine:3.24@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS updater
ARG VERSION=dev
LABEL org.opencontainers.image.title="awstats-updater" \
      org.opencontainers.image.description="AWStats updater: web server logs to AWStats data files" \
      org.opencontainers.image.licenses="GPL-3.0-or-later" \
      org.opencontainers.image.version="${VERSION}"
# Alpine security updates newer than the base image; sh stays for LogFile="command |".
RUN apk upgrade --no-cache \
 && adduser -D -H -u 1000 awstats
COPY --from=go /awstats /usr/local/bin/awstats
COPY LICENSE /usr/share/licenses/awstats/LICENSE
USER awstats
ENTRYPOINT ["awstats"]
CMD ["-all", "-every=1h"]

FROM node:26-alpine@sha256:8996bb5821d62a8fd9bb27c1cb07462f3fdce6e47d640c6a881ead0b064ce216 AS build
ENV NPM_CONFIG_UPDATE_NOTIFIER=false
WORKDIR /src
COPY webui/package.json webui/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY webui/ ./
RUN npm run build

# web: the last stage, so a plain "docker build" gives the web interface.
FROM node:26-alpine@sha256:8996bb5821d62a8fd9bb27c1cb07462f3fdce6e47d640c6a881ead0b064ce216 AS web
ARG VERSION=dev
LABEL org.opencontainers.image.title="awstats" \
      org.opencontainers.image.description="AWStats web interface for AWStats data files" \
      org.opencontainers.image.licenses="GPL-3.0-or-later" \
      org.opencontainers.image.version="${VERSION}"
WORKDIR /app
# Alpine security updates newer than the base image. npm, npx, corepack and
# yarn are not needed at run time (the server has no dependencies) and only
# add packages for the image scan to find.
RUN apk upgrade --no-cache \
 && rm -rf /usr/local/lib/node_modules/npm /usr/local/lib/node_modules/corepack \
      /usr/local/bin/npm /usr/local/bin/npx /usr/local/bin/corepack \
      /opt/yarn-* /usr/local/bin/yarn /usr/local/bin/yarnpkg /root/.npm
ENV NODE_ENV=production \
    HOST=0.0.0.0 \
    PORT=3000 \
    AWSTATS_DATA_DIR=/var/lib/awstats \
    AWSTATS_CONF_DIR=/etc/awstats
COPY --from=build /src/server ./server
COPY --from=build /src/web/dist ./web/dist
COPY LICENSE ./
USER node
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s \
  CMD node -e "fetch('http://127.0.0.1:'+(process.env.PORT||3000)+'/healthz').then(r=>process.exit(r.ok?0:1),()=>process.exit(1))"
CMD ["node", "server/index.js"]
