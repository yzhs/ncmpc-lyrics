package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"strings"
	//"text/template"

	"github.com/PuerkitoBio/goquery"
)

const baseUrl = "http://www.darklyrics.com/"

var homeDir string

func usage() {
	fmt.Println("Usage: darklyrics [artist] [title]")
}

func searchForSong(artist string, title string) (songUrl string, songIdOnPage string, success bool) {
	artist = strings.ToLower(artist)
	title = strings.ToLower(title)

	doc, err := goquery.NewDocument(fmt.Sprintf(baseUrl+"search?q=%s+%s", url.QueryEscape(artist), url.QueryEscape(title)))
	if err != nil {
		log.Panic(err)
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

func getLyricsFromUrl(address string, id string) (lyrics string, success bool) {
	resp, err := http.Get(baseUrl + address)
	if err != nil {
		log.Panic(err)
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

	URL, id, success := searchForSong(artist, title)
	if !success {
		os.Exit(1)
	}

	text, success := getLyricsFromUrl(URL, id)
	if !success {
		os.Exit(1)
	}
	fmt.Println(text)
}
