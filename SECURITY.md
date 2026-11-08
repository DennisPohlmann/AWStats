# Security policy

## Supported versions

Only the latest release receives security fixes. Please update
to the [latest release](https://github.com/DennisPohlmann/AWStats/releases/latest) before you report a problem, and
check whether it still happens there.

## Reporting a vulnerability

**Please do not report security vulnerabilities in public issues, pull
requests or discussions.**

Report them privately through GitHub:

1. Open the [Security tab](https://github.com/DennisPohlmann/AWStats/security) of this repository.
2. Choose **Report a vulnerability** and fill in the form
   ([direct link](https://github.com/DennisPohlmann/AWStats/security/advisories/new)).

Only the maintainer can read the report. If the form is not available to you,
open an issue that only asks for a private contact channel, without any
details of the problem.

Please include:

- the affected version and how it is installed (container image, built from source);
- what an attacker can do, and what they need for it (for example network
  access to the web interface, a crafted log line, a crafted request);
- the steps or a proof of concept to reproduce it;
- any idea for a fix, if you have one.

## What happens next

AWStats is maintained by one person in their spare time, so these are
goals, not guarantees:

- You get an answer within 7 days that the report has arrived.
- The maintainer confirms or rejects the problem and keeps you informed in the
  advisory.
- A confirmed problem is fixed in a new release; the advisory is published
  with the release and credits you, unless you prefer not to be named.

Please give the maintainer reasonable time to release a fix before you publish
details yourself.

## Scope

In scope is the code in this repository, in particular:

- the updater in `cmd/awstats`, including the parsing of log lines
- the web interface in `webui/` and its JSON API
- the container image built from this repository

Vulnerabilities in Go, Node.js or the web server belong to those projects;
please report them there.

The web interface shows IP addresses and single requests. Running it without
access control on a public address is a configuration issue rather than a
vulnerability; see the README for a reverse proxy with a login.
