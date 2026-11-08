# Third-party licenses

This fork of AWStats is licensed under GPL-3.0-or-later (see [LICENSE](LICENSE)).
It ships or uses the third-party components below. All of them are
compatible with GPL-3.0-or-later.

## AWStats

`awstats.pl`, its plugins, tools and the detection tables in
`wwwroot/cgi-bin/lib/` were written by Laurent Destailleur and contributors
of [eldy/awstats](https://github.com/eldy/awstats), GPL-3.0-or-later. The web
interface reads labels from those tables (`webui/server/labels.json`, made
by `webui/tools/export-labels.pl`).

## Browser bundle of the web interface

The interface in `webui/web/dist` (in the container image under
`/app/web/dist`) bundles these npm packages. The license texts are in each
package (`webui/node_modules/<package>/LICENSE` after `npm ci`). Versions as
pinned in `webui/package-lock.json`.

| Package | Version | License | Project |
|---------|---------|---------|---------|
| react | 18.3.1 | MIT | https://github.com/facebook/react |
| react-dom | 18.3.1 | MIT | https://github.com/facebook/react |
| scheduler | 0.23.2 | MIT | https://github.com/facebook/react |
| echarts | 6.1.0 | Apache-2.0 | https://github.com/apache/echarts |
| zrender | 6.1.0 | BSD-3-Clause | https://github.com/ecomfe/zrender |
| tslib | 2.3.0 | 0BSD | https://github.com/microsoft/tslib |

The Node.js server in `webui/server/` has no npm dependencies. Build-only
tools (Vite and its plugins) are not shipped.

## Container image

The image is based on the official `node:26-alpine` image: Node.js (MIT and
the licenses listed in its `LICENSE` file) on Alpine Linux, whose packages
carry their own licenses (see `apk info -L` in the image). The SBOM attached
to every release lists all packages of the image.

## Release and CI tooling

semantic-release and its plugins (MIT) are used only in CI
(`.github/release/`) and are not shipped.
