package main

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
)

type Local struct{}

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
