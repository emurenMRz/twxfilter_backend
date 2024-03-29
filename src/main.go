package main

import (
	"flag"
	"log"
)

var fromFile string
var cacheDir string
var cachingMode bool

func init() {
	flag.StringVar(&fromFile, "f", "", "Start caching media from an export file")
	flag.StringVar(&cacheDir, "c", "", "Set cache-dir and start caching media")
	flag.BoolVar(&cachingMode, "caching", false, "Start caching media with default cache dir")
}

func main() {
	flag.Parse()

	if flag.NFlag() > 0 {
		if len(fromFile) > 0 {
			log.Println("Start caching media from file...: " + cacheDir)
			err := cacheFromFile(cacheDir, fromFile)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		if cachingMode || len(cacheDir) > 0 {
			log.Println("Start caching media...: " + cacheDir)
			err := cache(cacheDir)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		flag.Usage()
		return
	}

	log.Println("Start daemon...")
	err := daemon()
	if err != nil {
		log.Fatal(err)
	}
}
