package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Darklyrics downloads lyrics from darklyrics.com
type Darklyrics struct{ baseURL string }

// Name of this source
func (dl Darklyrics) Name() string {
	return "Darklyrics"
}

func (dl *Darklyrics) searchForSong(artist string, title string) (songURL string, songIDOnPage string, success bool) {
	artist = strings.ToLower(artist)
	title = strings.ToLower(title)

	doc, err := goquery.NewDocument(fmt.Sprintf(dl.baseURL+"search?q=%s+%s", url.QueryEscape(artist), url.QueryEscape(title)))
	if err != nil {
		log.Warning(err)
		return "", "", false
	}

	// Go straight to the links
	doc.Find("div.sen > h2 > a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		var urlFound bool
		songURL, urlFound = s.Attr("href")
		if !urlFound {
			log.Warning("Not a link")
			return true
		}
		tmp := strings.Split(songURL, "#")
		if len(tmp) != 2 {
			return true
		}
		songURL = tmp[0]
		songIDOnPage = tmp[1]

		return false
	})

	return songURL, songIDOnPage, songURL != ""
}

func (dl *Darklyrics) getLyricsFromURL(address string, id string) (lyrics string, success bool) {
	resp, err := http.Get(dl.baseURL + address)
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

// Fetch lyrics for the given song
func (dl Darklyrics) Fetch(artist, title string) (lyrics string, success bool) {
	URL, id, success := darklyrics.searchForSong(artist, title)
	if !success {
		return "", false
	}

	return dl.getLyricsFromURL(URL, id)
}
