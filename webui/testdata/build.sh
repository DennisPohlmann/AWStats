#!/bin/sh
# Builds test logs and AWStats data files with the original awstats.pl.
# Usage: testdata/build.sh /path/to/awstats/wwwroot/cgi-bin
set -e
AW="$(cd "${1:?path to awstats wwwroot/cgi-bin}" && pwd)"
cd "$(dirname "$0")"; T="$(pwd)"
mkdir -p logs data conf
node genlogs.js logs/apache-access.log 120 1
node genlogs.js logs/nginx-access.log 60 7
for site in apache nginx; do
  sed -e "s#^LogFile=.*#LogFile=\"$T/logs/$site-access.log\"#" \
      -e "s#^DirData=.*#DirData=\"$T/data\"#" \
      -e "s#^SiteDomain=.*#SiteDomain=\"www.example.com\"#" \
      -e "s#^HostAliases=.*#HostAliases=\"www.example.com localhost 127.0.0.1\"#" \
      -e 's#^DNSLookup=.*#DNSLookup=0#' "$AW/awstats.model.conf" > "conf/awstats.$site.conf"
  rm -f data/awstats*."$site".txt
  perl "$AW/awstats.pl" -config=$site -configdir="$T/conf" -update | tail -1
done
# Simulate log rotation for the IP search: older half -> nginx-access.log.1.gz, newer half stays.
n=$(($(wc -l < logs/nginx-access.log) / 2))
head -n $n logs/nginx-access.log | gzip > logs/nginx-access.log.1.gz
tail -n +$((n + 1)) logs/nginx-access.log > logs/tmp && mv logs/tmp logs/nginx-access.log
