// streamcut — PoC: обрываем ответ ютуба на середине так, чтобы хвост не попал в канал.
//
// Сервер шлёт DATA, пока у стрима есть HTTP/2-окно, а fhttp возвращает окно
// только из resp.Body.Read. Значит сверх прочитанного в канал попадёт не больше
// одного окна: у Chrome-профилей это 6 МБ (весь ответ), у Firefox — 128 КБ.
//
// Каждый ответ (и полный, и обрезанный) разбирается: JSON после маркеров должен
// декодироваться и содержать реальные данные — playabilityStatus OK, тот же
// videoId, streamingData, channelMetadataRenderer и т.д. Иначе запрос не засчитан.
//
// Трафик — байты, прочитанные из сокета до TLS (WithBandwidthTracker), после
// прогрева соединения и с паузой, чтобы долетели DATA после RST_STREAM.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/http2"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

const (
	endTag       = "</script>"
	playerMarker = "var ytInitialPlayerResponse = "
	dataMarker   = "var ytInitialData = "
)

// Oo9EbArcQ1c, zRtGL0-5rg4 — из ids.txt (17 и 43 мин); dQw4w9WgXcQ — рикролл; kJQP7kiw5Fk — Despacito.
const defaultTargets = "watch:Oo9EbArcQ1c,watch:zRtGL0-5rg4,watch:dQw4w9WgXcQ,watch:kJQP7kiw5Fk," +
	"channel:@MrBeast,channel:@MrBeast/videos,channel:channel/UCX6OQ3DkcsbYNE6H8uQQuVA"

type target struct {
	kind, id string // kind: watch | channel
	url      string
	markers  []string // после последнего из них можно обрывать
}

func (t target) label() string { return t.kind + ":" + t.id }

type mode struct {
	name    string
	profile profiles.ClientProfile
	headers http.Header
	cut     bool
	base    string // с каким full-режимом сравнивать трафик
}

type stats struct {
	n, valid   int
	fetched    int // без сетевых ошибок
	wire, body int64
	spent      time.Duration
	fails      map[string]int
	sample     string // что достали из первого валидного ответа
	offsets    string // где стоят маркеры (из full-режима)
}

func main() {
	targetsFlag := flag.String("targets", defaultTargets, "watch:<id> или channel:<путь>, через запятую")
	need := flag.String("need", "player,data", "что нужно с /watch: player, data или оба")
	modesFlag := flag.String("modes", "chrome-full,chrome-cut,firefox-full,firefox-cut",
		"режимы, через запятую; ещё есть chrome-smallwin-cut")
	n := flag.Int("n", 3, "запросов на цель в каждом режиме")
	win := flag.Uint("win", 131072, "окно стрима для chrome-smallwin-cut")
	proxy := flag.String("proxy", "", "proxy url")
	flag.Parse()

	targets, err := parseTargets(*targetsFlag, *need)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	chrome, firefox := chromeHeaders(), firefoxHeaders()
	allModes := []mode{
		{"chrome-full", profiles.Chrome_146, chrome, false, ""},
		{"chrome-cut", profiles.Chrome_146, chrome, true, "chrome-full"},
		{"chrome-smallwin-cut", withStreamWindow(profiles.Chrome_146, uint32(*win)), chrome, true, "chrome-full"},
		{"firefox-full", profiles.Firefox_148, firefox, false, ""},
		{"firefox-cut", profiles.Firefox_148, firefox, true, "firefox-full"},
	}
	var modes []mode
	for _, m := range allModes {
		if slices.Contains(strings.Split(*modesFlag, ","), m.name) {
			modes = append(modes, m)
		}
	}

	results := map[string]map[string]*stats{} // mode → target → stats
	for _, m := range modes {
		fmt.Fprintf(os.Stderr, "%s ", m.name)
		res, err := runMode(m, targets, *n, *proxy)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", m.name, err)
			continue
		}
		results[m.name] = res
	}

	for _, t := range targets {
		fmt.Printf("\n%s\n", t.label())
		for _, m := range modes {
			if st := results[m.name][t.label()]; st != nil && st.sample != "" {
				fmt.Printf("  %s\n", st.sample)
				break
			}
		}
		for _, m := range modes {
			if st := results[m.name][t.label()]; st != nil && st.offsets != "" {
				fmt.Printf("  %s\n", st.offsets)
				break
			}
		}
		fmt.Printf("  %-20s %6s %12s %8s %12s %8s  %s\n", "mode", "valid", "wire KB/req", "saved", "body KB/req", "ms/req", "fails")
		for _, m := range modes {
			st := results[m.name][t.label()]
			if st == nil {
				continue
			}
			saved := ""
			if b := results[m.base][t.label()]; b != nil && b.wire > 0 {
				saved = fmt.Sprintf("%+.0f%%", (float64(st.wire)/float64(b.wire)-1)*100)
			}
			body := 0.0
			if st.fetched > 0 {
				body = float64(st.body) / float64(st.fetched) / 1024
			}
			fmt.Printf("  %-20s %3d/%-2d %12.0f %8s %12.0f %8d  %s\n", m.name, st.valid, st.n,
				float64(st.wire)/float64(st.n)/1024, saved, body,
				st.spent.Milliseconds()/int64(st.n), formatFails(st.fails))
		}
	}
}

func parseTargets(s, need string) ([]target, error) {
	var watchMarkers []string
	for _, w := range strings.Split(need, ",") {
		switch w {
		case "player":
			watchMarkers = append(watchMarkers, playerMarker)
		case "data":
			watchMarkers = append(watchMarkers, dataMarker)
		default:
			return nil, fmt.Errorf("-need: неизвестно %q", w)
		}
	}

	var targets []target
	for _, item := range strings.Split(s, ",") {
		kind, id, ok := strings.Cut(item, ":")
		if !ok {
			return nil, fmt.Errorf("-targets: %q без kind:", item)
		}
		switch kind {
		case "watch":
			targets = append(targets, target{kind, id, "https://www.youtube.com/watch?v=" + id, watchMarkers})
		case "channel":
			targets = append(targets, target{kind, id, "https://www.youtube.com/" + id, []string{dataMarker}})
		default:
			return nil, fmt.Errorf("-targets: неизвестный kind %q", kind)
		}
	}
	return targets, nil
}

func runMode(m mode, targets []target, n int, proxy string) (map[string]*stats, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithClientProfile(m.profile),
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithDisableHttp3(), // у QUIC свои окна, меряем только h2
		tls_client.WithBandwidthTracker(),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
	}
	if proxy != "" {
		opts = append(opts, tls_client.WithProxyUrl(proxy))
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, err
	}
	defer client.CloseIdleConnections()

	// прогрев: хендшейк, SETTINGS и куки не должны попасть в замер
	warm, err := fetch(client, targets[0], m)
	if err != nil {
		return nil, fmt.Errorf("warmup: %w", err)
	}
	if warm.proto != "HTTP/2.0" {
		fmt.Fprintf(os.Stderr, "%s: протокол %s, а не HTTP/2 — окна h2 тут ни при чём\n", m.name, warm.proto)
	}
	time.Sleep(time.Second)

	bt := client.GetBandwidthTracker()
	out := map[string]*stats{}
	for _, t := range targets {
		st := &stats{fails: map[string]int{}}
		bt.Reset()
		for range n {
			start := time.Now()
			res, err := fetch(client, t, m)
			st.spent += time.Since(start)
			st.n++
			if err != nil {
				st.fails[err.Error()]++
				continue
			}
			st.fetched++
			st.body += int64(len(res.body))
			if !m.cut && st.offsets == "" {
				st.offsets = offsets(res.body)
			}
			sample, err := validate(t, res, m.cut)
			if err != nil {
				st.fails[err.Error()]++
				continue
			}
			st.valid++
			if st.sample == "" {
				st.sample = sample
			}
		}
		// DATA, которые были в пути в момент RST_STREAM, тоже считаются
		time.Sleep(2 * time.Second)
		st.wire = bt.GetReadBytes()
		out[t.label()] = st
		fmt.Fprint(os.Stderr, ".")
	}
	return out, nil
}

type result struct {
	status   int
	proto    string
	finalURL string
	body     []byte // распакованное тело — целиком или до обрыва
	cut      bool   // оборвали до конца ответа
}

func fetch(client tls_client.HttpClient, t target, m mode) (result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url, nil)
	if err != nil {
		return result{}, err
	}
	req.Header = m.headers.Clone()

	resp, err := client.Do(req)
	if err != nil {
		return result{}, err
	}
	// Close недочитанного тела шлёт RST_STREAM(CANCEL); тело уже распаковано fhttp
	defer resp.Body.Close()

	res := result{status: resp.StatusCode, proto: resp.Proto, finalURL: resp.Request.URL.String()}
	var c *cutter
	if m.cut {
		c = newCutter(t.markers)
	}
	buf := make([]byte, 0, 4<<20)
	chunk := make([]byte, 32<<10)
	for {
		n, err := resp.Body.Read(chunk)
		buf = append(buf, chunk[:n]...)
		if c != nil && n > 0 && c.feed(buf, n) {
			res.body, res.cut = buf, true
			return res, nil
		}
		if err == io.EOF {
			res.body = buf
			return res, nil
		}
		if err != nil {
			return res, err
		}
	}
}

// validate проверяет, что в (возможно обрезанном) теле лежат настоящие данные.
func validate(t target, res result, wantCut bool) (string, error) {
	if !strings.HasPrefix(res.finalURL, "https://www.youtube.com/") {
		return "", fmt.Errorf("redirect → %s", res.finalURL)
	}
	if res.status != http.StatusOK {
		return "", fmt.Errorf("status %d", res.status)
	}
	if wantCut && !res.cut {
		return "", errors.New("не оборвали: маркеры не закрылись")
	}

	var parts []string
	for _, marker := range t.markers {
		raw, err := extractJSON(res.body, marker)
		if err != nil {
			return "", fmt.Errorf("%s: %w", markerName(marker), err)
		}
		var part string
		switch {
		case marker == playerMarker:
			part, err = checkPlayer(raw, t.id)
		case t.kind == "watch":
			part, err = checkWatchData(raw)
		default:
			part, err = checkChannelData(raw)
		}
		if err != nil {
			return "", err
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, " | "), nil
}

func extractJSON(body []byte, marker string) (json.RawMessage, error) {
	i := bytes.Index(body, []byte(marker))
	if i < 0 {
		return nil, errors.New("маркер не найден")
	}
	var raw json.RawMessage
	if err := json.NewDecoder(bytes.NewReader(body[i+len(marker):])).Decode(&raw); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}
	return raw, nil
}

func checkPlayer(raw json.RawMessage, id string) (string, error) {
	var p struct {
		PlayabilityStatus struct {
			Status string `json:"status"`
			Reason string `json:"reason"`
		} `json:"playabilityStatus"`
		VideoDetails struct {
			VideoID       string `json:"videoId"`
			Title         string `json:"title"`
			LengthSeconds string `json:"lengthSeconds"`
			ViewCount     string `json:"viewCount"`
			IsLive        bool   `json:"isLiveContent"`
		} `json:"videoDetails"`
		StreamingData struct {
			AdaptiveFormats []json.RawMessage `json:"adaptiveFormats"`
		} `json:"streamingData"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		return "", fmt.Errorf("player: %w", err)
	}
	switch {
	case p.PlayabilityStatus.Status != "OK":
		return "", fmt.Errorf("player: playability %s %q", p.PlayabilityStatus.Status, p.PlayabilityStatus.Reason)
	case p.VideoDetails.VideoID != id:
		return "", fmt.Errorf("player: videoId %q", p.VideoDetails.VideoID)
	case p.VideoDetails.Title == "":
		return "", errors.New("player: пустой title")
	case len(p.StreamingData.AdaptiveFormats) == 0:
		return "", errors.New("player: нет adaptiveFormats")
	}
	return fmt.Sprintf("%q %ss %s views live=%v formats=%d (player %d KB)", p.VideoDetails.Title,
		p.VideoDetails.LengthSeconds, p.VideoDetails.ViewCount, p.VideoDetails.IsLive,
		len(p.StreamingData.AdaptiveFormats), len(raw)/1024), nil
}

func checkWatchData(raw json.RawMessage) (string, error) {
	var d struct {
		Contents struct {
			Watch *struct {
				Results struct {
					Results struct {
						Contents []json.RawMessage `json:"contents"`
					} `json:"results"`
				} `json:"results"`
				Secondary struct {
					Results struct {
						Results []json.RawMessage `json:"results"`
					} `json:"secondaryResults"`
				} `json:"secondaryResults"`
			} `json:"twoColumnWatchNextResults"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return "", fmt.Errorf("data: %w", err)
	}
	w := d.Contents.Watch
	if w == nil || len(w.Results.Results.Contents) == 0 {
		return "", errors.New("data: нет twoColumnWatchNextResults")
	}
	return fmt.Sprintf("data: %d blocks, %d related (%d KB)",
		len(w.Results.Results.Contents), len(w.Secondary.Results.Results), len(raw)/1024), nil
}

func checkChannelData(raw json.RawMessage) (string, error) {
	var d struct {
		Metadata struct {
			Channel *struct {
				Title      string `json:"title"`
				ExternalID string `json:"externalId"`
			} `json:"channelMetadataRenderer"`
		} `json:"metadata"`
		Contents struct {
			Browse *struct {
				Tabs []json.RawMessage `json:"tabs"`
			} `json:"twoColumnBrowseResultsRenderer"`
		} `json:"contents"`
	}
	if err := json.Unmarshal(raw, &d); err != nil {
		return "", fmt.Errorf("data: %w", err)
	}
	c := d.Metadata.Channel
	switch {
	case c == nil || !strings.HasPrefix(c.ExternalID, "UC"):
		return "", errors.New("data: нет channelMetadataRenderer")
	case d.Contents.Browse == nil || len(d.Contents.Browse.Tabs) == 0:
		return "", errors.New("data: нет вкладок канала")
	}
	return fmt.Sprintf("%q %s, %d tabs (data %d KB)", c.Title, c.ExternalID, len(d.Contents.Browse.Tabs), len(raw)/1024), nil
}

func offsets(body []byte) string {
	pct := func(i int) float64 { return float64(i) * 100 / float64(len(body)) }
	parts := []string{fmt.Sprintf("body %d KB", len(body)/1024)}
	for _, marker := range []string{playerMarker, dataMarker} {
		start := bytes.Index(body, []byte(marker))
		if start < 0 {
			continue
		}
		end := bytes.Index(body[start:], []byte(endTag))
		if end < 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %.1f–%.1f%%", markerName(marker), pct(start), pct(start+end+len(endTag))))
	}
	return strings.Join(parts, " · ")
}

func markerName(marker string) string {
	if marker == playerMarker {
		return "player"
	}
	return "data"
}

func formatFails(fails map[string]int) string {
	var out []string
	for _, reason := range slices.Sorted(maps.Keys(fails)) {
		out = append(out, fmt.Sprintf("%s ×%d", reason, fails[reason]))
	}
	return strings.Join(out, "; ")
}

// cutter ищет каждый маркер и закрывающий его </script>, не пересканируя буфер целиком.
type cutter struct {
	markers    [][]byte
	start, end []int
}

func newCutter(markers []string) *cutter {
	c := &cutter{start: make([]int, len(markers)), end: make([]int, len(markers))}
	for i, m := range markers {
		c.markers = append(c.markers, []byte(m))
		c.start[i], c.end[i] = -1, -1
	}
	return c
}

// feed сообщает, закрыты ли все маркеры после добавления последних added байт.
func (c *cutter) feed(buf []byte, added int) bool {
	from := len(buf) - added
	done := true
	for i, marker := range c.markers {
		if c.start[i] < 0 {
			lo := max(0, from-len(marker))
			if j := bytes.Index(buf[lo:], marker); j >= 0 {
				c.start[i] = lo + j
			}
		}
		if c.start[i] >= 0 && c.end[i] < 0 {
			lo := max(c.start[i], from-len(endTag))
			if j := bytes.Index(buf[lo:], []byte(endTag)); j >= 0 {
				c.end[i] = lo + j + len(endTag)
			}
		}
		if c.end[i] < 0 {
			done = false
		}
	}
	return done
}

// withStreamWindow — тот же профиль, но с другим SETTINGS_INITIAL_WINDOW_SIZE.
// Меняет h2-отпечаток: Chrome TLS + нехромовое окно.
func withStreamWindow(p profiles.ClientProfile, win uint32) profiles.ClientProfile {
	settings := maps.Clone(p.GetSettings())
	settings[http2.SettingInitialWindowSize] = win
	return profiles.NewClientProfile(p.GetClientHelloId(), settings, p.GetSettingsOrder(),
		p.GetPseudoHeaderOrder(), p.GetConnectionFlow(), p.GetPriorities(), p.GetHeaderPriority(),
		p.GetStreamID(), p.GetAllowHTTP(), p.GetHttp3Settings(), p.GetHttp3SettingsOrder(),
		p.GetHttp3PriorityParam(), p.GetHttp3PseudoHeaderOrder(), p.GetHttp3SendGreaseFrames())
}

func chromeHeaders() http.Header {
	return http.Header{
		"sec-ch-ua":                 {`"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-user":            {"?1"},
		"sec-fetch-dest":            {"document"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"accept-language":           {"en-US,en;q=0.9"},
		"priority":                  {"u=0, i"},
		http.HeaderOrderKey: {
			"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "upgrade-insecure-requests",
			"user-agent", "accept", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-user",
			"sec-fetch-dest", "accept-encoding", "accept-language", "cookie", "priority",
		},
	}
}

func firefoxHeaders() http.Header {
	return http.Header{
		"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:148.0) Gecko/20100101 Firefox/148.0"},
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"accept-language":           {"en-US,en;q=0.5"},
		"accept-encoding":           {"gzip, deflate, br, zstd"},
		"upgrade-insecure-requests": {"1"},
		"sec-fetch-dest":            {"document"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-user":            {"?1"},
		"priority":                  {"u=0, i"},
		"te":                        {"trailers"},
		http.HeaderOrderKey: {
			"user-agent", "accept", "accept-language", "accept-encoding", "cookie",
			"upgrade-insecure-requests", "sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site",
			"sec-fetch-user", "priority", "te",
		},
	}
}
