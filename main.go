package main

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"
	//"text/template"

	"github.com/PuerkitoBio/goquery"
)

func usage() {
	fmt.Println("Usage: fetch [artist] [title]")
}

type LyricsFetcher interface {
	Fetch(artist, title string) (lyrics string, success bool)
}

type Local struct {
}

type Darklyrics struct {
	baseUrl string
}

type LyricsWiki struct {
	format string
}

var (
	homeDir    string
	local      = Local{}
	darklyrics = Darklyrics{baseUrl: "http://www.darklyrics.com/"}
	lyricswiki = LyricsWiki{format: "http://lyrics.wikia.com/api.php?action=lyrics&fmt=xml&func=getSong&artist=%s&song=%s"}
)

func (l Local) Fetch(artist string, title string) (lyrics string, success bool) {
	safeArtist := strings.Replace(artist, "/", "_", -1)
	safeTitle := strings.Replace(title, "/", "_", -1)
	lyricsPath := path.Join(homeDir, ".lyrics", fmt.Sprintf("%s - %s.txt", safeArtist, safeTitle))
	bytes, err := ioutil.ReadFile(lyricsPath)
	if err != nil {
		log.Panic(err)
		return "", false
	}

	return string(bytes), true
}

func (dl *Darklyrics) searchForSong(artist string, title string) (songUrl string, songIdOnPage string, success bool) {
	artist = strings.ToLower(artist)
	title = strings.ToLower(title)

	doc, err := goquery.NewDocument(fmt.Sprintf(dl.baseUrl+"search?q=%s+%s", url.QueryEscape(artist), url.QueryEscape(title)))
	if err != nil {
		log.Panic(err)
		return "", "", false
	}

	// Go straight to the links
	doc.Find("div.sen > h2 > a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		//linkText := strings.TrimSpace(strings.ToLower(s.Text()))
		//if !strings.HasPrefix(linkText, artist) || !strings.HasSuffix(linkText, title) {
		//	//log.Println("Wrong song: ", linkText)
		//	//return true
		//}
		var urlFound bool
		songUrl, urlFound = s.Attr("href")
		if !urlFound {
			log.Println("Not a link")
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
		log.Println(err)
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
		log.Panic(err)
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

func (lw *LyricsWiki) getUrl(artist, title string) (songUrl string, err error) {
	resp, err := http.Get(fmt.Sprintf(lw.format, artist, title))
	if err != nil {
		log.Println(err)
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.Panic(err)
		return "", err
	}

	return doc.Find("url").Text(), nil
}

func (lw LyricsWiki) Fetch(artist, title string) (lyrics string, success bool) {
	url, err := lw.getUrl(artist, title)
	if err != nil {
		log.Panic(err)
		return "", false
	}

	resp, err := http.Get(url)
	if err != nil {
		log.Panic(err)
		return "", false
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.Panic(err)
		return "", false
	}

	content, err := doc.Find("div.lyricbox").Html()
	if err != nil {
		return "", false
	}
	content = html.UnescapeString(content)
	content = strings.Replace(content, "<br/>", "\n", -1)
	content = strings.Replace(content, "<div class=\"lyricsbreak\"></div>", "", -1)

	return content, true
}

func main() {
	if len(os.Args) != 3 || (len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help")) {
		usage()
		os.Exit(0)
	}
	u, err := user.Current()
	if err != nil {
		log.Panic(err)
	}
	homeDir = u.HomeDir

	artist := os.Args[1]
	title := os.Args[2]

	var backend LyricsFetcher

	if strings.Contains(os.Args[0], "-darklyrics") {
		backend = darklyrics
	} else if strings.Contains(os.Args[0], "-hd") {
		backend = local
	} else {
		backend = lyricswiki
	}

	text, success := backend.Fetch(artist, title)
	if !success {
		os.Exit(1)
	}
	fmt.Println(text)
}
