package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// dataFile is a parsed AWStats text data file: section name -> rows of fields.
type dataFile struct {
	version  string
	sections map[string][][]string
}

func readDataFile(path string) (*dataFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	df := &dataFile{sections: map[string][][]string{}}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 16*1024*1024)
	first := true
	var name string
	var rows [][]string
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		if first {
			first = false
			if !strings.HasPrefix(line, "AWSTATS DATA FILE ") {
				return nil, fmt.Errorf("%s is not an AWStats text data file", path)
			}
			df.version = strings.TrimPrefix(line, "AWSTATS DATA FILE ")
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "#") {
			continue
		}
		if name == "" {
			if strings.HasPrefix(fields[0], "BEGIN_") {
				name, rows = strings.TrimPrefix(fields[0], "BEGIN_"), nil
			}
			continue
		}
		if fields[0] == "END_"+name {
			df.sections[name] = rows
			name = ""
			continue
		}
		rows = append(rows, fields)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if name != "" {
		return nil, fmt.Errorf("history file %q is corrupted (end of section %s not found)", path, name)
	}
	return df, nil
}

func fld(r []string, i int) string {
	if i < len(r) {
		return r[i]
	}
	return ""
}

func fi(r []string, i int) int64 { return pint(fld(r, i)) }

func minButNoZero(a, b int64) int64 {
	if a != 0 && (a < b || b == 0) {
		return a
	}
	return b
}

// merge adds a data file written earlier for this month to the records of this run, like
// Read_History_With_TmpUpdate with withupdate=1.
func (u *updater) merge(h *month, df *dataFile) {
	s := df.sections
	for _, r := range s["GENERAL"] {
		switch r[0] {
		case "LastLine":
			if v := fi(r, 1); u.lastLine == 0 || u.lastLine < v {
				u.lastLine = v
			}
		case "FirstTime":
			if v := fi(r, 1); h.firstTime == 0 || h.firstTime > v {
				h.firstTime = v
			}
		case "LastTime":
			if v := fi(r, 1); h.lastTime == 0 || h.lastTime < v {
				h.lastTime = v
			}
		}
	}
	for _, r := range s["MISC"] {
		addIf(h.miscP, r[0], fi(r, 1))
		addIf(h.miscH, r[0], fi(r, 2))
		addIf(h.miscK, r[0], fi(r, 3))
	}
	for _, r := range s["CLUSTER"] {
		addIf(h.clusterP, r[0], fi(r, 1))
		addIf(h.clusterH, r[0], fi(r, 2))
		addIf(h.clusterK, r[0], fi(r, 3))
	}
	for _, r := range s["TIME"] {
		if i := int(fi(r, 0)); i >= 0 && i < 24 {
			h.timeP[i] += fi(r, 1)
			h.timeH[i] += fi(r, 2)
			h.timeK[i] += fi(r, 3)
			h.timeNvP[i] += fi(r, 4)
			h.timeNvH[i] += fi(r, 5)
			h.timeNvK[i] += fi(r, 6)
		}
	}
	for _, r := range s["ORIGIN"] {
		if strings.HasPrefix(r[0], "From") {
			if i := int(pint(r[0][4:])); i >= 0 && i < 6 {
				h.fromP[i] += fi(r, 1)
				h.fromH[i] += fi(r, 2)
			}
		}
	}
	for _, r := range s["DAY"] {
		addIf(h.dayPages, r[0], fi(r, 1))
		h.dayHits[r[0]] += fi(r, 2)
		addIf(h.dayBytes, r[0], fi(r, 3))
		addIf(h.dayVisits, r[0], fi(r, 4))
	}
	for _, r := range s["VISITOR"] {
		host := r[0]
		if e := h.waitE[host]; e != "" {
			tl, ts := fi(r, 4), fi(r, 5)
			newS, newL := h.waitS[host], h.waitL[host]
			if newS == 0 {
				newS = h.hostS[host]
			}
			if newL == 0 {
				newL = h.hostL[host]
			}
			if newS > tl+visitTimeout { // the first visit of this run is a new visit
				if p := fld(r, 6); p != "" {
					h.urlX[p]++
				}
				h.urlE[e]++
				if d := strconv.FormatInt(newS, 10); len(d) >= 8 {
					h.dayVisits[d[:8]]++
				}
				if ts != 0 && tl != 0 {
					h.session[sessionRange(ts, tl)]++
				}
				if h.waitS[host] != 0 {
					h.session[sessionRange(newS, newL)]++
				}
			} else { // it continues the last visit of the data file
				if h.waitS[host] != 0 {
					h.session[sessionRange(minButNoZero(ts, newS), max(tl, newL))]++
				} else {
					if tl > newL {
						h.hostL[host], h.hostU[host] = tl, fld(r, 6)
					}
					if ts < newS {
						h.hostS[host] = ts
					}
				}
			}
			delete(h.waitE, host)
			delete(h.waitL, host)
			delete(h.waitS, host)
			delete(h.waitU, host)
		}
		addIf(h.hostP, host, fi(r, 1))
		addIf(h.hostH, host, fi(r, 2))
		addIf(h.hostK, host, fi(r, 3))
		if fi(r, 4) != 0 && h.hostL[host] == 0 {
			h.hostL[host] = fi(r, 4)
			if fi(r, 5) != 0 && h.hostS[host] == 0 {
				h.hostS[host] = fi(r, 5)
			}
			if p := fld(r, 6); p != "" && h.hostU[host] == "" {
				h.hostU[host] = p
			}
		}
	}
	for _, r := range s["LOGIN"] {
		addIf(h.loginP, r[0], fi(r, 1))
		addIf(h.loginH, r[0], fi(r, 2))
		addIf(h.loginK, r[0], fi(r, 3))
		if h.loginL[r[0]] == 0 && fi(r, 4) != 0 {
			h.loginL[r[0]] = fi(r, 4)
		}
	}
	for _, r := range s["DOMAIN"] {
		addIf(h.domP, r[0], fi(r, 1))
		addIf(h.domH, r[0], fi(r, 2))
		addIf(h.domK, r[0], fi(r, 3))
	}
	for _, r := range s["SESSION"] {
		h.session[r[0]] += fi(r, 1)
	}
	for _, r := range s["OS"] {
		h.osH[r[0]] += fi(r, 1)
		h.osP[r[0]] += fi(r, 2)
	}
	for _, r := range s["BROWSER"] {
		h.browserH[r[0]] += fi(r, 1)
		h.browserP[r[0]] += fi(r, 2)
	}
	for name, dst := range map[string]map[string]int64{"UNKNOWNREFERER": h.unknownRefL, "UNKNOWNREFERERBROWSER": h.unknownRefBrowserL} {
		for _, r := range s[name] {
			if dst[r[0]] == 0 {
				dst[r[0]] = fi(r, 1)
			}
		}
	}
	for _, r := range s["SCREENSIZE"] {
		addIf(h.screensize, r[0], fi(r, 1))
	}
	for _, r := range s["ROBOT"] {
		addIf(h.robotH, r[0], fi(r, 1))
		h.robotK[r[0]] += fi(r, 2)
		if h.robotL[r[0]] == 0 {
			h.robotL[r[0]] = fi(r, 3)
		}
		addIf(h.robotR, r[0], fi(r, 4))
	}
	for _, x := range []struct {
		name    string
		h, k, l map[string]int64
	}{{"WORMS", h.wormH, h.wormK, h.wormL}, {"EMAILSENDER", h.emailsH, h.emailsK, h.emailsL}, {"EMAILRECEIVER", h.emailrH, h.emailrK, h.emailrL}} {
		for _, r := range s[x.name] {
			addIf(x.h, r[0], fi(r, 1))
			x.k[r[0]] += fi(r, 2)
			if x.l[r[0]] == 0 {
				x.l[r[0]] = fi(r, 3)
			}
		}
	}
	for _, r := range s["SIDER"] {
		addIf(h.urlP, r[0], fi(r, 1))
		addIf(h.urlK, r[0], fi(r, 2))
		addIf(h.urlE, r[0], fi(r, 3))
		addIf(h.urlX, r[0], fi(r, 4))
	}
	for _, r := range s["FILETYPES"] {
		addIf(h.ftH, r[0], fi(r, 1))
		addIf(h.ftK, r[0], fi(r, 2))
		addIf(h.ftGzIn, r[0], fi(r, 3))
		addIf(h.ftGzOut, r[0], fi(r, 4))
	}
	for _, r := range s["DOWNLOADS"] {
		d := h.download(r[0])
		d[0] += fi(r, 1)
		d[1] += fi(r, 2)
		d[2] += fi(r, 3)
	}
	for _, r := range s["SEREFERRALS"] {
		addIf(h.seP, r[0], fi(r, 1))
		addIf(h.seH, r[0], fi(r, 2))
	}
	for _, r := range s["PAGEREFS"] {
		addIf(h.pagerefsP, r[0], fi(r, 1))
		addIf(h.pagerefsH, r[0], fi(r, 2))
	}
	for _, r := range s["SEARCHWORDS"] {
		addIf(h.keyphrases, r[0], fi(r, 1))
	}
	for _, r := range s["ERRORS"] {
		addIf(h.errorsH, r[0], fi(r, 1))
		addIf(h.errorsK, r[0], fi(r, 2))
	}
	for code, hits := range h.siderH {
		for _, r := range s["SIDER_"+code] {
			addIf(hits, r[0], fi(r, 1))
			if strings.ContainsAny(u.c.ShowHTTPErrorsPageDetail, "Rr") && fld(r, 2) != "" {
				h.refererH[code][r[0]] = fld(r, 2)
			}
		}
	}
	for _, r := range s["FILESIZE"] {
		h.filesize[r[0]] += fi(r, 1)
	}
	for _, r := range s["REQUESTTIME"] {
		h.requesttime[r[0]] += fi(r, 1)
	}
	for name := range s {
		if strings.HasPrefix(name, "EXTRA_") || strings.HasPrefix(name, "PLUGIN_") {
			fmt.Fprintf(os.Stderr, "Warning: section %s of the existing data file is not supported and is dropped.\n", name)
		}
	}
}

// finishWaitingVisits counts the first visit of hosts that were not in the data file.
func finishWaitingVisits(h *month) {
	for host, e := range h.waitE {
		newS, newL := h.waitS[host], h.waitL[host]
		if newS == 0 {
			newS = h.hostS[host]
		}
		if newL == 0 {
			newL = h.hostL[host]
		}
		h.urlE[e]++
		if d := strconv.FormatInt(newS, 10); len(d) >= 8 {
			h.dayVisits[d[:8]]++
		}
		if h.waitS[host] != 0 {
			h.session[sessionRange(newS, newL)]++
		}
	}
	h.waitE, h.waitL, h.waitS, h.waitU = map[string]string{}, m64(), m64(), map[string]string{}
}

// sortedKeys returns the keys of h, highest value of order first (ties by key), like BuildKeyList
// puts the top entries first; the rest of the order does not matter to readers.
func sortedKeys[V any](h map[string]V, order map[string]int64) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if order != nil && order[keys[i]] != order[keys[j]] {
			return order[keys[i]] > order[keys[j]]
		}
		return keys[i] < keys[j]
	})
	return keys
}

func encodeForHisto(s string) string { return reWhite.ReplaceAllString(s, "%20") }

var (
	reDoubleSlash = regexp.MustCompile(`([^:])//`)
	reRefSlash    = regexp.MustCompile(`(?i)^http(s|)://([^/]+)/$`)
)

var mapOrder = []string{"GENERAL", "TIME", "VISITOR", "DAY", "DOMAIN", "LOGIN", "ROBOT", "WORMS", "EMAILSENDER",
	"EMAILRECEIVER", "SESSION", "FILESIZE", "REQUESTTIME", "SIDER", "FILETYPES", "DOWNLOADS", "OS", "BROWSER",
	"SCREENSIZE", "UNKNOWNREFERER", "UNKNOWNREFERERBROWSER", "ORIGIN", "SEREFERRALS", "PAGEREFS", "SEARCHWORDS",
	"KEYWORDS", "MISC", "ERRORS", "CLUSTER"}

func (u *updater) dataFileName(year, mon string) string {
	name := "awstats" + mon + year
	if u.site != "" {
		name += "." + u.site
	}
	return filepath.Join(u.c.DirData, name+".txt")
}

// saveMonth merges the current month with its data file and writes the file again.
func (u *updater) saveMonth(lastLineNb, lastLineOffset, lastLineChecksum int64) error {
	h, c := u.cur, u.c
	path := u.dataFileName(h.year, h.mon)
	if _, err := os.Stat(path); err == nil {
		df, err := readDataFile(path)
		if err != nil {
			return err
		}
		u.merge(h, df)
	}
	finishWaitingVisits(h)

	var body bytes.Buffer
	pos := map[string]int{}
	section := func(name string, comments ...string) {
		body.WriteString("\n")
		for _, cm := range comments {
			body.WriteString(cm + "\n")
		}
		pos[name] = body.Len()
	}
	w := func(format string, a ...any) { fmt.Fprintf(&body, format, a...) }
	top := func(key string) string { return fmt.Sprintf("# The %d first", c.MaxNbOf[key]) }

	// VISITOR first: saving it closes finished visits (session and exit page counts).
	var vis bytes.Buffer
	known := 0
	for _, k := range sortedKeys(h.hostH, h.hostP) {
		if !reIPv4.MatchString(k) && !reIPv6.MatchString(k) {
			known++
		}
		p, hh, kb, tl, ts, lp := h.hostP[k], h.hostH[k], h.hostK[k], h.hostL[k], h.hostS[k], h.hostU[k]
		switch {
		case tl != 0 && ts != 0 && lp != "" && tl+visitTimeout < u.lastLine:
			h.session[sessionRange(ts, tl)]++
			h.urlX[lp]++
			delete(h.hostS, k)
			delete(h.hostU, k)
			fmt.Fprintf(&vis, "%s %d %d %d %d\n", k, p, hh, kb, tl)
		case tl != 0 && ts != 0 && lp != "":
			fmt.Fprintf(&vis, "%s %d %d %d %d %d %s\n", k, p, hh, kb, tl, ts, lp)
		default:
			l := ""
			if tl != 0 {
				l = strconv.FormatInt(tl, 10)
			}
			fmt.Fprintf(&vis, "%s %d %d %d %s\n", k, p, hh, kb, l)
		}
	}
	unique := len(h.hostP)
	monthVisits := int64(0)
	for _, d := range sortedKeys(h.dayHits, nil) {
		if strings.HasPrefix(d, h.year+h.mon) {
			monthVisits += h.dayVisits[d]
		}
	}

	lastLine := u.lastLine
	if lastLine <= 0 {
		lastLine = h.lastTime
	}
	section("GENERAL",
		"# LastLine    = Date of last record processed - Last record line number in last log - Last record offset in last log - Last record signature value",
		"# FirstTime   = Date of first visit for history file",
		"# LastTime    = Date of last visit for history file",
		"# LastUpdate  = Date of last update - Nb of parsed records - Nb of parsed old records - Nb of parsed new records - Nb of parsed corrupted - Nb of parsed dropped",
		"# TotalVisits = Number of visits",
		"# TotalUnique = Number of unique visitors",
		"# MonthHostsKnown   = Number of hosts known",
		"# MonthHostsUnKnown = Number of hosts unknown")
	opt := func(v int64) string {
		if v == 0 {
			return ""
		}
		return strconv.FormatInt(v, 10)
	}
	w("BEGIN_GENERAL 8\n")
	w("LastLine %d %d %d %d\n", lastLine, lastLineNb, lastLineOffset, lastLineChecksum)
	w("FirstTime %s\n", opt(h.firstTime))
	w("LastTime %s\n", opt(h.lastTime))
	w("LastUpdate %d %d %d %d %d %d\n", u.now, u.parsed, u.oldLines, u.newLines, u.corrupted, u.dropped)
	w("TotalVisits %-20d\n", monthVisits)
	w("TotalUnique %-20d\n", unique)
	w("MonthHostsKnown %-20d\n", known)
	w("MonthHostsUnknown %-20d\n", len(h.hostH)-known)
	w("END_GENERAL\n")

	section("MISC", "# Misc ID - Pages - Hits - Bandwidth")
	w("BEGIN_MISC %d\n", len(miscList))
	for _, k := range miscList {
		w("%s %d %d %d\n", k, h.miscP[k], h.miscH[k], h.miscK[k])
	}
	w("END_MISC\n")

	section("TIME", "# Hour - Pages - Hits - Bandwidth - Not viewed Pages - Not viewed Hits - Not viewed Bandwidth")
	w("BEGIN_TIME 24\n")
	for i := 0; i < 24; i++ {
		w("%d %d %d %d %d %d %d\n", i, h.timeP[i], h.timeH[i], h.timeK[i], h.timeNvP[i], h.timeNvH[i], h.timeNvK[i])
	}
	w("END_TIME\n")

	section("VISITOR",
		"# Host - Pages - Hits - Bandwidth - Last visit date - [Start date of last visit] - [Last page of last visit]",
		"# [Start date of last visit] and [Last page of last visit] are saved only if session is not finished",
		top("HostsShown")+" Hits must be first (order not required for others)")
	w("BEGIN_VISITOR %d\n", len(h.hostH))
	body.Write(vis.Bytes())
	w("END_VISITOR\n")

	section("DAY", "# Date - Pages - Hits - Bandwidth - Visits")
	w("BEGIN_DAY %d\n", len(h.dayHits))
	for _, d := range sortedKeys(h.dayHits, nil) {
		if strings.HasPrefix(d, h.year+h.mon) {
			w("%s %d %d %d %d\n", d, h.dayPages[d], h.dayHits[d], h.dayBytes[d], h.dayVisits[d])
		}
	}
	w("END_DAY\n")

	section("DOMAIN", "# Domain - Pages - Hits - Bandwidth", top("Domain")+" Pages must be first (order not required for others)")
	w("BEGIN_DOMAIN %d\n", len(h.domH))
	for _, k := range sortedKeys(h.domH, h.domP) {
		w("%s %d %d %d\n", k, h.domP[k], h.domH[k], h.domK[k])
	}
	w("END_DOMAIN\n")

	section("CLUSTER", "# Cluster ID - Pages - Hits - Bandwidth")
	w("BEGIN_CLUSTER %d\n", len(h.clusterH))
	for _, k := range sortedKeys(h.clusterH, nil) {
		w("%s %d %d %d\n", k, h.clusterP[k], h.clusterH[k], h.clusterK[k])
	}
	w("END_CLUSTER\n")

	section("LOGIN", "# Login - Pages - Hits - Bandwidth - Last visit", top("LoginShown")+" Pages must be first (order not required for others)")
	w("BEGIN_LOGIN %d\n", len(h.loginH))
	for _, k := range sortedKeys(h.loginH, h.loginP) {
		w("%s %d %d %d %s\n", k, h.loginP[k], h.loginH[k], h.loginK[k], opt(h.loginL[k]))
	}
	w("END_LOGIN\n")

	section("ROBOT", "# Robot ID - Hits - Bandwidth - Last visit - Hits on robots.txt", top("RobotShown")+" Hits must be first (order not required for others)")
	w("BEGIN_ROBOT %d\n", len(h.robotH))
	for _, k := range sortedKeys(h.robotH, h.robotH) {
		w("%s %d %d %s %d\n", k, h.robotH[k], h.robotK[k], opt(h.robotL[k]), h.robotR[k])
	}
	w("END_ROBOT\n")

	section("WORMS", "# Worm ID - Hits - Bandwidth - Last visit", top("WormsShown")+" Hits must be first (order not required for others)")
	w("BEGIN_WORMS %d\n", len(h.wormH))
	for _, k := range sortedKeys(h.wormH, h.wormH) {
		w("%s %d %d %s\n", k, h.wormH[k], h.wormK[k], opt(h.wormL[k]))
	}
	w("END_WORMS\n")

	section("EMAILSENDER", "# EMail - Hits - Bandwidth - Last visit", top("EMailsShown")+" Hits must be first (order not required for others)")
	w("BEGIN_EMAILSENDER %d\n", len(h.emailsH))
	for _, k := range sortedKeys(h.emailsH, h.emailsH) {
		w("%s %d %d %s\n", k, h.emailsH[k], h.emailsK[k], opt(h.emailsL[k]))
	}
	w("END_EMAILSENDER\n")

	section("EMAILRECEIVER", "# EMail - Hits - Bandwidth - Last visit", top("EMailsShown")+" hits must be first (order not required for others)")
	w("BEGIN_EMAILRECEIVER %d\n", len(h.emailrH))
	for _, k := range sortedKeys(h.emailrH, h.emailrH) {
		w("%s %d %d %s\n", k, h.emailrH[k], h.emailrK[k], opt(h.emailrL[k]))
	}
	w("END_EMAILRECEIVER\n")

	section("SESSION", "# Session range - Number of visits")
	w("BEGIN_SESSION %d\n", len(h.session))
	for _, k := range sortedKeys(h.session, nil) {
		w("%s %d\n", k, h.session[k])
	}
	w("END_SESSION\n")

	section("SIDER", "# URL - Pages - Bandwidth - Entry - Exit", top("PageShown")+" Pages must be first (order not required for others)")
	w("BEGIN_SIDER %d\n", len(h.urlP))
	for _, k := range sortedKeys(h.urlP, h.urlP) {
		w("%s %d %d %d %d\n", encodeForHisto(reDoubleSlash.ReplaceAllString(k, "$1/")), h.urlP[k], h.urlK[k], h.urlE[k], h.urlX[k])
	}
	w("END_SIDER\n")

	section("FILETYPES", "# Files type - Hits - Bandwidth - Bandwidth without compression - Bandwidth after compression")
	w("BEGIN_FILETYPES %d\n", len(h.ftH))
	for _, k := range sortedKeys(h.ftH, nil) {
		w("%s %d %d %d %d\n", k, h.ftH[k], h.ftK[k], h.ftGzIn[k], h.ftGzOut[k])
	}
	w("END_FILETYPES\n")

	section("DOWNLOADS", "# Downloads - Hits - Bandwidth")
	w("BEGIN_DOWNLOADS %d\n", len(h.downloads))
	dl := make([]string, 0, len(h.downloads))
	for k := range h.downloads {
		dl = append(dl, k)
	}
	sort.SliceStable(dl, func(i, j int) bool {
		if h.downloads[dl[i]][0] != h.downloads[dl[j]][0] {
			return h.downloads[dl[i]][0] > h.downloads[dl[j]][0]
		}
		return dl[i] < dl[j]
	})
	for _, k := range dl {
		d := h.downloads[k]
		w("%s %d %d %d\n", encodeForHisto(k), d[0], d[1], d[2])
	}
	w("END_DOWNLOADS\n")

	section("OS", "# OS ID - Hits")
	w("BEGIN_OS ID - Hits - Pages %d\n", len(h.osH))
	for _, k := range sortedKeys(h.osH, nil) {
		w("%s %d %d\n", k, h.osH[k], h.osP[k])
	}
	w("END_OS\n")

	section("BROWSER", "# Browser ID - Hits - Pages")
	w("BEGIN_BROWSER %d\n", len(h.browserH))
	for _, k := range sortedKeys(h.browserH, nil) {
		w("%s %d %d\n", k, h.browserH[k], h.browserP[k])
	}
	w("END_BROWSER\n")

	section("SCREENSIZE", "# Screen size - Hits")
	w("BEGIN_SCREENSIZE %d\n", len(h.screensize))
	for _, k := range sortedKeys(h.screensize, nil) {
		w("%s %d\n", k, h.screensize[k])
	}
	w("END_SCREENSIZE\n")

	section("UNKNOWNREFERER", "# Unknown referer OS - Last visit date")
	w("BEGIN_UNKNOWNREFERER %d\n", len(h.unknownRefL))
	for _, k := range sortedKeys(h.unknownRefL, nil) {
		w("%s %d\n", encodeForHisto(k), h.unknownRefL[k])
	}
	w("END_UNKNOWNREFERER\n")

	section("UNKNOWNREFERERBROWSER", "# Unknown referer Browser - Last visit date")
	w("BEGIN_UNKNOWNREFERERBROWSER %d\n", len(h.unknownRefBrowserL))
	for _, k := range sortedKeys(h.unknownRefBrowserL, nil) {
		w("%s %d\n", encodeForHisto(k), h.unknownRefBrowserL[k])
	}
	w("END_UNKNOWNREFERERBROWSER\n")

	section("ORIGIN", "# Origin - Pages - Hits ")
	w("BEGIN_ORIGIN 6\n")
	for i := 0; i < 6; i++ {
		w("From%d %d %d\n", i, h.fromP[i], h.fromH[i])
	}
	w("END_ORIGIN\n")

	section("SEREFERRALS", "# Search engine referers ID - Pages - Hits")
	w("BEGIN_SEREFERRALS %d\n", len(h.seH))
	for _, k := range sortedKeys(h.seH, nil) {
		w("%s %d %d\n", k, h.seP[k], h.seH[k])
	}
	w("END_SEREFERRALS\n")

	section("PAGEREFS", "# External page referers - Pages - Hits", top("RefererShown")+" Pages must be first (order not required for others)")
	w("BEGIN_PAGEREFS %d\n", len(h.pagerefsH))
	for _, k := range sortedKeys(h.pagerefsH, h.pagerefsP) {
		w("%s %d %d\n", encodeForHisto(reRefSlash.ReplaceAllString(k, "http$1://$2")), h.pagerefsP[k], h.pagerefsH[k])
	}
	w("END_PAGEREFS\n")

	section("SEARCHWORDS", "# Search keyphrases - Number of search", top("KeyphrasesShown")+" number of search must be first (order not required for others)")
	w("BEGIN_SEARCHWORDS %d\n", len(h.keyphrases))
	h.keywords = m64()
	for _, k := range sortedKeys(h.keyphrases, h.keyphrases) {
		w("%s %d\n", encodeForHisto(squeezeSpacesToPlus(k)), h.keyphrases[k])
		words := strings.Split(k, "+")
		for len(words) > 0 && words[len(words)-1] == "" { // Perl split drops trailing empty fields
			words = words[:len(words)-1]
		}
		for _, wd := range words {
			h.keywords[wd] += h.keyphrases[k]
		}
	}
	w("END_SEARCHWORDS\n")

	section("KEYWORDS", "# Search keywords - Number of search", top("KeywordsShown")+" number of search must be first (order not required for others)")
	w("BEGIN_KEYWORDS %d\n", len(h.keywords))
	for _, k := range sortedKeys(h.keywords, h.keywords) {
		w("%s %d\n", encodeForHisto(k), h.keywords[k])
	}
	w("END_KEYWORDS\n")

	section("ERRORS", "# Errors - Hits - Bandwidth")
	w("BEGIN_ERRORS %d\n", len(h.errorsH))
	for _, k := range sortedKeys(h.errorsH, nil) {
		w("%s %d %d\n", k, h.errorsH[k], h.errorsK[k])
	}
	w("END_ERRORS\n")

	section("FILESIZE", "# Payload Range - Payload Frequency")
	w("BEGIN_FILESIZE %d\n", len(h.filesize))
	for _, k := range sortedKeys(h.filesize, nil) {
		w("%s %d\n", k, h.filesize[k])
	}
	w("END_FILESIZE\n")

	section("REQUESTTIME", "# Request Time Range - Request Time Frequency")
	w("BEGIN_REQUESTTIME %d\n", len(h.requesttime))
	for _, k := range sortedKeys(h.requesttime, nil) {
		w("%s %d\n", k, h.requesttime[k])
	}
	w("END_REQUESTTIME\n")

	withR := strings.ContainsAny(c.ShowHTTPErrorsPageDetail, "Rr")
	withH := strings.ContainsAny(c.ShowHTTPErrorsPageDetail, "Hh")
	codes := sortedKeys(h.siderH, nil)
	for _, code := range codes {
		title := "# URL with " + code + " errors - Hits"
		if withR {
			title += " - Last URL referrer"
		}
		if withH {
			title += " - Host"
		}
		section("SIDER_"+code, title)
		w("BEGIN_SIDER_%s %d\n", code, len(h.siderH[code]))
		for _, k := range sortedKeys(h.siderH[code], nil) {
			w("%s %d", encodeForHisto(k), h.siderH[code][k])
			if withR {
				w(" %s", encodeForHisto(h.refererH[code][k]))
			}
			if withH {
				w(" %s", encodeForHisto(h.errHostH[code][k]))
			}
			w("\n")
		}
		w("END_SIDER_%s\n", code)
	}

	// Header and MAP. Every POS value is padded to 20 characters, so the MAP length is known
	// before the offsets are.
	var head bytes.Buffer
	fmt.Fprintf(&head, "AWSTATS DATA FILE %s\n", dataFileVersion())
	fmt.Fprintf(&head, "# If you remove this file, all statistics for date %s%s will be lost/reset.\n", h.year, h.mon)
	fmt.Fprintf(&head, "# Last config file used to build this data file was %s.\n", c.file)
	head.WriteString("\n")
	head.WriteString("# Position (offset in bytes) in this file for beginning of each section for\n")
	head.WriteString("# direct I/O access. If you made changes somewhere in this file, you should\n")
	head.WriteString("# also remove completely the MAP section (AWStats will rewrite it at next\n")
	head.WriteString("# update).\n")
	fmt.Fprintf(&head, "BEGIN_MAP %d\n", 27+len(codes)) // awstats.pl writes 27 + codes although it lists 29 + codes
	names := append(append([]string{}, mapOrder...), func() []string {
		var out []string
		for _, cd := range codes {
			out = append(out, "SIDER_"+cd)
		}
		return out
	}()...)
	mapLen := head.Len() + len("END_MAP\n")
	for _, n := range names {
		mapLen += len("POS_"+n+" ") + 20 + 1
	}
	for _, n := range names {
		fmt.Fprintf(&head, "POS_%s %-20d\n", n, mapLen+pos[n])
	}
	head.WriteString("END_MAP\n")

	if err := os.MkdirAll(c.DirData, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp." + strconv.Itoa(os.Getpid())
	if err := os.WriteFile(tmp, append(head.Bytes(), body.Bytes()...), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	u.saved = append(u.saved, path)
	return nil
}
