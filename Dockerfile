# AWStats web interface - container image.
#
#   docker build -t awstats .
#   docker run -d --name awstats -p 127.0.0.1:3000:3000 \
#     -v /var/lib/awstats:/var/lib/awstats:ro awstats
#
# The AWStats data directory is mounted read-only; the interface never
# writes to it. Optional, for the IP search in the raw logs: mount the
# AWStats configs at /etc/awstats and the logs at the path LogFile names
# there (or set AWSTATS_LOG_FILES). See webui/README.md.

# Base image written out in both FROM lines, pinned by digest, so Dependabot
# can update it (a new digest behind the same tag is an Alpine or Node patch).
FROM node:26-alpine@sha256:8996bb5821d62a8fd9bb27c1cb07462f3fdce6e47d640c6a881ead0b064ce216 AS build
ENV NPM_CONFIG_UPDATE_NOTIFIER=false
WORKDIR /src
COPY webui/package.json webui/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY webui/ ./
RUN npm run build

FROM node:26-alpine@sha256:8996bb5821d62a8fd9bb27c1cb07462f3fdce6e47d640c6a881ead0b064ce216
ARG VERSION=dev
LABEL org.opencontainers.image.title="awstats" \
      org.opencontainers.image.description="Web interface for AWStats data files" \
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
