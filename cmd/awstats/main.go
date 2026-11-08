// Command awstats builds and updates AWStats data files (awstatsMMYYYY.<config>.txt) from web
// server logs. It replaces the update process of awstats.pl ("awstats.pl -config=x -update") and
// writes the same text format, so the web interface and existing data files keep working.
package main

import (
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// version is set at build time (-ldflags "-X main.version=9.1.0").
var version = "dev"

// dataFileVersion is written in the first line of data files. Readers parse major.minor from it.
func dataFileVersion() string {
	if regexp.MustCompile(`^\d+\.\d+`).MatchString(version) {
		return version + " (build go)"
	}
	return "9.0 (build go-" + version + ")"
}

func main() {
	var (
		configName = flag.String("config", "", "config name: reads awstats.<name>.conf")
		configDir  = flag.String("configdir", "", "directory of the config file (default: the usual AWStats locations)")
		logFile    = flag.String("logfile", "", "log file to read instead of LogFile from the config")
		_          = flag.Bool("update", true, "update the data files (the only action; accepted for compatibility)")
		showVer    = flag.Bool("version", false, "print the version")
		all        = flag.Bool("all", false, "update every awstats.<name>.conf in -configdir (default /etc/awstats)")
		every      = flag.Duration("every", 0, "repeat the update at this interval, e.g. -every=1h (default: run once)")
	)
	flag.CommandLine.Init(os.Args[0], flag.ContinueOnError)
	// awstats.pl accepts -LogFile=... and -configdir=...; accept any case for the flag names.
	args := os.Args[1:]
	for i, a := range args {
		if k, v, ok := strings.Cut(a, "="); ok && strings.HasPrefix(k, "-") {
			args[i] = strings.ToLower(k) + "=" + v
		} else {
			args[i] = strings.ToLower(a)
		}
	}
	if err := flag.CommandLine.Parse(args); err != nil {
		os.Exit(2)
	}
	if *showVer {
		fmt.Println("awstats", version)
		return
	}
	if *all && (*configName != "" || *logFile != "") {
		fmt.Fprintln(os.Stderr, "Error: -all can't be used with -config or -logfile")
		os.Exit(2)
	}
	if *all && *configDir == "" {
		*configDir = "/etc/awstats"
	}
	update := func() bool {
		names := []string{*configName}
		if *all {
			var err error
			if names, err = allConfigs(*configDir); err != nil { // read again each round: new sites are picked up
				fmt.Fprintln(os.Stderr, "Error:", err)
				return false
			}
		}
		ok := true
		for _, n := range names {
			if *all {
				fmt.Printf("Update for config %q\n", n)
			}
			if err := run(n, *configDir, *logFile, time.Now(), os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				ok = false
			}
		}
		return ok
	}
	if *every <= 0 {
		if !update() {
			os.Exit(1)
		}
		return
	}
	// Long-running mode for a container: an error is logged and retried in the next round.
	// SIGTERM (docker stop) ends it with exit code 0: during an update it finishes that update,
	// and a kill after the stop timeout is safe too, because data files are written to a
	// temporary file and renamed.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, os.Interrupt)
	for {
		update()
		select {
		case <-stop:
			return
		case <-time.After(*every):
		}
	}
}

// allConfigs returns the names of all awstats.<name>.conf files in dir.
func allConfigs(dir string) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "awstats.*.conf"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, f := range files {
		if n := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "awstats."), ".conf"); n != "model" {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no awstats.<name>.conf in %s", dir)
	}
	return names, nil
}

func run(name, dir, logOverride string, now time.Time, out io.Writer) error {
	path, err := findConfig(name, dir)
	if err != nil {
		return err
	}
	c, err := loadConfig(path)
	if err != nil {
		return err
	}
	if logOverride != "" {
		c.LogFile = logOverride
	}
	if c.LogFile == "" {
		return fmt.Errorf("LogFile parameter is not defined in %s", path)
	}
	c.LogFile = substituteTags(c.LogFile, now)
	u, err := newUpdater(c, now)
	if err != nil {
		return err
	}
	u.site = name
	fmt.Fprintf(out, "Create/Update database for config \"%s\" by AWStats version %s (Go)\n", path, version)
	fmt.Fprintf(out, "From data in log file \"%s\"...\n", c.LogFile)
	if err := u.readLastLine(); err != nil {
		return err
	}
	if err := u.processLog(); err != nil {
		return err
	}
	fmt.Fprintf(out, "Jumped lines in file: %d\n", u.oldLines)
	fmt.Fprintf(out, "Parsed lines in file: %d\n", u.parsed)
	fmt.Fprintf(out, " Found %d dropped records,\n", u.dropped)
	fmt.Fprintf(out, " Found %d comments,\n", u.comments)
	fmt.Fprintf(out, " Found %d blank records,\n", u.blanks)
	fmt.Fprintf(out, " Found %d corrupted records,\n", u.corrupted)
	fmt.Fprintf(out, " Found %d old records,\n", u.oldLines)
	fmt.Fprintf(out, " Found %d new qualified records.\n", u.newLines)
	return nil
}

// readLastLine loads LastLine from the newest data file of this config, so an update skips the
// records it already counted.
func (u *updater) readLastLine() error {
	re := regexp.MustCompile(`^awstats(\d\d)(\d\d\d\d)` + func() string {
		if u.site == "" {
			return ""
		}
		return regexp.QuoteMeta("." + u.site)
	}() + `\.txt$`)
	entries, err := os.ReadDir(u.c.DirData)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var best, bestFile string
	for _, e := range entries {
		if m := re.FindStringSubmatch(e.Name()); m != nil && m[2]+m[1] > best {
			best, bestFile = m[2]+m[1], e.Name()
		}
	}
	if bestFile == "" {
		return nil
	}
	df, err := readDataFile(filepath.Join(u.c.DirData, bestFile))
	if err != nil {
		return err
	}
	for _, r := range df.sections["GENERAL"] {
		if r[0] == "LastLine" {
			u.lastLine, u.lastLineNumber, u.lastLineOffset, u.lastLineChecksum = fi(r, 1), fi(r, 2), fi(r, 3), fi(r, 4)
		}
	}
	return nil
}

type logReader struct {
	r      *bufio.Reader
	closer func() error
	seeker io.Seeker
}

func openLog(name string) (*logReader, error) {
	if strings.HasSuffix(strings.TrimSpace(name), "|") {
		cmd := exec.Command("sh", "-c", strings.TrimSuffix(strings.TrimSpace(name), "|"))
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, fmt.Errorf("couldn't run log command %q: %w", name, err)
		}
		return &logReader{r: bufio.NewReaderSize(stdout, 1<<20), closer: cmd.Wait}, nil
	}
	f, err := os.Open(name)
	if err != nil {
		return nil, fmt.Errorf("couldn't open server log file %q: %w", name, err)
	}
	if strings.HasSuffix(name, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return &logReader{r: bufio.NewReaderSize(gz, 1<<20), closer: f.Close}, nil
	}
	return &logReader{r: bufio.NewReaderSize(f, 1<<20), closer: f.Close, seeker: f}, nil
}

func (l *logReader) seek(off int64) bool {
	if l.seeker == nil {
		return false
	}
	if _, err := l.seeker.Seek(off, io.SeekStart); err != nil {
		return false
	}
	l.r.Reset(l.seeker.(io.Reader))
	return true
}

func (u *updater) processLog() error {
	lr, err := openLog(u.c.LogFile)
	if err != nil {
		return err
	}
	defer lr.closer()
	var offset, baseLineNb int64
	if u.lastLine != 0 && u.lastLineNumber != 0 && u.lastLineOffset != 0 && u.lastLineChecksum != 0 && lr.seek(u.lastLineOffset) {
		line, err := lr.r.ReadString('\n')
		if (err == nil || line != "") && checkSum(strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")) == u.lastLineChecksum {
			baseLineNb, offset, u.newLinePhase = u.lastLineNumber, u.lastLineOffset+int64(len(line)), true
		} else {
			lr.seek(0)
		}
	}
	var lastOffset int64
	var lastText string
	for {
		line, err := lr.r.ReadString('\n')
		if line == "" && err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		lastOffset, lastText = offset, strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		offset += int64(len(line))
		if err := u.processLine(strings.TrimSuffix(line, "\n"), lastOffset, baseLineNb); err != nil {
			return err
		}
	}
	if u.cur != nil {
		return u.saveMonth(baseLineNb+u.parsed, lastOffset, checkSum(lastText))
	}
	return nil
}

var reTag = regexp.MustCompile(`%(YYYY|YY|MM|MO|DD|HH|NS)(?:-(\d+))?`)

// substituteTags replaces the date tags awstats.pl supports in LogFile, e.g. access.log.%YYYY-24%MM-24%DD-24.
func substituteTags(s string, now time.Time) string {
	return reTag.ReplaceAllStringFunc(s, func(m string) string {
		p := reTag.FindStringSubmatch(m)
		t := now
		if p[2] != "" {
			n, _ := strconv.Atoi(p[2])
			t = now.Add(-time.Duration(n) * time.Hour)
		}
		switch p[1] {
		case "YYYY":
			return t.Format("2006")
		case "YY":
			return t.Format("06")
		case "MM":
			return t.Format("01")
		case "MO":
			return t.Format("Jan")
		case "DD":
			return t.Format("02")
		case "HH":
			return t.Format("15")
		default:
			return strconv.FormatInt(t.Unix(), 10)
		}
	})
}
