# Contributing

Thank you for your interest in AWStats. Bug reports, ideas,
documentation fixes and code are all welcome. Please follow the
[Code of Conduct](CODE_OF_CONDUCT.md) in issues, pull requests and anywhere
else in the project.

## Reporting bugs and asking for features

- Search the [existing issues](https://github.com/DennisPohlmann/AWStats/issues) first; a comment or a reaction on an
  existing issue helps more than a duplicate.
- Use the issue forms ("Bug report" or "Feature request") and fill in the
  version and your setup. Logs help a lot.
- Remove passwords, tokens, private IP addresses and anything else private
  from logs and configs before you post them.
- **Security problems are not reported in issues**; see [SECURITY.md](SECURITY.md).

## How changes get into `main`

AWStats is maintained by one person. The maintainer merges changes locally
and pushes them to `main`; pull requests are not merged through the GitHub
interface.

Pull requests are still very welcome as **proposals**: the maintainer reviews
them, may adjust them, and applies them by hand, keeping you as author or
crediting you with a `Co-authored-by:` line. The pull request is then closed
with a link to the commit. For larger changes, please open an issue first so
we can agree on the approach before you spend time on it.

## Development

The log analyzer is the Perl script `wwwroot/cgi-bin/awstats.pl`; the web
interface is the Node.js application in [`webui/`](webui/README.md). Before
you propose a change to the web interface, run:

```bash
cd webui
npm ci
sh testdata/build.sh ../wwwroot/cgi-bin   # test logs and data files, needs Perl
npm test
npm run build
```

Please also:

- Keep changes focused: one topic per pull request.
- Update the documentation in the same change when behavior, configuration
  or commands change.
- Write documentation and code comments in English.
- Keep the data file format of `awstats.pl` compatible: existing data files
  must stay readable.

## Commit messages

Releases are made by [semantic-release](https://semantic-release.gitbook.io/)
from [Conventional Commits](https://www.conventionalcommits.org/); nobody tags
by hand. The type of a commit decides the next version:

| Type | Effect |
| --- | --- |
| `fix: ...` | patch release |
| `feat: ...` | minor release |
| `feat!: ...` or a `BREAKING CHANGE:` footer | major release |
| `docs:`, `chore:`, `ci:`, `test:`, `refactor:`, `build:` | no release |

Every push builds and tests; a release is only made by hand from `main`
(Actions -> build -> Run workflow with "release" ticked). It tags the
version, writes CHANGELOG.md, publishes the GitHub release and the container
image `ghcr.io/dennispohlmann/awstats` with the tags `<version>`,
`<major>.<minor>` and `latest`. Every push to `main` publishes `edge`.

Write the subject in plain English and say in the body why the change is
needed and what a user notices.

## License

By contributing, you agree that your contribution is licensed under the
[GNU General Public License v3.0 or later](LICENSE), the license of this
repository.
