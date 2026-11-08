package main

// month holds everything awstats.pl keeps in its %_xxx hashes for one month (DatabaseBreak=month).
// Each Perl hash is its own map, because a key that exists in one hash and not in another
// (for example _filetypes_k without _filetypes_h after a 206 continuation) changes what is written.
type month struct {
	year, mon string // "2026", "07"

	firstTime, lastTime int64

	timeP, timeH, timeK, timeNvP, timeNvH, timeNvK [24]int64

	dayPages, dayHits, dayBytes, dayVisits map[string]int64

	domP, domH, domK map[string]int64

	hostP, hostH, hostK, hostL, hostS map[string]int64
	hostU                             map[string]string
	waitL, waitS                      map[string]int64
	waitE, waitU                      map[string]string

	loginP, loginH, loginK, loginL map[string]int64
	robotH, robotK, robotL, robotR map[string]int64
	wormH, wormK, wormL            map[string]int64
	emailsH, emailsK, emailsL      map[string]int64
	emailrH, emailrK, emailrL      map[string]int64

	session map[string]int64

	urlP, urlK, urlE, urlX map[string]int64

	ftH, ftK, ftGzIn, ftGzOut map[string]int64

	downloads     map[string]*[3]int64 // hits, 206 hits, bytes
	downloadOrder []string

	osH, osP, browserH, browserP, screensize map[string]int64
	unknownRefL, unknownRefBrowserL          map[string]int64

	fromP, fromH [6]int64

	seP, seH             map[string]int64
	pagerefsP, pagerefsH map[string]int64
	keyphrases           map[string]int64
	keywords             map[string]int64 // rebuilt from keyphrases when saving

	errorsH, errorsK map[string]int64
	siderH           map[string]map[string]int64  // code -> url -> hits
	refererH         map[string]map[string]string // code -> url -> last referer
	errHostH         map[string]map[string]string // code -> url -> last host

	clusterP, clusterH, clusterK map[string]int64
	miscP, miscH, miscK          map[string]int64
	filesize, requesttime        map[string]int64
}

func m64() map[string]int64 { return map[string]int64{} }

func newMonth(year, mon string, trap []string) *month {
	m := &month{year: year, mon: mon,
		dayPages: m64(), dayHits: m64(), dayBytes: m64(), dayVisits: m64(),
		domP: m64(), domH: m64(), domK: m64(),
		hostP: m64(), hostH: m64(), hostK: m64(), hostL: m64(), hostS: m64(), hostU: map[string]string{},
		waitL: m64(), waitS: m64(), waitE: map[string]string{}, waitU: map[string]string{},
		loginP: m64(), loginH: m64(), loginK: m64(), loginL: m64(),
		robotH: m64(), robotK: m64(), robotL: m64(), robotR: m64(),
		wormH: m64(), wormK: m64(), wormL: m64(),
		emailsH: m64(), emailsK: m64(), emailsL: m64(), emailrH: m64(), emailrK: m64(), emailrL: m64(),
		session: m64(), urlP: m64(), urlK: m64(), urlE: m64(), urlX: m64(),
		ftH: m64(), ftK: m64(), ftGzIn: m64(), ftGzOut: m64(), downloads: map[string]*[3]int64{},
		osH: m64(), osP: m64(), browserH: m64(), browserP: m64(), screensize: m64(),
		unknownRefL: m64(), unknownRefBrowserL: m64(),
		seP: m64(), seH: m64(), pagerefsP: m64(), pagerefsH: m64(), keyphrases: m64(), keywords: m64(),
		errorsH: m64(), errorsK: m64(),
		siderH: map[string]map[string]int64{}, refererH: map[string]map[string]string{}, errHostH: map[string]map[string]string{},
		clusterP: m64(), clusterH: m64(), clusterK: m64(), miscP: m64(), miscH: m64(), miscK: m64(),
		filesize: m64(), requesttime: m64(),
	}
	for _, c := range trap {
		m.siderH[c], m.refererH[c], m.errHostH[c] = m64(), map[string]string{}, map[string]string{}
	}
	return m
}

func (m *month) download(u string) *[3]int64 {
	d := m.downloads[u]
	if d == nil {
		d = &[3]int64{}
		m.downloads[u] = d
		m.downloadOrder = append(m.downloadOrder, u)
	}
	return d
}

// addIf mirrors awstats.pl's "if ($field[n]) { $h{$k} += $field[n] }" when merging a data file:
// zero values do not create keys.
func addIf(h map[string]int64, k string, v int64) {
	if v != 0 {
		h[k] += v
	}
}
