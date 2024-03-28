package main

import (
	"database/sql"
	"fmt"
	"log"
	"mediadata"
	"os"
	"path"
	"time"
)

func cache(cacheDir string) {
	conn, err := GetConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	mediaList, err := conn.GetNoCacheMedia()
	if err != nil {
		log.Fatal(err)
	}

	lines := []mediadata.MediaData{}
	for _, media := range mediaList {
		m := mediadata.MediaData{
			Id:   media["mediaId"].(string),
			Type: media["type"].(string),
			Url:  media["url"].(string),
		}
		videoUrl := media["videoUrl"].(sql.NullString)
		if videoUrl.Valid {
			m.VideoUrl = videoUrl.String
		}
		lines = append(lines, m)
	}

	year, month, day := time.Now().Date()
	baseDir := path.Join(cacheDir, fmt.Sprintf("%04d%02d%02d", year, int(month), day))
	err = os.MkdirAll(baseDir, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) {
			log.Fatal(err)
		}
	}

	for _, m := range lines {
		cacheData, err := m.DownloadMedia(baseDir)
		if err != nil {
			log.Println(err)
			continue
		}
		conn.SetCacheData(m.Id, cacheData.ContentLength, cacheData.CachePath)
	}
}
