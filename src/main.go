package main

import (
	"flag"
	"log"
	"os"

	"mediadata"
)

var daemonMode bool
var cacheDir string

func init() {
	flag.BoolVar(&daemonMode, "d", false, "Standby in daemon mode")
	flag.StringVar(&cacheDir, "c", ".cache", "Set cache-dir and start caching media")
}

func main() {
	flag.Parse()

	if len(cacheDir) > 0 {
		log.Println("Start caching media...: " + cacheDir)
		err := cache(cacheDir)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if daemonMode || len(flag.Args()) == 0 {
		log.Println("Start daemon...")
		err := daemon()
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	inputFile := flag.Arg(0)

	_, err := os.Stat(inputFile)
	if err != nil {
		flag.Usage()
		return
	}

	media, err := mediadata.ParseMediaDataFromFile(inputFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range media {
		_, err = m.DownloadMedia("")
		if err != nil {
			log.Println(err)
		}
	}
}
