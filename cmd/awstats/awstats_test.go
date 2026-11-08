package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// The golden data files were written by the original awstats.pl from the same logs and configs.
// The Go updater must produce the same sections and rows. Row order inside a section is not
// compared (awstats.pl writes most sections in Perl hash order), nor LastUpdate (the run time).

// rowsOf returns section -> sorted rows of a data file, without MAP and LastUpdate.
func rowsOf(t *testing.T, path string, dropErrorReferer bool) map[string][]string {
	t.Helper()
	df, err := readDataFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]string{}
	for name, rows := range df.sections {
		if name == "MAP" {
			continue
		}
		var lines []string
		for _, r := range rows {
			if name == "GENERAL" && r[0] == "LastUpdate" {
				continue
			}
			if dropErrorReferer && strings.HasPrefix(name, "SIDER_") && len(r) > 2 {
				r = r[:2]
			}
			lines = append(lines, strings.Join(r, " "))
		}
		sort.Strings(lines)
		out[name] = lines
	}
	return out
}

func compareDirs(t *testing.T, goldenDir, gotDir, goldenSuffix, gotSuffix string, dropErrorReferer bool) {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(goldenDir, "awstats*"+goldenSuffix+".txt"))
	if len(files) == 0 {
		t.Fatalf("no golden files in %s", goldenDir)
	}
	for _, g := range files {
		name := strings.TrimSuffix(filepath.Base(g), goldenSuffix+".txt") + gotSuffix + ".txt"
		want, got := rowsOf(t, g, dropErrorReferer), rowsOf(t, filepath.Join(gotDir, name), dropErrorReferer)
		for sec := range merged(want, got) {
			w, h := want[sec], got[sec]
			if strings.Join(w, "\n") == strings.Join(h, "\n") {
				continue
			}
			onlyW, onlyG := diff(w, h), diff(h, w)
			t.Errorf("%s section %s differs: awstats.pl has %d rows not in Go, Go has %d rows not in awstats.pl\n  awstats.pl: %s\n  go:         %s",
				name, sec, len(onlyW), len(onlyG), first(onlyW), first(onlyG))
		}
	}
	got, _ := filepath.Glob(filepath.Join(gotDir, "awstats*.txt"))
	if len(got) != len(files) {
		t.Errorf("Go wrote %d data files, awstats.pl %d", len(got), len(files))
	}
}

func merged(a, b map[string][]string) map[string]bool {
	m := map[string]bool{}
	for k := range a {
		m[k] = true
	}
	for k := range b {
		m[k] = true
	}
	return m
}

func diff(a, b []string) []string {
	n := map[string]int{}
	for _, x := range b {
		n[x]++
	}
	var out []string
	for _, x := range a {
		if n[x] > 0 {
			n[x]--
			continue
		}
		out = append(out, x)
	}
	return out
}

func first(a []string) string {
	if len(a) > 3 {
		a = a[:3]
	}
	return strings.Join(a, " | ")
}

// update runs the updater for testdata/conf/awstats.<name>.conf with the log in logDir.
func update(t *testing.T, name, logDir, dataDir string) {
	t.Helper()
	t.Setenv("AWSTATS_TESTDATA", logDir)
	t.Setenv("AWSTATS_DATA", dataDir)
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.Local)
	if err := run(name, "testdata/conf", "", now, io.Discard); err != nil {
		t.Fatal(err)
	}
}

func TestSameAsAwstatsPl(t *testing.T) {
	for _, name := range []string{"edge", "vh"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			abs, _ := filepath.Abs("testdata")
			update(t, name, abs, dir)
			compareDirs(t, filepath.Join("testdata/golden", name), dir, "."+name, "."+name, false)
		})
	}
}

// testdata/golden/{apache,nginx} were written by awstats.pl from the web interface test logs
// (webui/testdata/build.sh makes the logs).
func TestSameAsAwstatsPlOnWebUITestData(t *testing.T) {
	logs := "../../webui/testdata/logs"
	if _, err := os.Stat(logs); err != nil {
		t.Skip("webui test data not found")
	}
	tmp := t.TempDir()
	copyFile(t, filepath.Join(logs, "apache-access.log"), filepath.Join(tmp, "apache.log"), false)
	// nginx-access.log was rotated after the data was built: the older half is in .1.gz.
	copyFile(t, filepath.Join(logs, "nginx-access.log.1.gz"), filepath.Join(tmp, "nginx.log"), true)
	appendFile(t, filepath.Join(logs, "nginx-access.log"), filepath.Join(tmp, "nginx.log"))
	for _, site := range []string{"apache", "nginx"} {
		t.Run(site, func(t *testing.T) {
			dir := t.TempDir()
			update(t, site, tmp, dir)
			compareDirs(t, filepath.Join("testdata/golden", site), dir, "."+site, "."+site, false)
		})
	}
}

// An update that starts in the middle of a month must give the same result as one run over the
// whole log: this checks LastLine, the direct seek and the merge of visits with the data file.
func TestIncrementalUpdate(t *testing.T) {
	logs := "../../webui/testdata/logs"
	if _, err := os.Stat(logs); err != nil {
		t.Skip("webui test data not found")
	}
	data, err := os.ReadFile(filepath.Join(logs, "apache-access.log"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.SplitAfter(string(data), "\n")
	tmp := t.TempDir()
	log := filepath.Join(tmp, "apache.log")
	dir := t.TempDir()
	for _, cut := range []int{len(lines) * 37 / 100, len(lines) * 81 / 100, len(lines)} {
		if err := os.WriteFile(log, []byte(strings.Join(lines[:cut], "")), 0o644); err != nil {
			t.Fatal(err)
		}
		update(t, "apache", tmp, dir)
	}
	// The last referrer of an error URL is the one of the data file after an update (awstats.pl
	// does the same), so it can differ from a single run.
	compareDirs(t, "testdata/golden/apache", dir, ".apache", ".apache", true)
}

func copyFile(t *testing.T, src, dst string, gunzip bool) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	var r io.Reader = in
	if gunzip {
		gz, err := gzip.NewReader(in)
		if err != nil {
			t.Fatal(err)
		}
		r = gz
	}
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, r); err != nil {
		t.Fatal(err)
	}
}

func appendFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(dst, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
}

func TestHelpers(t *testing.T) {
	for in, want := range map[string]int64{"123": 123, "-": 0, "": 0, "12-3": 12, " 7x": 7} {
		if got := pint(in); got != want {
			t.Errorf("pint(%q) = %d, want %d", in, got, want)
		}
	}
	if got := decodeRFC3986("/a%7Eb%2Fc%41"); got != "/a~b%2FcA" {
		t.Errorf("decodeRFC3986 = %q", got)
	}
	if got := changeWordSeparatorsIntoSpace("a+b%20c,,d"); got != "a b c d" {
		t.Errorf("changeWordSeparatorsIntoSpace = %q", got)
	}
	if got := sessionRange(20261001100000, 20261001100500); got != "2mn-5mn" {
		t.Errorf("sessionRange = %q", got)
	}
	if got := fmt.Sprint(substituteTags("log.%YYYY-24%MM-24%DD-24", time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC))); got != "log.20260228" {
		t.Errorf("substituteTags = %q", got)
	}
}

func TestAllConfigs(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"awstats.b.example.com.conf", "awstats.a.conf", "awstats.model.conf", "other.conf"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := allConfigs(dir)
	if err != nil || strings.Join(got, ",") != "a,b.example.com" {
		t.Errorf("allConfigs = %v, %v", got, err)
	}
	if _, err := allConfigs(t.TempDir()); err == nil {
		t.Error("allConfigs of an empty directory: no error")
	}
}
