#!/bin/sh
# Builds test logs and AWStats data files with the Go updater (cmd/awstats).
# Usage: testdata/build.sh   (needs Go and Node.js)
set -e
cd "$(dirname "$0")"; T="$(pwd)"
mkdir -p logs data conf
(cd ../.. && go build -o "$T/awstats" ./cmd/awstats)
node genlogs.js logs/apache-access.log 120 1
node genlogs.js logs/nginx-access.log 60 7
for site in apache nginx; do
  sed -e "s#^LogFile=.*#LogFile=\"$T/logs/$site-access.log\"#" \
      -e "s#^DirData=.*#DirData=\"$T/data\"#" \
      -e "s#^SiteDomain=.*#SiteDomain=\"www.example.com\"#" \
      -e "s#^HostAliases=.*#HostAliases=\"www.example.com localhost 127.0.0.1\"#" \
      -e 's#^DNSLookup=.*#DNSLookup=0#' ../../conf/awstats.model.conf > "conf/awstats.$site.conf"
  rm -f data/awstats*."$site".txt
  ./awstats -config=$site -configdir="$T/conf" -update | tail -1
done
rm -f awstats
# Simulate log rotation for the IP search: older half -> nginx-access.log.1.gz, newer half stays.
n=$(($(wc -l < logs/nginx-access.log) / 2))
head -n $n logs/nginx-access.log | gzip > logs/nginx-access.log.1.gz
tail -n +$((n + 1)) logs/nginx-access.log > logs/tmp && mv logs/tmp logs/nginx-access.log
