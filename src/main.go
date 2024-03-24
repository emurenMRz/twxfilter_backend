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
		flag.Usage()
		return
	}

	if daemonMode {
		// No implement.
		return
	}

	inputFile := flag.Arg(0)

	media, err := mediadata.ParseMediaData(inputFile)
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
