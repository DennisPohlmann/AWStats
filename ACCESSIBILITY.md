# Accessibility

The web interface in `webui/` should be usable by everyone, including people
who use a keyboard instead of a mouse, a screen reader, zoom or a high
contrast setting, and on small phone screens.

## Goals

- Work in current desktop and mobile browsers, from about 360 px wide.
- Follow the [Web Content Accessibility Guidelines (WCAG) 2.2](https://www.w3.org/TR/WCAG22/)
  at level AA: enough color contrast in light and dark mode, a visible
  keyboard focus, labelled controls, and no content that works only on hover.
- Charts never carry information alone: every chart has a table with the same
  values, and colors are not the only way to tell series apart.

The interface has not had a formal accessibility audit. The classic
`awstats.pl` HTML output is kept as it is and is not covered by these goals.

## Reporting a problem

Accessibility problems are bugs. Please open a
[bug report](https://github.com/DennisPohlmann/AWStats/issues/new?template=bug_report.yml) and tell us:

- the page (tab) and the period you looked at;
- the browser, the operating system and the assistive technology you use
  (for example NVDA, VoiceOver or TalkBack), with versions;
- what you expected and what happened.
