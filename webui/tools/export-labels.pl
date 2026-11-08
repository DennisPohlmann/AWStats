#!/usr/bin/perl
# Exports AWStats' detection label tables (lib/*.pm) to JSON for the Node UI.
# Usage (in webui/): perl tools/export-labels.pl ../wwwroot/cgi-bin/lib > server/labels.json
use strict; no strict 'vars'; use JSON::PP; use File::Spec;
my $lib = File::Spec->rel2abs(shift // die "usage: $0 <awstats lib dir>\n"); # require needs an absolute path
for (qw(browsers operating_systems robots search_engines domains status_http worms)) { require "$lib/$_.pm"; }
sub clean { my %h = %{$_[0]}; for (values %h) { s/<[^>]*>//g; s/&nbsp;/ /g; } \%h }
print JSON::PP->new->canonical->encode({
  browsers => clean(\%BrowsersHashIDLib),
  os => clean(\%OSHashLib), robots => clean(\%RobotsHashIDLib),
  searchEngines => clean(\%SearchEnginesHashLib), domains => clean(\%DomainsHashIDLib),
  http => clean(\%httpcodelib), worms => clean(\%WormsHashLib),
});
