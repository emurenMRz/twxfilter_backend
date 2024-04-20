package mediadata

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

type MediaData struct {
	Id             string `json:"id"`
	ParentUrl      string `json:"parentUrl"`
	Type           string `json:"type"`
	Url            string `json:"url"`
	Timestamp      uint64 `json:"timestamp"`
	Selected       bool   `json:"selected"`
	DurationMillis uint   `json:"durationMillis,omitempty"`
	VideoUrl       string `json:"videoUrl,omitempty"`
}

type CacheData struct {
	ContentLength uint64
	CachePath     string
}

func (m *MediaData) DownloadMedia(baseDir string) (cacheData CacheData, err error) {
	if m.Type == "video" || m.Type == "animated_gif" {
		cacheData, err = DownloadFile(baseDir, m.VideoUrl)
		return
	}

	imageUri, err := normalizeImageUrl(m.Url)
	if err != nil {
		return
	}

	cacheData, err = DownloadFile(baseDir, imageUri)
	return
}

func ParseMediaData(jsonData []byte) (media []MediaData, err error) {
	err = json.Unmarshal(jsonData, &media)
	return
}

func ParseMediaDataFromFile(datafile string) (media []MediaData, err error) {
	in, err := os.ReadFile(datafile)
	if err != nil {
		return
	}

	media, err = ParseMediaData(in)
	return
}

func ParseRegexp(reg string, mediaUrl string) map[string]string {
	pattern := regexp.MustCompile(reg)
	names := pattern.SubexpNames()
	tokenMap := map[string]string{}
	for i, m := range pattern.FindStringSubmatch(mediaUrl) {
		name := names[i]
		if name == "" {
			name = "*"
		}
		tokenMap[name] = m
	}
	return tokenMap
}

func normalizeImageUrl(mediaUrl string) (imageUrl string, err error) {
	m := ParseRegexp(`^https://pbs\.twimg\.com/media/(?P<filename>.+?)\?format=(?P<extension>.+)&name=.+$`, mediaUrl)
	if len(m) < 2 {
		m = ParseRegexp(`^https://pbs\.twimg\.com/media/(?P<filename>[^\.]+?)\.(?P<extension>.+)$`, mediaUrl)
		if len(m) < 2 {
			err = fmt.Errorf("failed parse url: " + mediaUrl)
			return
		}
	}
	ext := m["extension"]
	filename := m["filename"] + "." + ext
	imageUrl = "https://pbs.twimg.com/media/" + filename + "?format=" + ext + "&name=orig"
	return
}

func DownloadFile(baseDir string, targetUrl string) (cacheData CacheData, err error) {
	log.Println("URL: " + targetUrl)
	u, err := url.Parse(targetUrl)
	if err != nil {
		return
	}

	pathSegments := strings.Split(u.Path, "/")
	filename := pathSegments[len(pathSegments)-1]

	res, err := http.Get(targetUrl)
	if err != nil {
		return
	}
	defer res.Body.Close()

	contentTypes := res.Header.Values("Content-Type")
	if len(contentTypes) == 0 {
		err = fmt.Errorf("no Content-type is obtained")
		return
	}
	log.Println("Content-Type: " + strings.Join(contentTypes, "; "))

	contentLengths := res.Header.Values("Content-Length")
	if len(contentLengths) == 0 {
		err = fmt.Errorf("no Content-length is obtained")
		return
	}
	log.Println("Content-Length: " + strings.Join(contentLengths, "; "))

	size, err := strconv.ParseUint(contentLengths[0], 10, 64)
	if err != nil {
		return
	}

	if size == 0 {
		cacheData.ContentLength = size

		log.Println("Not cached: content-length is zero")
		return
	}

	outputPath := path.Join(baseDir, filename)

	log.Println("Output: " + outputPath)
	out, err := os.Create(outputPath)
	if err != nil {
		return
	}
	defer out.Close()

	writtenSize, err := io.Copy(out, res.Body)
	if err != nil {
		return
	}

	if uint64(writtenSize) != size {
		err = fmt.Errorf("download error: %d/%d", writtenSize, size)
		return
	}

	cacheData.ContentLength = size
	cacheData.CachePath = outputPath

	log.Println("Complete")
	return
}
