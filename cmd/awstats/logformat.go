package main

import (
	"fmt"
	"regexp"
	"strings"
)

// logFormat is the compiled line regex and the capture position of each field (-1 = absent),
// as built by DefinePerlParsingFormat in awstats.pl.
type logFormat struct {
	re                                                  *regexp.Regexp
	vh, host, hostr, logname, date, method, url, code   int
	size, referer, agent, query, gzipin, gzipout, ratio int
	ratioIsGzip                                         bool
	cluster, rqtime, nfields                            int
}

func newLogFormat(format, sep string) (*logFormat, error) {
	f := &logFormat{vh: -1, host: -1, hostr: -1, logname: -1, date: -1, method: -1, url: -1, code: -1,
		size: -1, referer: -1, agent: -1, query: -1, gzipin: -1, gzipout: -1, ratio: -1, cluster: -1, rqtime: -1}
	var pat string
	switch format {
	case "1", "6":
		pat = `([^ ]+) [^ ]+ ([^\/\[]+) \[([^ ]+) [^ ]+\] \"([^ ]+) ([^ ]+)(?: [^\"]+|)\" ([\d|-]+) ([\d|-]+) \"(.*?)\" \"([^\"]*)\"`
		f.host, f.logname, f.date, f.method, f.url, f.code, f.size, f.referer, f.agent = 0, 1, 2, 3, 4, 5, 6, 7, 8
	case "4":
		pat = `([^ ]+) [^ ]+ (.+) \[([^ ]+) [^ ]+\] \"([^ ]+) (.+) [^\"]+\" ([\d|-]+) ([\d|-]+)`
		f.host, f.logname, f.date, f.method, f.url, f.code, f.size = 0, 1, 2, 3, 4, 5, 6
	case "2", "3", "5", "json":
		return nil, fmt.Errorf("LogFormat=%s is not supported yet; use 1, 4 or a personalized format string", format)
	default:
		var err error
		if pat, err = f.personalized(format, sep); err != nil {
			return nil, err
		}
	}
	for _, req := range []struct {
		pos int
		tag string
	}{{f.host, "%host"}, {f.date, "%time1 or %time2"}, {f.method, "%methodurl or %method"}, {f.url, "%methodurl or %url"}, {f.code, "%code"}} {
		if req.pos < 0 {
			return nil, fmt.Errorf("your personalized LogFormat does not include all fields required by AWStats (add %s)", req.tag)
		}
	}
	re, err := regexp.Compile("^" + pat)
	if err != nil {
		return nil, fmt.Errorf("LogFormat %q: %w", format, err)
	}
	f.re = re
	return f, nil
}

func (f *logFormat) personalized(format, sep string) (string, error) {
	s := format
	for _, r := range []struct{ re, to string }{
		{`%vh`, `%virtualname`}, {`%v(\s)`, `%virtualname$1`}, {`%v$`, `%virtualname`},
		{`%h(\s)`, `%host$1`}, {`%h$`, `%host`}, {`%l(\s)`, `%other$1`}, {`%l$`, `%other`},
		{`"%u"`, `%lognamequot`}, {`%u(\s)`, `%logname$1`}, {`%u$`, `%logname`},
		{`%t(\s)`, `%time1$1`}, {`%t$`, `%time1`}, {`"%r"`, `%methodurl`}, {`%>s`, `%code`},
		{`%b(\s)`, `%bytesd$1`}, {`%b$`, `%bytesd`}, {`"%\{Referer}i"`, `%refererquot`},
		{`"%\{User-Agent}i"`, `%uaquot`}, {`%\{mod_gzip_input_size}n`, `%gzipin`},
		{`%\{mod_gzip_output_size}n`, `%gzipout`}, {`%\{mod_gzip_compression_ratio}n`, `%gzipratio`},
		{`\(%\{ratio}n\)`, `%deflateratio`}, {`cs-uri-query`, `%query`}, {`date\stime`, `%time2`},
		{`c-ip`, `%host`}, {`cs-username`, `%logname`}, {`cs-method`, `%method`}, {`cs-uri-stem`, `%url`},
		{`cs-uri`, `%url`}, {`sc-status`, `%code`}, {`sc-bytes`, `%bytesd`}, {`cs-version`, `%other`},
		{`cs\(User-Agent\)`, `%ua`}, {`c-agent`, `%ua`}, {`cs\(Referer\)`, `%referer`}, {`cs-referred`, `%referer`},
		{`sc-authenticated`, `%other`}, {`s-svcname`, `%other`}, {`s-computername`, `%other`},
		{`r-host`, `%virtualname`}, {`cs-host`, `%virtualname`}, {`r-ip`, `%other`}, {`r-port`, `%other`},
		{`time-taken`, `%other`}, {`cs-bytes`, `%other`}, {`cs-protocol`, `%other`}, {`cs-transport`, `%other`},
		{`s-operation`, `%method`}, {`cs-mime-type`, `%other`}, {`s-object-source`, `%other`},
		{`s-cache-info`, `%other`}, {`cluster-node`, `%cluster`}, {`s-sitename`, `%other`}, {`s-ip`, `%other`},
		{`s-port`, `%other`}, {`cs\(Cookie\)`, `%other`}, {`sc-substatus`, `%other`}, {`sc-win32-status`, `%other`},
		{`protocol`, `%protocolmms`}, {`c-status`, `%codemms`},
	} {
		s = regexp.MustCompile(r.re).ReplaceAllString(s, r.to)
	}
	ns := regexp.MustCompile(`[\*\+]`).ReplaceAllString(sep, "") // LogSeparatorWithoutStar
	field := "([^" + ns + "]+)"
	var pat strings.Builder
	i := 0
	next := func() int { i++; return i - 1 }
	for _, t := range strings.Fields(s) {
		if pat.Len() > 0 {
			pat.WriteString(sep)
		}
		if m := regexp.MustCompile(`^([^%]+)%`).FindStringSubmatch(t); m != nil {
			pat.WriteString(m[1])
		}
		has := func(tag string) bool { return strings.HasSuffix(t, tag) }
		switch {
		case has("%lognamequot"):
			f.logname = next()
			pat.WriteString(`\"?([^\"]*)\"?`)
		case has("%logname"):
			f.logname = next()
			pat.WriteString(`([^\/\[]+)`)
		case has("%time1"), has("%time1b"):
			f.date = next()
			next() // tz
			pat.WriteString(`\[([^` + ns + `]+)( [^` + ns + `]+)?\]`)
		case has("%time2"):
			f.date = next()
			pat.WriteString(`([^` + ns + `]+\s[^` + ns + `]+)`)
		case has("%time3"):
			f.date = next()
			pat.WriteString(`(?:\w\w\w )?(\w\w\w \s?\d+ \d\d:\d\d:\d\d(?: \d\d\d\d)?)`)
		case has("%time4"):
			f.date = next()
			pat.WriteString(`(\d+)`)
		case has("%time5"):
			f.date = next()
			next() // tz
			pat.WriteString(`([^` + ns + `]+T[^` + ns + `]+)(Z|[-+\.]\d\d[:\.\dZ]*)?`)
		case has("%time6"):
			f.date = next()
			pat.WriteString(`([^,]+,[^,]+)`)
		case has("%methodurl"):
			f.method, f.url = next(), next()
			pat.WriteString(`\"([^` + ns + `]+) ([^` + ns + `]+)(?: [^\"]+|)\"`)
		case has("%methodurlprot"):
			f.method, f.url = next(), next() // awstats.pl counts only two of the three captures here
			pat.WriteString(`\"([^` + ns + `]+) ([^\"]+) ([^\"]+)\"`)
		case has("%methodurlnoprot"):
			f.method, f.url = next(), next()
			pat.WriteString(`\"([^` + ns + `]+) ([^` + ns + `]+)\"`)
		case has("%virtualnamequot"):
			f.vh = next()
			pat.WriteString(`\"([^` + ns + `]+)\"`)
		case has("%virtualname"):
			f.vh = next()
			pat.WriteString(field)
		case has("%host_r"):
			f.hostr = next()
			pat.WriteString(field)
		case has("%host"):
			f.host = next()
			pat.WriteString(field)
		case has("%host_proxy"):
			f.host = next()
			pat.WriteString(`(.+?)(?:, .*)*`)
		case has("%method"):
			f.method = next()
			pat.WriteString(field)
		case has("%url"):
			f.url = next()
			pat.WriteString(field)
		case has("%query"):
			f.query = next()
			pat.WriteString(field)
		case has("%code"):
			f.code = next()
			pat.WriteString(field)
		case has("%bytesd"):
			f.size = next()
			pat.WriteString(field)
		case has("%rqtime"):
			f.rqtime = next()
			pat.WriteString(field)
		case has("%refererquot"):
			f.referer = next()
			pat.WriteString(`\"([^\"]*)\"`)
		case has("%referer"):
			f.referer = next()
			pat.WriteString(field)
		case has("%uaquot"):
			f.agent = next()
			pat.WriteString(`\"([^\"]*)\"`)
		case has("%uabracket"):
			f.agent = next()
			pat.WriteString(`\[([^\]]*)\]`)
		case has("%ua"):
			f.agent = next()
			pat.WriteString(field)
		case has("%gzipin"):
			f.gzipin = next()
			pat.WriteString(field)
		case strings.Contains(t, "%gzipout"):
			f.gzipout = next()
			pat.WriteString(field)
		case strings.Contains(t, "%gzipratio"):
			f.ratio, f.ratioIsGzip = next(), true
			pat.WriteString(field)
		case strings.Contains(t, "%deflateratio"):
			f.ratio = next()
			pat.WriteString(field)
		case has("%cluster"):
			f.cluster = next()
			pat.WriteString(field)
		case has("%codemms"):
			if f.code < 0 {
				f.code = next()
				pat.WriteString(field)
			}
		case regexp.MustCompile(`%extra\d+$`).MatchString(t):
			next() // extra sections are not supported; keep the capture so positions stay right
			pat.WriteString(field)
		case has("%otherquot"):
			pat.WriteString(`\"[^\"]*\"`)
		default: // %other and unknown tags
			pat.WriteString(`[^` + ns + `]+`)
		}
	}
	if pat.Len() == 0 {
		return "", fmt.Errorf("no recognized format tag in personalized LogFormat string")
	}
	f.nfields = i
	return pat.String(), nil
}
