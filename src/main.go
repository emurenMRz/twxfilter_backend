package main

import (
	"flag"
	"log"
)

var fromFile string
var cacheDir string
var deleteCacheFile string
var cachingMode bool
var makeThumbnailMode bool
var calcDiffHashMode bool

func init() {
	flag.StringVar(&fromFile, "f", "", "Start caching media from an export file")
	flag.StringVar(&cacheDir, "c", "", "Set cache-dir and start caching media")
	flag.StringVar(&deleteCacheFile, "delete-cache", "", "Delete media cache files")
	flag.BoolVar(&cachingMode, "caching", false, "Start caching media with default cache dir")
	flag.BoolVar(&makeThumbnailMode, "make-thumbnails", false, "Start creating thumbnails for video media")
	flag.BoolVar(&calcDiffHashMode, "calc-diffhash", false, "Starts calculating the media difference hash")
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

		if len(deleteCacheFile) > 0 {
			err := DeleteCacheFile(deleteCacheFile)
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

		if makeThumbnailMode || len(cacheDir) > 0 {
			log.Println("Start creating video thumbnails...: " + cacheDir)
			err := createThumbnails(cacheDir)
			if err != nil {
				log.Fatal(err)
			}
			return
		}

		if calcDiffHashMode || len(cacheDir) > 0 {
			log.Println("Starts calculating the media difference hash...: " + cacheDir)
			err := calculateDiffHashs(cacheDir)
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
