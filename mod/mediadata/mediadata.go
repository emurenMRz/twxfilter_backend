package mediadata

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type MediaData struct {
	Id             string
	ParentUrl      string
	Type           string
	Url            string
	Timestamp      uint64
	Selected       bool
	Removed        bool
	DurationMillis uint
	VideoUrl       string
}

func (m *MediaData) DownloadMedia() (err error) {
	if m.Type == "video" {
		err = DownloadFile(m.VideoUrl)
		if err != nil {
			log.Println(err)
		}
	} else {
		imageUri, err := normalizeImageUrl(m.Url)
		if err != nil {
			log.Println(err)
		}
		err = DownloadFile(imageUri)
		if err != nil {
			log.Println(err)
		}
	}
	return
}

func ParseMediaData(datafile string) (media []MediaData, err error) {
	in, err := os.ReadFile(datafile)
	if err != nil {
		return
	}

	err = json.Unmarshal(in, &media)
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

func DownloadFile(targetUrl string) error {
	log.Println("URL: " + targetUrl)
	u, err := url.Parse(targetUrl)
	if err != nil {
		return err
	}

	pathSegments := strings.Split(u.Path, "/")
	filename := pathSegments[len(pathSegments)-1]

	res, err := http.Get(targetUrl)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	contentTypes := res.Header.Values("Content-Type")
	contentLengths := res.Header.Values("Content-Length")
	log.Println("Content-Type: " + strings.Join(contentTypes, "; "))
	log.Println("Content-Length: " + strings.Join(contentLengths, "; "))

	contentLength, err := strconv.ParseInt(contentLengths[0], 10, 64)
	if err != nil {
		return err
	}

	log.Println("Output: " + filename)
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	writtenSize, err := io.Copy(out, res.Body)
	if err != nil {
		return err
	}

	if writtenSize != contentLength {
		err = fmt.Errorf("download error: %d/%d", writtenSize, contentLength)
		return err
	}

	log.Printf("Complete")
	return nil
}
