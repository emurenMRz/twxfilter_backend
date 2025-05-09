package main

import (
	"database/sql"
	"diffhash"
	"fmt"
	"log"
	"mediadata"
	"os"
	"path"
	"time"
)

func cache(cacheDir string) (err error) {
	runData, err := RunDaemon("caching.pid")
	if err != nil {
		return
	}
	defer runData.Close()

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
			if merr, ok := err.(*mediadata.MediaError); ok {
				if merr.IsNotFound() {
					err = conn.DeleteMedia(m.Id)
				}
			}
			log.Println(err)
			continue
		}
		err = conn.SetCacheData(m.Id, cacheData.ContentLength, cacheData.ContentHash, cacheData.CachePath)
		if err != nil {
			log.Println(err)
		}
		err = conn.SetThumbnail(m.Id, cacheData.Thumbnail)
		if err != nil {
			log.Println(err)
		}
	}

	return
}

func cacheFromFile(cacheDir string, fromFile string) (err error) {
	runData, err := RunDaemon("caching.pid")
	if err != nil {
		return
	}
	defer runData.Close()

	if fromFile == "" {
		fromFile = "twfilter-all-data.json"
	}

	_, err = os.Stat(fromFile)
	if err != nil {
		return
	}

	media, err := mediadata.ParseMediaDataFromFile(fromFile)
	if err != nil {
		return
	}

	baseDir, err := makeBaseDir(cacheDir)
	if err != nil {
		return
	}

	for _, m := range media {
		_, err = m.DownloadMedia(baseDir)
		if err != nil {
			log.Println(err)
		}
	}

	return
}

func createThumbnails(cacheDir string) (err error) {
	runData, err := RunDaemon("caching.pid")
	if err != nil {
		return
	}
	defer runData.Close()

	conn, err := GetConnection()
	if err != nil {
		return
	}
	defer conn.Close()

	cachedVideoMediaList, err := conn.GetCachedVideoMedia()
	if err != nil {
		return
	}

	if len(cachedVideoMediaList) == 0 {
		err = fmt.Errorf("no cached video media")
		return
	}

	_, err = makeBaseDir(cacheDir)
	if err != nil {
		return
	}

	for _, cachedVideoMedia := range cachedVideoMediaList {
		thumbnail, err := mediadata.MakeThumbnail(cachedVideoMedia.path, 0)
		if err != nil {
			log.Println(err)
			continue
		}
		if err = conn.SetThumbnail(cachedVideoMedia.id, thumbnail); err != nil {
			log.Println(err)
			continue
		}
		log.Println("Thumbnail created: " + cachedVideoMedia.path)
	}

	return
}

func calculateDiffHashs() (err error) {
	runData, err := RunDaemon("caching.pid")
	if err != nil {
		return
	}
	defer runData.Close()

	conn, err := GetConnection()
	if err != nil {
		return
	}
	defer conn.Close()

	unhashedMediaList, err := conn.GetUnhashedMedia()
	if err != nil {
		return
	}

	if len(unhashedMediaList) == 0 {
		err = fmt.Errorf("no cached media")
		return
	}

	for _, unhashedMedia := range unhashedMediaList {
		contentHash := diffhash.CalcDiffHashFromImage(unhashedMedia.Thumbnail)
		log.Printf("Diff-hashed: %s %016x\n", unhashedMedia.MediaId, contentHash)
		conn.SetContentHashData(unhashedMedia.MediaId, contentHash)
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

	os.Chmod(baseDir, 0777)

	return
}
