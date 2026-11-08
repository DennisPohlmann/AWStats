package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// config holds the awstats.<site>.conf values the update process uses, after the
// defaults of Check_Config in awstats.pl.
type config struct {
	file string
	raw  map[string]string

	LogFile, LogType, LogFormat, LogSeparator, SiteDomain, DirData   string
	DNSLookup                                                        int
	HostAliases, SkipHosts, SkipUserAgents, SkipFiles, SkipReferrers []*regexp.Regexp
	OnlyHosts, OnlyUsers, OnlyUserAgents, OnlyFiles, NotPageFiles    []*regexp.Regexp
	DefaultFile                                                      []string
	NotPageList, ValidHTTPCodes                                      map[string]bool
	TrapInfosForHTTPErrorCodes                                       []string
	URLNotCaseSensitive, URLWithAnchor, URLWithQuery                 bool
	URLReferrerWithQuery, AuthenticatedUsersNotCaseSensitive         bool
	KeyWordsNotSensitive, DecodeUA                                   bool
	URLQuerySeparators                                               string
	URLWithQueryWithOnly, URLWithQueryWithout                        []string
	MiscTrackerUrl, ShowHTTPErrorsPageDetail                         string
	LevelForBrowsers, LevelForOS, LevelForReferer, LevelForRobots    int
	LevelForSearchEngines, LevelForKeywords, LevelForFileTypes       int
	LevelForWorms                                                    int
	NbOfLinesForCorruptedLog                                         int
	MaxNbOf                                                          map[string]int
}

var envVar = regexp.MustCompile(`__([^\s_]+(?:_[^\s_]+)*)__`)

func expandEnv(s string) string {
	for {
		m := envVar.FindStringSubmatch(s)
		if m == nil {
			return s
		}
		s = strings.ReplaceAll(s, m[0], os.Getenv(m[1]))
	}
}

// parseConfigFile reads key=value lines like Parse_Config: Include, comments, quotes, __ENV__.
func parseConfigFile(path string, level int, into map[string]string, mainFile string) error {
	if level > 10 {
		return fmt.Errorf("can't read down more than 10 levels of includes (%s)", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	includeRe := regexp.MustCompile(`^#?[Ii]nclude "([^"]+)"`)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSuffix(sc.Text(), "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if m := includeRe.FindStringSubmatch(line); m != nil && (strings.HasPrefix(line, "Include") || strings.HasPrefix(line, "#include")) {
			inc := expandEnv(m[1])
			if !filepath.IsAbs(inc) {
				inc = filepath.Join(filepath.Dir(mainFile), inc)
			}
			if err := parseConfigFile(inc, level+1, into, mainFile); err != nil {
				return fmt.Errorf("could not open include file: %w", err)
			}
			continue
		}
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			continue
		}
		if i := regexp.MustCompile(`\s#`).FindStringIndex(line); i != nil {
			line = line[:i[0]]
		}
		param, value, ok := strings.Cut(line, "=")
		param = strings.TrimSpace(param)
		if !ok || param == "" {
			fmt.Fprintf(os.Stderr, "Warning: Syntax error in file '%s'. Config line is ignored: %s\n", path, line)
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.TrimPrefix(value, `"`)
		value = regexp.MustCompile(`";?$`).ReplaceAllString(value, "")
		value = expandEnv(value)
		into[param] = value
	}
	return sc.Err()
}

// regexList turns "a b REGEX[c]" into case insensitive regexes: exact match unless REGEX[...].
func regexList(v string) ([]*regexp.Regexp, error) {
	var out []*regexp.Regexp
	seen := map[string]bool{}
	for _, e := range strings.Fields(v) {
		var p string
		if m := regexp.MustCompile(`(?i)^REGEX\[(.*)\]$`).FindStringSubmatch(e); m != nil {
			p = m[1]
		} else {
			p = "^" + regexp.QuoteMeta(e) + "$"
		}
		if p == "" || seen[strings.ToLower(p)] {
			continue
		}
		seen[strings.ToLower(p)] = true
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("bad regex %q: %w", p, err)
		}
		out = append(out, re)
	}
	return out, nil
}

func set(v string) map[string]bool {
	m := map[string]bool{}
	for _, e := range strings.Fields(v) {
		m[e] = true
	}
	return m
}

func atoiDefault(v string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n >= 0 {
		return n
	}
	return def
}

func loadConfig(path string) (*config, error) {
	raw := map[string]string{}
	if err := parseConfigFile(path, 1, raw, path); err != nil {
		return nil, err
	}
	c := &config{file: path, raw: raw}
	get := func(k string) string { return raw[k] }
	bool01 := func(k string) bool { return get(k) == "1" }

	c.LogFile = get("LogFile")
	c.LogType = strings.ToUpper(get("LogType"))
	if !regexp.MustCompile(`^[WSMF]`).MatchString(c.LogType) {
		c.LogType = "W"
	}
	c.LogFormat = strings.ReplaceAll(get("LogFormat"), `\`, "")
	if c.LogFormat == "" {
		return nil, fmt.Errorf("LogFormat parameter is not defined in config file %s", path)
	}
	c.LogSeparator = get("LogSeparator")
	if c.LogSeparator == "" {
		c.LogSeparator = `\s`
	}
	c.SiteDomain = get("SiteDomain")
	if c.SiteDomain == "" {
		return nil, fmt.Errorf("SiteDomain parameter not defined in config file %s", path)
	}
	c.DirData = get("DirData")
	if c.DirData == "" {
		c.DirData = "."
	}
	c.DNSLookup = atoiDefault(get("DNSLookup"), 0)

	var err error
	lists := []struct {
		dst *[]*regexp.Regexp
		key string
	}{
		{&c.HostAliases, "HostAliases"}, {&c.SkipHosts, "SkipHosts"}, {&c.SkipUserAgents, "SkipUserAgents"},
		{&c.SkipFiles, "SkipFiles"}, {&c.OnlyHosts, "OnlyHosts"}, {&c.OnlyUsers, "OnlyUsers"},
		{&c.OnlyUserAgents, "OnlyUserAgents"}, {&c.OnlyFiles, "OnlyFiles"}, {&c.NotPageFiles, "NotPageFiles"},
	}
	for _, l := range lists {
		if *l.dst, err = regexList(get(l.key)); err != nil {
			return nil, fmt.Errorf("%s: %w", l.key, err)
		}
	}
	if bl := get("SkipReferrersBlackList"); bl != "" {
		data, err := os.ReadFile(bl)
		if err != nil {
			return nil, fmt.Errorf("failed to open blacklist: %w", err)
		}
		for _, l := range strings.Split(string(data), "\n") {
			l = strings.Replace(l, " ", "", 1)
			l = regexp.MustCompile(`#.*`).ReplaceAllString(l, "")
			l = strings.TrimRight(l, "\r")
			if l == "" {
				continue
			}
			re, err := regexp.Compile("(?i)" + l)
			if err != nil {
				return nil, fmt.Errorf("SkipReferrersBlackList %q: %w", l, err)
			}
			c.SkipReferrers = append(c.SkipReferrers, re)
		}
	}
	// SiteDomain always comes first; without HostAliases awstats.pl adds localhost and 127.0.0.1.
	site := regexp.MustCompile("(?i)^" + regexp.QuoteMeta(strings.ToLower(c.SiteDomain)) + "$")
	if len(c.HostAliases) == 0 {
		c.HostAliases = []*regexp.Regexp{site, regexp.MustCompile(`(?i)^localhost$`), regexp.MustCompile(`(?i)^127\.0\.0\.1$`)}
	} else {
		c.HostAliases = append([]*regexp.Regexp{site}, c.HostAliases...)
	}

	c.DefaultFile = strings.Fields(get("DefaultFile"))
	if len(c.DefaultFile) == 0 {
		c.DefaultFile = []string{"index.html"}
	}
	if v, ok := raw["NotPageList"]; ok {
		c.NotPageList = set(v)
	} else {
		c.NotPageList = set("css js class gif jpg jpeg png bmp ico rss swf webp xml eot woff woff2")
	}
	c.ValidHTTPCodes = set(get("ValidHTTPCodes"))
	if len(c.ValidHTTPCodes) == 0 {
		c.ValidHTTPCodes = set("200 304")
	}
	c.TrapInfosForHTTPErrorCodes = strings.Fields(get("TrapInfosForHTTPErrorCodes"))
	if len(c.TrapInfosForHTTPErrorCodes) == 0 {
		c.TrapInfosForHTTPErrorCodes = []string{"404"}
	}
	c.URLNotCaseSensitive = bool01("URLNotCaseSensitive")
	c.URLWithAnchor = bool01("URLWithAnchor")
	c.URLWithQuery = bool01("URLWithQuery")
	c.URLReferrerWithQuery = bool01("URLReferrerWithQuery")
	c.AuthenticatedUsersNotCaseSensitive = bool01("AuthenticatedUsersNotCaseSensitive")
	c.KeyWordsNotSensitive = true // awstats.pl default
	if v, ok := raw["KeyWordsNotSensitive"]; ok {
		c.KeyWordsNotSensitive = truthy(v)
	}
	c.DecodeUA = bool01("DecodeUA")
	c.URLQuerySeparators = strings.Join(strings.Fields(get("URLQuerySeparators")), "")
	if c.URLQuerySeparators == "" {
		c.URLQuerySeparators = "?;"
	}
	c.URLWithQueryWithOnly = strings.Fields(get("URLWithQueryWithOnlyFollowingParameters"))
	c.URLWithQueryWithout = strings.Fields(get("URLWithQueryWithoutFollowingParameters"))
	if len(c.URLWithQueryWithOnly) > 0 && len(c.URLWithQueryWithout) > 0 {
		return nil, fmt.Errorf("URLWithQueryWithOnlyFollowingParameters and URLWithQueryWithoutFollowingParameters can't be both set")
	}
	c.MiscTrackerUrl = get("MiscTrackerUrl")
	if c.MiscTrackerUrl == "" {
		c.MiscTrackerUrl = "/js/awstats_misc_tracker.js"
	}
	c.ShowHTTPErrorsPageDetail = get("ShowHTTPErrorsPageDetail")
	if !regexp.MustCompile(`[RH]`).MatchString(c.ShowHTTPErrorsPageDetail) {
		c.ShowHTTPErrorsPageDetail = "R"
	}
	level := func(k string, def int) int {
		if m := regexp.MustCompile(`^\d+`).FindString(get(k)); m != "" {
			n, _ := strconv.Atoi(m)
			return n
		}
		return def
	}
	c.LevelForWorms = level("LevelForWormsDetection", 0)
	c.LevelForRobots = level("LevelForRobotsDetection", 2)
	c.LevelForBrowsers = level("LevelForBrowsersDetection", 2)
	c.LevelForOS = level("LevelForOSDetection", 2)
	c.LevelForReferer = level("LevelForRefererAnalyze", 2)
	c.LevelForFileTypes = level("LevelForFileTypesDetection", 2)
	c.LevelForSearchEngines = level("LevelForSearchEnginesDetection", 2)
	c.LevelForKeywords = level("LevelForKeywordsDetection", 2)
	c.NbOfLinesForCorruptedLog = atoiDefault(get("NbOfLinesForCorruptedLog"), 50)
	if c.NbOfLinesForCorruptedLog < 1 {
		c.NbOfLinesForCorruptedLog = 50
	}
	c.MaxNbOf = map[string]int{}
	for k, def := range map[string]int{"Domain": 10, "HostsShown": 10, "LoginShown": 10, "RobotShown": 10, "WormsShown": 5,
		"PageShown": 10, "OsShown": 10, "BrowsersShown": 10, "ScreenSizesShown": 5, "RefererShown": 10,
		"KeyphrasesShown": 10, "KeywordsShown": 10, "EMailsShown": 20} {
		c.MaxNbOf[k] = def
		if n, err := strconv.Atoi(strings.TrimSpace(get("MaxNbOf" + k))); err == nil && n >= 1 {
			c.MaxNbOf[k] = n
		}
	}
	for _, k := range []string{"ExtraSectionName1", "LoadPlugin"} {
		if _, ok := raw[k]; ok {
			fmt.Fprintf(os.Stderr, "Warning: %s is set in %s; extra sections and plugins are not supported and are ignored.\n", k, path)
		}
	}
	return c, nil
}

// findConfig looks for awstats.<name>.conf like awstats.pl does.
func findConfig(name, dir string) (string, error) {
	var dirs []string
	if dir != "" {
		dirs = []string{dir}
	} else {
		exe, _ := os.Executable()
		dirs = []string{filepath.Dir(exe), "/etc/awstats", "/usr/local/etc/awstats", "/etc", "/etc/opt/awstats"}
	}
	for _, d := range dirs {
		for _, f := range []string{"awstats." + name + ".conf", "awstats.conf"} {
			if name == "" && f != "awstats.conf" {
				continue
			}
			p := filepath.Join(d, f)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p, nil
			}
		}
	}
	return "", fmt.Errorf("couldn't open config file \"awstats.%s.conf\" in %s", name, strings.Join(dirs, ", "))
}
