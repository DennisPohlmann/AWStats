package main

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	visitTimeout             = 10000 // awstats.pl $VISITTIMEOUT, in YYYYMMDDHHMMSS units (1 hour)
	notSortedRecordTolerance = 20000
	maxLengthOfStoredURL     = 256
)

var (
	sessionsRange = []string{"0s-30s", "30s-2mn", "2mn-5mn", "5mn-15mn", "15mn-30mn", "30mn-1h", "1h+"}
	payloadRange  = []string{"0-44", "44-100", "100-500", "500-1K", "1K-2K", "2K-5K", "5K+"}
	miscList      = []string{"TotalMisc", "AddToFavourites", "JavascriptDisabled", "JavaEnabled", "DirectorSupport",
		"FlashSupport", "RealPlayerSupport", "QuickTimeSupport", "WindowsMediaPlayerSupport", "PDFSupport"}
	monthNum = map[string]string{"Jan": "01", "jan": "01", "Feb": "02", "feb": "02", "Mar": "03", "mar": "03",
		"Apr": "04", "apr": "04", "May": "05", "may": "05", "Jun": "06", "jun": "06", "Jul": "07", "jul": "07",
		"Aug": "08", "aug": "08", "Sep": "09", "sep": "09", "Oct": "10", "oct": "10", "Nov": "11", "nov": "11",
		"Dec": "12", "dec": "12"}
	allowedMethodsW = map[string]bool{"GET": true, "POST": true, "HEAD": true, "PROPFIND": true, "CHECKOUT": true,
		"LOCK": true, "PROPPATCH": true, "OPTIONS": true, "MKACTIVITY": true, "PUT": true, "MERGE": true,
		"DELETE": true, "REPORT": true, "MKCOL": true, "COPY": true, "RPC_IN_DATA": true, "RPC_OUT_DATA": true,
		"OK": true, "ERR!": true, "PRIV": true}
	streamMethods = map[string]bool{"GET": true, "MMS": true, "RTSP": true, "HTTP": true, "RTP": true}

	reIPv4      = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	reIPv4L     = regexp.MustCompile(`^::ffff:\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	reIPv6      = regexp.MustCompile(`(?i)^[0-9A-F]*:`)
	reHostPort  = regexp.MustCompile(`^([^:]+):[0-9]+$`)
	reFavicon   = regexp.MustCompile(`(?i)/favicon\.ico$`)
	reRobotsTxt = regexp.MustCompile(`(?i)/robots\.txt$`)
	reAnchor    = regexp.MustCompile(`#(\w*)$`)
	reExt       = regexp.MustCompile(`\.(\w{1,6})$`)
	reRFC3986   = regexp.MustCompile(`(?i)%([46][1-9A-F]|[57][0-9A]|3[0-9]|2D|2E|5F|7E)`)
	reCode3     = regexp.MustCompile(`^\d\d\d$`)
	reReferer   = regexp.MustCompile(`^(\w+)://([^/:]+)(:\d+|)`)
	reHTTP      = regexp.MustCompile(`(?i)^http`)
	reCacheRel  = regexp.MustCompile(`^(cache|related):[^\+]+`)
	reSep1      = regexp.MustCompile(`(?i)%0[ad]`)
	reSep2      = regexp.MustCompile(`(?i)%2[02789abc]`)
	reSep3      = regexp.MustCompile(`(?i)%3a`)
	reSpaces    = regexp.MustCompile(` +`)
	reUpToEq    = regexp.MustCompile(`.*=`)
	reDomainTLD = regexp.MustCompile(`\.(\w+)$`)

	reEdge      = regexp.MustCompile(`(?i)edge\/([\d]+)`)
	reMSIE      = regexp.MustCompile(`(?i)msie([+_ ]|)([\d\.]*)`)
	reMSIE11    = regexp.MustCompile(`(?i)trident\/7\.\d*\;([a-zA-Z;+_ ]+|)rv:([\d\.]*)`)
	reNetscape  = regexp.MustCompile(`(?i)netscape.?\/([\d\.]*)`)
	reFirefox   = regexp.MustCompile(`(?i)firefox\/([\d\.]*)`)
	reOpera     = regexp.MustCompile(`(?i)opera\/9\.80\s.+\sversion\/([\d\.]+)|ope?ra?[\/\s]([\d\.]+)`)
	reSafari    = regexp.MustCompile(`(?i)safari\/([\d\.]*)`)
	reSafariVer = regexp.MustCompile(`(?i)version\/([\d\.]*)`)
	reChrome    = regexp.MustCompile(`(?i)chrome\/([\d\.]*)`)
	reKonqueror = regexp.MustCompile(`(?i)konqueror\/([\d\.]*)`)
	reSVN       = regexp.MustCompile(`(?i)svn\/([\d\.]*)`)
	reMozilla   = regexp.MustCompile(`(?i)mozilla(\/|)([\d\.]*)`)
	reNotIE     = regexp.MustCompile(`(?i)webtv|omniweb|opera`)
	reNotNS     = regexp.MustCompile(`(?i)gecko|compatible|opera|galeon|safari|charon`)
	reNotFF     = regexp.MustCompile(`(?i)flock`)
	reNotSafari = regexp.MustCompile(`(?i)android|arora|chrome|shiira|webpositive`)
	reMiscParam = []struct {
		re   *regexp.Regexp
		yes  string
		misc string
	}{
		{regexp.MustCompile(`(?i)^nojs=(\w+)`), "y", "JavascriptDisabled"},
		{regexp.MustCompile(`(?i)^java=(\w+)`), "true", "JavaEnabled"},
		{regexp.MustCompile(`(?i)^shk=(\w+)`), "y", "DirectorSupport"},
		{regexp.MustCompile(`(?i)^fla=(\w+)`), "y", "FlashSupport"},
		{regexp.MustCompile(`(?i)^rp=(\w+)`), "y", "RealPlayerSupport"},
		{regexp.MustCompile(`(?i)^mov=(\w+)`), "y", "QuickTimeSupport"},
		{regexp.MustCompile(`(?i)^wma=(\w+)`), "y", "WindowsMediaPlayerSupport"},
		{regexp.MustCompile(`(?i)^pdf=(\w+)`), "y", "PDFSupport"},
	}
	reScreen         = regexp.MustCompile(`(?i)^screen=(\d+)x(\d+)`)
	reWhite          = regexp.MustCompile(`\s`)
	reRobotsTxtLoose = regexp.MustCompile(`robots.txt$`)
	reDigits         = regexp.MustCompile(`(\d+)`)
	reShortDate      = regexp.MustCompile(`^..:..:..:`)
)

// pint is Perl's int() on a string: the leading number, 0 if there is none.
func pint(s string) int64 {
	s = strings.TrimLeft(s, " \t\n")
	end := 0
	if end < len(s) && (s[end] == '-' || s[end] == '+') {
		end++
	}
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	n, _ := strconv.ParseInt(s[:end], 10, 64)
	return n
}

func truthy(s string) bool { return s != "" && s != "0" }

func decodeRFC3986(s string) string {
	return reRFC3986.ReplaceAllStringFunc(s, func(m string) string {
		b, _ := strconv.ParseUint(m[1:], 16, 8)
		return string(rune(b))
	})
}

// changeWordSeparatorsIntoSpace mirrors ChangeWordSeparatorsIntoSpace: tr/+'()"*,:/ /s squeezes runs.
func changeWordSeparatorsIntoSpace(s string) string {
	s = reSep1.ReplaceAllString(s, " ")
	s = reSep2.ReplaceAllString(s, " ")
	s = reSep3.ReplaceAllString(s, " ")
	var b strings.Builder
	last := false
	for _, r := range s {
		if strings.ContainsRune(`+'()"*,:`, r) {
			if !last {
				b.WriteByte(' ')
			}
			last = true
			continue
		}
		last = false
		b.WriteRune(r)
	}
	return b.String()
}

// squeezeSpacesToPlus is tr/ /\+/s.
func squeezeSpacesToPlus(s string) string { return reSpaces.ReplaceAllString(s, "+") }

func checkSum(s string) int64 {
	var sum int64
	j := 0
	for i := 0; i < len(s); i++ {
		sum += int64(s[i]) << (8 * j)
		if j > 3 {
			j = 0
		} else {
			j++
		}
	}
	return sum
}

var regDate = regexp.MustCompile(`(\d\d\d\d)(\d\d)(\d\d)(\d\d)(\d\d)(\d\d)`)

func timeOf(t int64) time.Time {
	m := regDate.FindStringSubmatch(strconv.FormatInt(t, 10))
	if m == nil {
		return time.Time{}
	}
	a := make([]int, 6)
	for i := range a {
		a[i], _ = strconv.Atoi(m[i+1])
	}
	return time.Date(a[0], time.Month(a[1]), a[2], a[3], a[4], a[5], 0, time.Local)
}

func sessionRange(start, end int64) string {
	if start == end {
		return sessionsRange[0]
	}
	d := timeOf(end).Unix() - timeOf(start).Unix()
	for i, lim := range []int64{30, 120, 300, 900, 1800, 3600} {
		if d <= lim {
			return sessionsRange[i]
		}
	}
	return sessionsRange[6]
}

func bandwidthRange(p int64) string {
	for i, lim := range []int64{44, 100, 500, 1024, 2048, 5120} {
		if p <= lim {
			return payloadRange[i]
		}
	}
	return payloadRange[6]
}

func stamp(t time.Time) int64 {
	n, _ := strconv.ParseInt(t.Format("20060102150405"), 10, 64)
	return n
}

// updater runs one -update for one config, like the update process of awstats.pl.
type updater struct {
	c    *config
	site string // config name, the suffix of the data files
	lf   *logFormat
	lib  *lib

	regDefaults []*regexp.Regexp

	regMisc, regDefault, regLocal, regTruncURL, regRefNoQuery, regQuerySep *regexp.Regexp

	now, tomorrow int64

	lastLine, lastLineNumber, lastLineOffset, lastLineChecksum int64

	parsed, oldLines, newLines, corrupted, dropped, comments, blanks int64
	newLinePhase                                                     bool

	cur                                       *month
	saved                                     []string // data files written
	tmpRobot, tmpBrowser, tmpOS, tmpRefServer map[string]string
	dnsCache                                  map[string]string
}

func newUpdater(c *config, now time.Time) (*updater, error) {
	if c.LogType != "W" && c.LogType != "S" {
		return nil, fmt.Errorf("LogType=%s (mail or ftp logs) is not supported", c.LogType)
	}
	lf, err := newLogFormat(c.LogFormat, c.LogSeparator)
	if err != nil {
		return nil, err
	}
	lb, err := loadLib(c.LevelForRobots, c.LevelForSearchEngines)
	if err != nil {
		return nil, err
	}
	u := &updater{c: c, lf: lf, lib: lb, now: stamp(now), tomorrow: stamp(now.Add(24 * time.Hour)),
		tmpRobot: map[string]string{}, tmpBrowser: map[string]string{}, tmpOS: map[string]string{},
		tmpRefServer: map[string]string{}, dnsCache: map[string]string{}}
	qs := regexp.QuoteMeta(c.URLQuerySeparators)
	u.regMisc = regexp.MustCompile("^" + regexp.QuoteMeta(c.MiscTrackerUrl))
	def := regexp.QuoteMeta("/"+c.DefaultFile[0]) + "$"
	if c.URLNotCaseSensitive {
		def = "(?i)" + def
	}
	u.regDefault = regexp.MustCompile(def)
	for _, e := range c.DefaultFile {
		re, err := regexp.Compile("/" + e + "$") // the value is used as a regex, like awstats.pl does
		if err != nil {
			return nil, fmt.Errorf("DefaultFile %q: %w", e, err)
		}
		u.regDefaults = append(u.regDefaults, re)
	}
	site := strings.Replace(strings.ToLower(c.SiteDomain), "www.", "", 1)
	u.regLocal = regexp.MustCompile(`(?i)^(www\.|)` + regexp.QuoteMeta(site))
	u.regTruncURL = regexp.MustCompile("([" + qs + "])(.*)$")
	u.regRefNoQuery = regexp.MustCompile("^([^" + qs + "]+)")
	u.regQuerySep = regexp.MustCompile("[" + qs + "].*$")
	return u, nil
}

func matchAny(res []*regexp.Regexp, s string) bool {
	for _, re := range res {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}

// parseDate mirrors the date handling of the main loop. It returns day, month, year, h, m, s as
// strings plus ok=false for a corrupted date.
func parseDate(d string) ([]string, bool) {
	if strings.Contains(d, ",") { // %time6 "mm/dd/yyyy, hh:mm:ss"
		d = strings.Replace(d, ",", "", 1)
		sp := strings.Fields(d)
		if len(sp) >= 2 {
			dp := strings.Split(sp[0], "/")
			tp := strings.Split(sp[1], ":")
			for len(dp) < 3 {
				dp = append(dp, "")
			}
			for len(tp) < 3 {
				tp = append(tp, "")
			}
			pad := func(s string) string {
				if len(s) == 1 {
					return "0" + s
				}
				return s
			}
			d = fmt.Sprintf("%s-%s-%s %s:%s:%s", dp[2], pad(dp[0]), pad(dp[1]), pad(tp[0]), pad(tp[1]), pad(tp[2]))
		}
	}
	// tr/,-\/ \tT/::::::/s
	var b strings.Builder
	last := false
	for i := 0; i < len(d); i++ {
		ch := d[i]
		if strings.IndexByte(",-/ \tT", ch) >= 0 {
			if !last {
				b.WriteByte(':')
			}
			last = true
			continue
		}
		last = false
		b.WriteByte(ch)
	}
	orig := b.String()
	p := strings.Split(orig, ":")
	for len(p) < 6 {
		p = append(p, "")
	}
	switch {
	case !truthy(p[1]): // unix timestamp
		t := time.Unix(pint(orig), 0)
		p = []string{fmt.Sprint(t.Day()), fmt.Sprint(int(t.Month())), fmt.Sprint(t.Year()),
			fmt.Sprint(t.Hour()), fmt.Sprint(t.Minute()), fmt.Sprint(t.Second())}
	case len(p[0]) == 4:
		p[0], p[2] = p[2], p[0]
	case reShortDate.MatchString(orig):
		p[2] = fmt.Sprint(pint(p[2]) + 2000)
		p[0], p[1] = p[1], p[0]
	case len(p[0]) == 3:
		p[0], p[1] = p[1], p[0]
		tmp := p[5]
		p[5], p[4], p[3] = p[4], p[3], p[2]
		if truthy(tmp) {
			p[2] = tmp
		} else {
			p[2] = strconv.Itoa(time.Now().Year())
		}
	}
	if n, ok := monthNum[p[1]]; ok {
		p[1] = n
	}
	if pint(p[1]) <= 0 {
		return nil, false
	}
	return p, true
}

// processLine handles one log line. It returns errFormat when the log does not match LogFormat at all.
func (u *updater) processLine(line string, lineOffset, baseLineNb int64) error {
	c, f := u.c, u.lf
	if len(line) > 0 && line[0] == 0 {
		u.corrupted++
		return u.checkFormat()
	}
	line = strings.TrimSuffix(line, "\r")
	u.parsed++
	m := f.re.FindStringSubmatch(line)
	if m == nil {
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			u.comments++
		} else if strings.TrimSpace(line) == "" {
			u.blanks++
		} else {
			u.corrupted++
		}
		if u.parsed >= int64(c.NbOfLinesForCorruptedLog) && u.parsed == u.corrupted+u.comments+u.blanks {
			return fmt.Errorf("format error: no line of the first %d matches LogFormat=%q, for example: %s", u.parsed, c.LogFormat, line)
		}
		return nil
	}
	field := m[1:]
	get := func(pos int) string {
		if pos < 0 || pos >= len(field) {
			return ""
		}
		return field[pos]
	}
	if f.vh >= 0 && !strings.EqualFold(field[f.vh], c.SiteDomain) && !matchAny(c.HostAliases, field[f.vh]) {
		u.dropped++
		return nil
	}
	if c.LogType != "M" {
		field[f.url] = reWhite.ReplaceAllString(field[f.url], "%20")
	}
	method := field[f.method]
	switch {
	case c.LogType == "W" && allowedMethodsW[method]:
	case (c.LogType == "W" || c.LogType == "S") && streamMethods[strings.ToUpper(method)]:
	case strings.Contains(line, "#Fields:"):
		u.comments++
		return nil
	default:
		u.dropped++
		return nil
	}

	dp, ok := parseDate(field[f.date])
	if !ok {
		u.corrupted++
		return nil
	}
	yearRec, monthRec, hourRec := pint(dp[2]), pint(dp[1]), int(pint(dp[3]))
	ymd := fmt.Sprintf("%s%02d%02d", dp[2], pint(dp[1]), pint(dp[0]))
	timeRec := ((pint(ymd)*100+pint(dp[3]))*100+pint(dp[4]))*100 + pint(dp[5])
	if timeRec < 10000000000000 || timeRec > u.tomorrow || hourRec < 0 || hourRec > 23 {
		u.corrupted++
		return nil
	}
	if u.newLinePhase {
		if timeRec < u.lastLine-notSortedRecordTolerance {
			u.corrupted++
			return nil
		}
	} else {
		if timeRec <= u.lastLine {
			u.oldLines++
			return nil
		}
		u.newLinePhase = true
	}
	if timeRec > u.lastLine {
		u.lastLine = timeRec
	}

	host := get(f.host)
	switch {
	case len(c.SkipHosts) > 0 && (matchAny(c.SkipHosts, host) || (f.hostr > 0 && matchAny(c.SkipHosts, get(f.hostr)))),
		len(c.SkipFiles) > 0 && matchAny(c.SkipFiles, field[f.url]),
		len(c.SkipUserAgents) > 0 && f.agent >= 0 && matchAny(c.SkipUserAgents, get(f.agent)),
		len(c.SkipReferrers) > 0 && f.referer >= 0 && matchAny(c.SkipReferrers, get(f.referer)),
		len(c.OnlyHosts) > 0 && !matchAny(c.OnlyHosts, host) && (f.hostr <= 0 || !matchAny(c.OnlyHosts, get(f.hostr))),
		len(c.OnlyUsers) > 0 && !matchAny(c.OnlyUsers, get(f.logname)),
		len(c.OnlyFiles) > 0 && !matchAny(c.OnlyFiles, field[f.url]),
		len(c.OnlyUserAgents) > 0 && !matchAny(c.OnlyUserAgents, get(f.agent)):
		u.dropped++
		return nil
	}

	dateRec := fmt.Sprintf("%04d%02d", yearRec, monthRec)
	if u.cur == nil || dateRec > u.cur.year+u.cur.mon {
		if u.cur != nil {
			if err := u.saveMonth(baseLineNb+u.parsed, lineOffset, checkSum(line)); err != nil {
				return err
			}
		}
		u.cur = newMonth(fmt.Sprintf("%04d", yearRec), fmt.Sprintf("%02d", monthRec), c.TrapInfosForHTTPErrorCodes)
	}
	h := u.cur
	u.newLines++

	// URL
	url := decodeRFC3986(field[f.url])
	if c.URLNotCaseSensitive {
		url = strings.ToLower(url)
	}
	anchor := ""
	if am := reAnchor.FindStringSubmatchIndex(url); am != nil {
		anchor = url[am[2]:am[3]]
		url = url[:am[0]]
	}
	var urlNoQuery, standaloneQuery string
	query := get(f.query)
	if c.URLWithQuery {
		urlNoQuery = url
		tokenQuery := ""
		found := false
		if qm := u.regTruncURL.FindStringSubmatchIndex(urlNoQuery); qm != nil {
			found = true
			tokenQuery, standaloneQuery = urlNoQuery[qm[2]:qm[3]], urlNoQuery[qm[4]:qm[5]]
			urlNoQuery = urlNoQuery[:qm[0]]
		}
		if !found && f.query >= 0 && truthy(query) && query != "-" {
			found, tokenQuery, standaloneQuery = true, "?", query
			url += "?" + query
		}
		if found {
			nq := standaloneQuery
			match := func(p, name string) bool {
				re := "^" + name + "="
				if c.URLNotCaseSensitive {
					re = "(?i)" + re
				}
				ok, _ := regexp.MatchString(re, p)
				return ok
			}
			if len(c.URLWithQueryWithOnly) > 0 {
				var keep []string
				for _, name := range c.URLWithQueryWithOnly {
					for _, p := range strings.Split(standaloneQuery, "&") {
						if match(p, name) {
							keep = append(keep, p)
							break
						}
					}
				}
				nq = strings.Join(keep, "&")
			} else if len(c.URLWithQueryWithout) > 0 {
				var keep []string
				for _, p := range strings.Split(standaloneQuery, "&") {
					drop := false
					for _, name := range c.URLWithQueryWithout {
						if match(p, name) {
							drop = true
							break
						}
					}
					if !drop {
						keep = append(keep, p)
					}
				}
				nq = strings.Join(keep, "&")
			}
			url = urlNoQuery
			if truthy(nq) {
				url += tokenQuery + nq
			}
		}
	} else {
		if qm := u.regTruncURL.FindStringSubmatchIndex(url); qm != nil {
			standaloneQuery = url[qm[4]:qm[5]]
			url = url[:qm[0]]
		}
		urlNoQuery = url
		if f.query >= 0 && truthy(query) && query != "-" {
			standaloneQuery = query
		}
	}
	if c.URLWithAnchor && anchor != "" {
		url += "#" + anchor
	}

	pageBool := true
	ext := "Unknown"
	if em := reExt.FindStringSubmatch(urlNoQuery); em != nil {
		ext = em[1]
	} else if strings.HasSuffix(urlNoQuery, "/") || strings.HasSuffix(urlNoQuery, `\`) {
		if em := reExt.FindStringSubmatch(c.DefaultFile[0]); em != nil {
			ext = em[1]
		}
	}
	if ext != "Unknown" {
		if _, known := u.lib.mime[ext]; c.LevelForFileTypes >= 2 || known {
			ext = strings.ToLower(ext)
		} else {
			ext = "Unknown"
		}
	}
	mime := u.lib.mime[ext]
	if c.NotPageList[ext] || (mime[1] != "" && mime[1] != "p") {
		pageBool = false
	}
	if len(c.NotPageFiles) > 0 && matchAny(c.NotPageFiles, url) {
		pageBool = false
	}

	size := int64(0)
	if f.size > 0 {
		size = pint(field[f.size])
	}
	code := field[f.code]
	counted := 0

	if u.regMisc.MatchString(urlNoQuery) {
		found := 0
		for _, p := range strings.Split(standaloneQuery, "&") {
			if sm := reScreen.FindStringSubmatch(p); sm != nil {
				found++
				h.screensize[sm[1]+"x"+sm[2]]++
				continue
			}
			for _, mp := range reMiscParam {
				if mm := mp.re.FindStringSubmatch(p); mm != nil {
					found++
					if mm[1] == mp.yes {
						h.miscH[mp.misc]++
					}
					break
				}
			}
		}
		if found > 0 {
			h.miscH["TotalMisc"]++
		}
	}
	if reFavicon.MatchString(urlNoQuery) {
		if pint(code) != 404 {
			h.miscH["AddToFavourites"]++
		}
		counted = 1
		h.timeNvH[hourRec]++
		if pint(code) != 404 && f.size > 0 {
			h.timeNvK[hourRec] += size
		}
	}
	notViewed := func() {
		if pageBool {
			h.timeNvP[hourRec]++
		}
		h.timeNvH[hourRec]++
		if f.size > 0 {
			h.timeNvK[hourRec] += size
		}
	}
	if counted == 0 && c.LevelForWorms > 0 {
		for _, w := range u.lib.worms {
			if w.re.MatchString(field[f.url]) {
				worm := u.lib.wormID[w.id]
				if worm == "" {
					worm = "unknown"
				}
				h.wormH[worm]++
				h.wormK[worm] += size
				h.wormL[worm] = timeRec
				counted = 2
				notViewed()
				break
			}
		}
	}
	if counted == 0 && (c.LogType == "W" || c.LogType == "S") {
		icode := pint(code)
		if c.ValidHTTPCodes[code] {
			if icode == 304 && f.size > 0 {
				size, field[f.size] = 0, "0"
			}
			if icode == 200 && mime[1] == "d" && !reRobotsTxtLoose.MatchString(urlNoQuery) {
				d := h.download(urlNoQuery)
				d[0]++
				d[2] += size
			}
		} else if icode == 206 && mime[1] == "d" {
			d := h.download(urlNoQuery)
			d[2] += size
			d[1]++
			if f.size > 0 {
				h.dayBytes[ymd] += size
				h.timeK[hourRec] += size
			}
			counted = 6
		} else {
			if !reCode3.MatchString(code) {
				code = "999"
			}
			h.errorsH[code]++
			h.errorsK[code] += size
			for _, tc := range c.TrapInfosForHTTPErrorCodes {
				if pint(code) != pint(tc) {
					continue
				}
				nu := url
				if len(nu) > maxLengthOfStoredURL {
					nu = nu[:maxLengthOfStoredURL]
				}
				nu = u.regQuerySep.ReplaceAllString(nu, "")
				h.siderH[tc][nu]++
				if f.referer >= 0 && strings.ContainsAny(c.ShowHTTPErrorsPageDetail, "Rr") {
					r := field[f.referer]
					if !c.URLReferrerWithQuery {
						r = u.regQuerySep.ReplaceAllString(r, "")
					}
					h.refererH[tc][nu] = r
				}
				if f.host >= 0 && strings.ContainsAny(c.ShowHTTPErrorsPageDetail, "Hh") {
					hh := field[f.host]
					if !c.URLReferrerWithQuery {
						hh = u.regQuerySep.ReplaceAllString(hh, "")
					}
					h.errHostH[tc][nu] = hh
					break
				}
			}
			counted = 3
			notViewed()
		}
	}

	ua := ""
	if counted == 0 || counted == 6 {
		if f.agent >= 0 {
			if c.DecodeUA {
				field[f.agent] = strings.ReplaceAll(field[f.agent], "%20", "_")
			}
			ua = field[f.agent]
			if ua == "-" {
				ua = ""
			}
			if c.LevelForRobots > 0 {
				bot := ""
				if ua != "" {
					bot = u.tmpRobot[ua]
					if bot == "" {
						bot = "-"
						for _, r := range u.lib.robots {
							if r.re.MatchString(ua) {
								bot = r.id
								break
							}
						}
						u.tmpRobot[ua] = bot
					}
				} else {
					bot = "no_user_agent"
				}
				if bot != "-" {
					h.robotH[bot]++
					h.robotK[bot] += size
					h.robotL[bot] = timeRec
					if reRobotsTxt.MatchString(urlNoQuery) {
						h.robotR[bot]++
					}
					counted = 4
					notViewed()
				}
			}
		}
	}
	if counted == 0 && reRobotsTxt.MatchString(urlNoQuery) {
		h.robotH["unknown"]++
		h.robotK["unknown"] += size
		h.robotL["unknown"] = timeRec
		h.robotR["unknown"]++
		counted = 5
		notViewed()
	}
	if counted == 0 || counted == 6 {
		if c.LevelForFileTypes > 0 {
			if counted != 6 {
				h.ftH[ext]++
			}
			h.ftK[ext] += size
			if f.gzipin >= 0 && truthy(field[f.gzipin]) {
				in := strings.Split(field[f.gzipin], ":")
				out := strings.Split(get(f.gzipout), ":")
				if len(out) > 1 && truthy(out[1]) && len(in) > 1 {
					h.ftGzIn[ext] += pint(in[1])
					h.ftGzOut[ext] += pint(out[1])
				}
			} else if f.ratio >= 0 {
				if rm := reDigits.FindStringSubmatch(field[f.ratio]); rm != nil {
					r := pint(rm[1])
					den := r
					if f.ratioIsGzip {
						den = 100 - r
					}
					if den == 0 {
						den = 1
					}
					h.ftGzIn[ext] += int64(float64(size) * 100 / float64(den))
					h.ftGzOut[ext] += size
				}
			}
		}
		if pageBool {
			if len(c.DefaultFile) > 1 {
				for _, re := range u.regDefaults {
					if re.MatchString(url) {
						url = re.ReplaceAllString(url, "/")
						break
					}
				}
			} else {
				url = u.regDefault.ReplaceAllString(url, "/")
			}
			if h.firstTime == 0 {
				h.firstTime = timeRec
			}
			h.lastTime = timeRec
			h.dayPages[ymd]++
			h.urlP[url]++
			h.urlK[url] += size
			h.timeP[hourRec]++
		}
		if counted != 6 {
			h.timeH[hourRec]++
			h.dayHits[ymd]++
		}
		h.timeK[hourRec] += size
		h.dayBytes[ymd] += size
		if f.logname >= 0 && truthy(field[f.logname]) && field[f.logname] != "-" {
			login := strings.ReplaceAll(field[f.logname], " ", "_")
			if c.LogFormat == "6" {
				login = strings.TrimSuffix(strings.TrimPrefix(login, `"`), `"`)
			}
			if c.AuthenticatedUsersNotCaseSensitive {
				login = strings.ToLower(login)
			}
			if pageBool {
				h.loginP[login]++
			}
			if counted != 6 {
				h.loginH[login]++
			}
			h.loginK[login] += size
			h.loginL[login] = timeRec
		}
	}

	if hm := reHostPort.FindStringSubmatch(host); hm != nil {
		host = hm[1]
	}
	if counted == 0 || counted == 6 {
		u.countHost(h, host, url, ua, field, timeRec, ymd, size, pageBool, counted)
	}
	if f.cluster >= 0 {
		cl := field[f.cluster]
		if pageBool {
			h.clusterP[cl]++
		}
		h.clusterH[cl]++
		h.clusterK[cl] += size
	}
	if f.size >= 0 {
		h.filesize[bandwidthRange(pint(field[f.size]))]++
	}
	if f.rqtime >= 0 {
		h.requesttime[bandwidthRange(pint(field[f.rqtime]))]++ // same ranges as GetRequestTimeRange
	}
	return nil
}

func (u *updater) checkFormat() error {
	if u.parsed >= int64(u.c.NbOfLinesForCorruptedLog) && u.parsed == u.corrupted {
		return fmt.Errorf("format error: the log file looks binary or corrupted")
	}
	return nil
}

func (u *updater) resolve(host string) (resolved string, isIP bool) {
	ipv := 0
	if reIPv4L.MatchString(host) {
		host, ipv = strings.TrimPrefix(host, "::ffff:"), 4
	} else if reIPv4.MatchString(host) {
		ipv = 4
	} else if reIPv6.MatchString(host) {
		ipv = 6
	}
	if ipv == 0 {
		return strings.ToLower(host), false
	}
	if u.c.DNSLookup != 1 {
		return host, true
	}
	if r, ok := u.dnsCache[host]; ok {
		if r == "*" {
			return host, true
		}
		return r, false
	}
	r := "*"
	if names, err := net.LookupAddr(host); err == nil && len(names) > 0 {
		n := strings.TrimSuffix(names[0], ".")
		if !reIPv4.MatchString(n) && isASCII(n) {
			r = strings.ToLower(n)
		}
	}
	u.dnsCache[host] = r
	if r == "*" {
		return host, true
	}
	return r, false
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func (u *updater) countHost(h *month, host, url, ua string, field []string, timeRec int64, ymd string, size int64, pageBool bool, counted int) {
	c, f := u.c, u.lf
	hostResolved, isIP := u.resolve(host)
	domain := "ip"
	if !isIP {
		if dm := reDomainTLD.FindStringSubmatch(hostResolved); dm != nil {
			domain = dm[1]
		}
	}
	if pageBool {
		h.domP[domain]++
	}
	if counted != 6 {
		h.domH[domain]++
	}
	h.domK[domain] += size

	if pageBool {
		hr := hostResolved
		if tl := h.hostL[hr]; tl != 0 {
			switch {
			case timeRec > tl+visitTimeout: // a new visit
				ts, page := h.hostS[hr], h.hostU[hr]
				if page != "" {
					h.urlX[page]++
				}
				h.urlE[url]++
				h.dayVisits[ymd]++
				if h.waitS[hr] == 0 {
					h.waitL[hr], h.waitS[hr], h.waitU[hr] = tl, ts, page
				} else if ts != 0 {
					h.session[sessionRange(ts, tl)]++
				}
				h.hostS[hr], h.hostL[hr], h.hostU[hr] = timeRec, timeRec, url
			case timeRec > tl:
				h.hostL[hr], h.hostU[hr] = timeRec, url
			case timeRec == tl:
				h.hostU[hr] = url
			case timeRec < h.hostS[hr]:
				if h.waitS[hr] == 0 {
					h.waitE[hr] = url
				}
				h.hostS[hr] = timeRec
			}
		} else {
			h.waitE[hr] = url
			h.hostU[hr], h.hostS[hr], h.hostL[hr] = url, timeRec, timeRec
		}
		h.hostP[hr]++
	}
	h.hostH[hostResolved]++
	h.hostK[hostResolved] += size

	if f.agent >= 0 {
		if c.LevelForBrowsers > 0 {
			u.countBrowser(h, ua, timeRec, pageBool)
		}
		if c.LevelForOS > 0 {
			os := u.tmpOS[ua]
			if os == "" {
				os = "Unknown"
				for _, o := range u.lib.os {
					if o.re.MatchString(ua) {
						os = u.lib.osID[o.id]
						break
					}
				}
				u.tmpOS[ua] = os
			}
			h.osH[os]++
			if pageBool {
				h.osP[os]++
			}
			if os == "Unknown" {
				h.unknownRefL[strings.NewReplacer("+", "_", " ", "_").Replace(ua)] = timeRec
			}
		}
	} else {
		h.browserH["Unknown"]++
		h.osH["Unknown"]++
		if pageBool {
			h.browserP["Unknown"]++
			h.osP["Unknown"]++
		}
	}
	u.countReferer(h, field, pageBool)
}

func (u *updater) countBrowser(h *month, ua string, timeRec int64, pageBool bool) {
	b := u.tmpBrowser[ua]
	if b == "" {
		sm := func(re *regexp.Regexp) []string { return re.FindStringSubmatch(ua) }
		if m := sm(reEdge); m != nil {
			b = "edge" + m[1]
		} else if m := sm(reOpera); m != nil {
			if m[1] != "" {
				b = "opera" + m[1]
			} else {
				b = "opera" + m[2]
			}
		} else if m := sm(reFirefox); m != nil && !reNotFF.MatchString(ua) {
			b = "firefox" + m[1]
		} else if m := sm(reChrome); m != nil {
			b = "chrome" + m[1]
		} else if m := sm(reSafari); m != nil && !reNotSafari.MatchString(ua) {
			ver := u.lib.safariBuild[m[1]]
			if vm := sm(reSafariVer); vm != nil {
				ver = vm[1]
			}
			b = "safari" + ver
		} else if m := sm(reKonqueror); m != nil {
			b = "konqueror" + m[1]
		} else if m := sm(reSVN); m != nil {
			b = "svn" + m[1]
		} else if m := sm(reMSIE); m != nil && !reNotIE.MatchString(ua) {
			b = "msie" + m[2]
		} else if m := sm(reMSIE11); m != nil && !reNotIE.MatchString(ua) {
			b = "msie" + m[2]
		} else if m := sm(reNetscape); m != nil {
			b = "netscape" + m[1]
		} else if m := sm(reMozilla); m != nil && !reNotNS.MatchString(ua) {
			b = "netscape" + m[2]
		} else {
			for _, r := range u.lib.browsers {
				if r.re.MatchString(ua) {
					b = r.id
					break
				}
			}
		}
		if b == "" {
			b = "Unknown"
		}
		u.tmpBrowser[ua] = b
	}
	h.browserH[b]++
	if pageBool {
		h.browserP[b]++
	}
	if b == "Unknown" {
		h.unknownRefBrowserL[strings.NewReplacer("+", "_", " ", "_").Replace(ua)] = timeRec
	}
}

func (u *updater) countReferer(h *month, field []string, pageBool bool) {
	c, f := u.c, u.lf
	found := false
	if f.referer >= 0 && c.LevelForReferer > 0 && truthy(field[f.referer]) {
		ref := field[f.referer]
		if ref == "-" || ref == "bookmarks" {
			if pageBool {
				h.fromP[0]++
			}
			h.fromH[0]++
			found = true
		} else if rm := reReferer.FindStringSubmatch(ref); rm != nil && reHTTP.MatchString(rm[1]) {
			server := rm[2]
			if rm[3] != "" && rm[3] != ":80" {
				server += rm[3]
			}
			tmp, cached := u.tmpRefServer[server]
			if !cached { // "" caches "other site" (awstats.pl searches again each time, with the same result)
				if u.regLocal.MatchString(server) || matchAny(c.HostAliases, server) {
					tmp = "="
				} else if c.LevelForSearchEngines > 0 {
					for _, se := range u.lib.searchEngines {
						if se.re.MatchString(server) {
							if not := u.lib.seNotKeys[se.id]; not == nil || !not.MatchString(server) {
								tmp = u.lib.seID[se.id]
							}
							break
						}
					}
				}
				u.tmpRefServer[server] = tmp
			}
			switch {
			case tmp == "=":
				if pageBool {
					h.fromP[4]++
				}
				h.fromH[4]++
				found = true
			case tmp != "":
				if pageBool {
					h.fromP[2]++
					h.seP[tmp]++
				}
				h.fromH[2]++
				h.seH[tmp]++
				found = true
				if pageBool && c.LevelForKeywords > 0 {
					u.countKeyphrase(h, tmp, ref)
				}
			default:
				if pageBool {
					h.fromP[3]++
				}
				h.fromH[3]++
				key := ref
				if !c.URLReferrerWithQuery {
					if qm := u.regRefNoQuery.FindStringSubmatch(ref); qm != nil {
						key = qm[1]
					}
				}
				if pageBool {
					h.pagerefsP[key]++
				}
				h.pagerefsH[key]++
				found = true
			}
		}
	}
	if !found {
		if pageBool {
			h.fromP[1]++
		}
		h.fromH[1]++
	}
}

func (u *updater) countKeyphrase(h *month, se, ref string) {
	c := u.c
	base, q, _ := strings.Cut(ref, "?")
	if truthy(q) {
		if c.KeyWordsNotSensitive {
			q = strings.ToLower(q)
		}
		if known := u.lib.seKnownURL[se]; known != nil {
			for _, p := range strings.Split(q, "&") {
				if loc := known.FindStringIndex(p); loc != nil {
					p = p[loc[1]:]
					p = reCacheRel.ReplaceAllString(p, "")
					p = strings.Trim(changeWordSeparatorsIntoSpace(p), " ")
					p = squeezeSpacesToPlus(p)
					if len(p) > 0 && len(p) < 80 {
						h.keyphrases[p]++
					}
					break
				}
			}
		} else if c.LevelForKeywords >= 2 {
			for _, p := range strings.Split(q, "&") {
				if matchAny(u.lib.wordsToClean, p) {
					continue
				}
				p = reUpToEq.ReplaceAllString(p, "")
				p = reCacheRel.ReplaceAllString(p, "")
				p = strings.Trim(changeWordSeparatorsIntoSpace(p), " ")
				p = squeezeSpacesToPlus(p)
				if len(p) > 2 {
					h.keyphrases[p]++
					break
				}
			}
		}
	} else if u.lib.seNoQuery[se] {
		if tail := u.lib.seKnownTail[se]; tail != nil {
			if m := tail.FindStringSubmatch(base); m != nil {
				p := squeezeSpacesToPlus(changeWordSeparatorsIntoSpace(m[1]))
				if len(p) > 0 {
					h.keyphrases[p]++
				}
			}
		}
	}
}
