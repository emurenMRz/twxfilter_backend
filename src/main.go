package main

import (
	"flag"
	"log"

	"mediadata"
)

func main() {
	daemonMode := false
	flag.BoolVar(&daemonMode, "d", false, "Standby in daemon mode")

	flag.Parse()
	if len(flag.Args()) == 0 {
		daemonMode = true
	} else {
		flag.Usage()
		return
	}

	if daemonMode {
		log.Println("Start daemon...")
		daemon()
		return
	}

	inputFile := flag.Arg(0)

	media, err := mediadata.ParseMediaDataFromFile(inputFile)
	if err != nil {
		log.Fatal(err)
	}

	for _, m := range media {
		err = m.DownloadMedia()
		if err != nil {
			log.Println(err)
		}
	}
}
