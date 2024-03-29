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

func cache(cacheDir string) (err error) {
	conn, err := GetConnection()
	if err != nil {
		return
	}
	defer conn.Close()

	mediaList, err := conn.GetNoCacheMedia()
	if err != nil {
		return
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

	baseDir, err := makeBaseDir(cacheDir)
	if err != nil {
		return
	}

	for _, m := range lines {
		cacheData, err := m.DownloadMedia(baseDir)
		if err != nil {
			log.Println(err)
			continue
		}
		conn.SetCacheData(m.Id, cacheData.ContentLength, cacheData.CachePath)
	}

	return
}


func makeBaseDir(cacheDir string) (baseDir string, err error) {
	if cacheDir == "" {
		cacheDir, err = ExecPath(".cache")
		if err != nil {
			return
		}
	}

	year, month, day := time.Now().Date()
	baseDir = path.Join(cacheDir, fmt.Sprintf("%04d%02d%02d", year, int(month), day))
	err = os.MkdirAll(baseDir, os.ModePerm)
	if err != nil {
		if !os.IsExist(err) {
			return
		}
		err = nil
	}

	return
}
