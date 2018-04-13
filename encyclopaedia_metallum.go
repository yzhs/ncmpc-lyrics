package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// EncyclopediaMetallum downloads lyrics from the metal archives
type EncyclopaediaMetallum struct {
	baseURL   string
	searchURL string
	lyricsURL string
}

// Name of this source
func (em EncyclopaediaMetallum) Name() string {
	return "EncyclopaediaMetallum"
}

func (em *EncyclopaediaMetallum) searchForSong(artist string, title string) (songID int, success bool) {
	artist = strings.ToLower(artist)
	title = strings.ToLower(title)

	resp, err := http.Get(fmt.Sprintf(em.baseURL+em.searchURL, url.QueryEscape(title), url.QueryEscape(artist)))
	if err != nil {
		log.Warning(err)
		return 0, false
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Warning(err)
		return 0, false
	}

	var f interface{}
	err = json.Unmarshal(bodyBytes, &f)
	if err != nil {
		log.Warning(err)
		return 0, false
	}
	m := f.(map[string]interface{})
	aaData := m["aaData"].([]interface{})

	var id int
	for _, recordInterface := range aaData {
		record := recordInterface.([]interface{})
		showLyricsLink := record[len(record)-1]

		idString := strings.TrimPrefix(showLyricsLink.(string), "<a href=\"javascript:;\" id=\"lyricsLink_")
		endOfID := strings.Index(idString, "\"")
		if endOfID == -1 {
			log.Warning("Could not find '\"' in string", idString)
			continue
		}

		id, err = strconv.Atoi(idString[0:endOfID])
		if err != nil {
			log.Warning(err)
			continue
		}
		break
	}

	return id, true
}

func (em *EncyclopaediaMetallum) getLyricsFromURL(id int) (lyrics string, success bool) {
	resp, err := http.Get(fmt.Sprintf(em.baseURL+em.lyricsURL, id))
	if err != nil {
		log.Warning(err)
		return "", false
	}
	defer resp.Body.Close()

	// Select the correct song
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	body := strings.Replace(string(bodyBytes), "<br />", "", -1)

	// Strip tokens
	lyrics = strings.TrimSpace(body)

	return lyrics, lyrics != "<em>(lyrics not available)</em>"
}

// Fetch lyrics for the given song
func (em EncyclopaediaMetallum) Fetch(artist, title string) (lyrics string, success bool) {
	id, success := em.searchForSong(artist, title)
	if !success {
		return "", false
	}

	return em.getLyricsFromURL(id)
}
