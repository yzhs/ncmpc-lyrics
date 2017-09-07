package main

import (
	"errors"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/op/go-logging"
)

func usage() {
	fmt.Println("Usage: fetch [artist] [title]")
}

var (
	log    = logging.MustGetLogger("fetch")
	format = logging.MustStringFormatter(
		`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
	)
)

type LyricsFetcher interface {
	Name() string
	Fetch(artist, title string) (lyrics string, success bool)
}

type Local struct{}

type Darklyrics struct{ baseUrl string }

type LyricsWiki struct{ format string }

var (
	homeDir    string
	local      = Local{}
	darklyrics = Darklyrics{baseUrl: "http://www.darklyrics.com/"}
	lyricswiki = LyricsWiki{format: "http://lyrics.wikia.com/api.php?action=lyrics&fmt=xml&func=getSong&artist=%s&song=%s"}
)

func (l Local) Name() string {
	return "Local"
}

func (l Local) Fetch(artist string, title string) (lyrics string, success bool) {
	safeArtist := strings.Replace(artist, "/", "_", -1)
	safeTitle := strings.Replace(title, "/", "_", -1)
	lyricsPath := path.Join(homeDir, ".lyrics", fmt.Sprintf("%s - %s.txt", safeArtist, safeTitle))
	bytes, err := ioutil.ReadFile(lyricsPath)
	if err != nil {
		log.Debug(err)
		return "", false
	}

	return string(bytes), true
}

func (l Darklyrics) Name() string {
	return "Darklyrics"
}

func (dl *Darklyrics) searchForSong(artist string, title string) (songUrl string, songIdOnPage string, success bool) {
	artist = strings.ToLower(artist)
	title = strings.ToLower(title)

	doc, err := goquery.NewDocument(fmt.Sprintf(dl.baseUrl+"search?q=%s+%s", url.QueryEscape(artist), url.QueryEscape(title)))
	if err != nil {
		log.Warning(err)
		return "", "", false
	}

	// Go straight to the links
	doc.Find("div.sen > h2 > a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		var urlFound bool
		songUrl, urlFound = s.Attr("href")
		if !urlFound {
			log.Warning("Not a link")
			return true
		}
		tmp := strings.Split(songUrl, "#")
		if len(tmp) != 2 {
			return true
		}
		songUrl = tmp[0]
		songIdOnPage = tmp[1]

		return false
	})

	return songUrl, songIdOnPage, songUrl != ""
}

func (dl *Darklyrics) getLyricsFromUrl(address string, id string) (lyrics string, success bool) {
	resp, err := http.Get(dl.baseUrl + address)
	if err != nil {
		log.Warning(err)
		return "", false
	}
	defer resp.Body.Close()

	// Select the correct song
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	body := string(bodyBytes)
	i := strings.Index(body, "<h3><a name=\""+id+"\">")
	if i < 0 || i > len(body) {
		return "", false
	}
	body = body[i:]
	i = strings.Index(body, "<br />\n") + len("<br />\n")
	if i < 0 || i > len(body) {
		return "", false
	}
	body = body[i:]
	i = strings.Index(body, "<h3>")
	if i < 0 || i > len(body) {
		return "", false
	}
	body = body[:i]

	// Strip html tags
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(body))
	if err != nil {
		log.Warning(err)
		return "", false
	}

	// Strip tokens
	return strings.TrimSpace(doc.Text()), true
}

func (dl Darklyrics) Fetch(artist, title string) (lyrics string, success bool) {
	URL, id, success := darklyrics.searchForSong(artist, title)
	if !success {
		return "", false
	}

	return dl.getLyricsFromUrl(URL, id)
}

func (l LyricsWiki) Name() string {
	return "Lyrics Wiki"
}

func (lw *LyricsWiki) getUrl(artist, title string) (songUrl string, err error) {
	artist = strings.Replace(artist, "’", "'", -1)
	title = strings.Replace(title, "’", "'", -1)
	resp, err := http.Get(fmt.Sprintf(lw.format, url.QueryEscape(artist), url.QueryEscape(title)))
	if err != nil {
		log.Warning(err)
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.Warning(err)
		return "", err
	}

	text := doc.Find("url").Text()
	if text != "" {
		return text, nil
	} else {
		return "", errors.New("Could not determine URL")
	}
}

func (lw LyricsWiki) Fetch(artist, title string) (lyrics string, success bool) {
	url, err := lw.getUrl(artist, title)
	if err != nil {
		log.Warning(err)
		return "", false
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Warning(err)
		return "", false
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.Warning(err)
		return "", false
	}

	content, err := doc.Find("div.lyricbox").Html()
	if err != nil {
		log.Warning(err)
		return "", false
	}
	content = html.UnescapeString(content)
	if strings.Contains(content, "Category:Instrumental") {
		return "[Instrumental]", true
	}
	content = strings.Replace(content, "<br/>", "\n", -1)
	content = strings.Replace(content, "<div class=\"lyricsbreak\"></div>", "", -1)

	return content, true
}

func main() {
	if len(os.Args) != 3 || (len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
		usage()
		os.Exit(0)
	}
	logging.SetFormatter(format)
	logging.SetLevel(logging.INFO, "fetch")

	u, err := user.Current()
	if err != nil {
		log.Panic(err)
	}
	homeDir = u.HomeDir

	artist := os.Args[1]
	title := os.Args[2]

	backends := []LyricsFetcher{local, darklyrics, lyricswiki}

	for _, backend := range backends {
		text, success := backend.Fetch(artist, title)
		if success {
			fmt.Println(text)
			os.Exit(0)
		}
		log.Info("Could not find lyrics using backend", backend.Name())
	}

	os.Exit(69)
}
