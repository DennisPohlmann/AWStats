package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
)

// lib.json holds the detection tables of AWStats (browsers, operating systems, robots, search
// engines, worms, mime types), exported once from the original lib/*.pm files.
//
//go:embed lib.json
var libJSON []byte

type idRegex struct {
	id string // the pattern as written in the table; AWStats uses it as key
	re *regexp.Regexp
}

type lib struct {
	browsers      []idRegex
	os            []idRegex
	osID          map[string]string
	robots        []idRegex
	worms         []idRegex
	wormID        map[string]string
	searchEngines []idRegex
	seID          map[string]string
	seNotKeys     map[string]*regexp.Regexp
	seKnownURL    map[string]*regexp.Regexp // ^(?:known)  for s/^$SearchEnginesKnownUrl{$se}//
	seKnownTail   map[string]*regexp.Regexp // known(.*)$  for keys outside the query string
	seNoQuery     map[string]bool
	wordsToClean  []*regexp.Regexp
	safariBuild   map[string]string
	mime          map[string][2]string
}

func compileList(raw []string) ([]idRegex, error) {
	out := make([]idRegex, 0, len(raw))
	for _, p := range raw {
		re, err := regexp.Compile("(?i)" + p)
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", p, err)
		}
		out = append(out, idRegex{p, re})
	}
	return out, nil
}

// loadLib builds the lists in the order awstats.pl uses for the given detection levels.
func loadLib(levelRobots, levelSE int) (*lib, error) {
	var t struct {
		Browsers     []string             `json:"@BrowsersSearchIDOrder"`
		OS           []string             `json:"@OSSearchIDOrder"`
		OSID         map[string]string    `json:"%OSHashID"`
		Robots1      []string             `json:"@RobotsSearchIDOrder_list1"`
		Robots2      []string             `json:"@RobotsSearchIDOrder_list2"`
		RobotsGen    []string             `json:"@RobotsSearchIDOrder_listgen"`
		Worms        []string             `json:"@WormsSearchIDOrder"`
		WormID       map[string]string    `json:"%WormsHashID"`
		SE1          []string             `json:"@SearchEnginesSearchIDOrder_list1"`
		SEGen        []string             `json:"@SearchEnginesSearchIDOrder_listgen"`
		SEID         map[string]string    `json:"%SearchEnginesHashID"`
		SENot        map[string]string    `json:"%NotSearchEnginesKeys"`
		SEKnown      map[string]string    `json:"%SearchEnginesKnownUrl"`
		SENoQuery    map[string]any       `json:"%SearchEnginesWithKeysNotInQuery"`
		WordsToClean []string             `json:"@WordsToCleanSearchUrl"`
		SafariBuild  map[string]string    `json:"%BrowsersSafariBuildToVersionHash"`
		Mime         map[string][2]string `json:"%MimeHashLib"`
	}
	if err := json.Unmarshal(libJSON, &t); err != nil {
		return nil, err
	}
	l := &lib{osID: t.OSID, wormID: t.WormID, seID: t.SEID, safariBuild: t.SafariBuild, mime: t.Mime,
		seNotKeys: map[string]*regexp.Regexp{}, seKnownURL: map[string]*regexp.Regexp{}, seKnownTail: map[string]*regexp.Regexp{}, seNoQuery: map[string]bool{}}
	var err error
	if l.browsers, err = compileList(t.Browsers); err != nil {
		return nil, err
	}
	if l.os, err = compileList(t.OS); err != nil {
		return nil, err
	}
	if l.worms, err = compileList(t.Worms); err != nil {
		return nil, err
	}
	var robots, se []string
	if levelRobots >= 1 {
		robots = append(robots, t.Robots1...)
		if levelRobots >= 2 {
			robots = append(robots, t.Robots2...)
		}
		robots = append(robots, t.RobotsGen...)
	}
	if levelSE >= 1 {
		se = append(append(se, t.SE1...), t.SEGen...) // there is no list2 for search engines
	}
	if l.robots, err = compileList(robots); err != nil {
		return nil, err
	}
	if l.searchEngines, err = compileList(se); err != nil {
		return nil, err
	}
	for k, v := range t.SENot {
		if l.seNotKeys[k], err = regexp.Compile("(?i)" + v); err != nil {
			return nil, err
		}
	}
	for k, v := range t.SEKnown {
		if l.seKnownURL[k], err = regexp.Compile("^(?:" + v + ")"); err != nil {
			return nil, err
		}
		if l.seKnownTail[k], err = regexp.Compile(v + "(.*)$"); err != nil {
			return nil, err
		}
	}
	for k := range t.SENoQuery {
		l.seNoQuery[k] = true
	}
	for _, w := range t.WordsToClean {
		re, err := regexp.Compile("(?i)" + w)
		if err != nil {
			return nil, err
		}
		l.wordsToClean = append(l.wordsToClean, re)
	}
	return l, nil
}
