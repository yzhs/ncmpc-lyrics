package main

import (
	"fmt"
	"os"
	"os/user"

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

// Source prodides an interface for fetching lyrics from somewhere
type Source interface {
	Name() string
	Fetch(artist, title string) (lyrics string, success bool)
}

var (
	homeDir  string
	backends = []Source{
		Local{},
		Darklyrics{baseURL: "http://www.darklyrics.com/"},
		LyricsWiki{format: "http://lyrics.wikia.com/api.php?action=lyrics&fmt=xml&func=getSong&artist=%s&song=%s"},
	}
)

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
