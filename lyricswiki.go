package main

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// LyricsWiki downloads lyrics from lyricswiki.org
type LyricsWiki struct{ format string }

// Name of this source
func (lw LyricsWiki) Name() string {
	return "Lyrics Wiki"
}

func (lw *LyricsWiki) getURL(artist, title string) (songURL string, err error) {
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
	}

	if strings.Contains(artist, ",") || strings.Contains(title, ",") {
		return lw.getURL(strings.Replace(artist, ",", "", -1), strings.Replace(title, ",", "", -1))
	}
	return "", errors.New("Could not determine URL")
}

// Fetch lyrics for this song
func (lw LyricsWiki) Fetch(artist, title string) (lyrics string, success bool) {
	url, err := lw.getURL(artist, title)
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
