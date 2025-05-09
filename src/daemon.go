package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mediadata"
	"net/http"
	"net/http/cgi"
	"reflect"
	"router"
	"strings"
)

func daemon() (err error) {
	conn, err := GetConnection()
	if err != nil {
		return
	}
	defer conn.Close()

	selfName := GetSelfName()

	router.RegistorEndpoint("GET /"+selfName+"/media/duplicated", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		duplicatedMediaList, err := conn.GetDuplicatedMedia()
		if err != nil {
			handleError(w, err)
			return
		}

		o, err := mediaRecordSetListToJson(duplicatedMediaList)
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, o)
	})

	router.RegistorEndpoint("POST /"+selfName+"/media/duplicated", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, err)
			return
		}

		mediaObjectSetList := [][]any{}
		err = json.Unmarshal(body, &mediaObjectSetList)
		if err != nil {
			handleError(w, err)
			return
		}

		mediaSetList := [][]map[string]any{}
		for _, setList := range mediaObjectSetList {
			condition := []string{}
			args := []any{}
			for i, set := range setList {
				if v, ok := set.(string); ok {
					condition = append(condition, fmt.Sprintf(`cache_path LIKE $%d`, i+1))
					args = append(args, fmt.Sprintf(`%%%s%%`, v))
				} else if _, ok := set.(map[string]string); ok {
					log.Printf("no implement: map[string]string\n")
				} else {
					log.Printf("unkown type: %s\n", reflect.TypeOf(set))
				}
			}
			mediaSet, err := conn.GetMediaDataSet(strings.Join(condition, " OR "), args...)
			if err != nil {
				handleError(w, err)
				return
			}
			if len(mediaSet) >= 2 {
				mediaSetList = append(mediaSetList, mediaSet)
			}
		}

		o, err := mediaRecordSetListToJson(mediaSetList)
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, o)
	})

	router.RegistorEndpoint("GET /"+selfName+"/media/:id", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		id := values["id"]

		mediaRecord, err := conn.GetMediaByID(id)
		if err != nil {
			handleError(w, err)
			return
		}

		if !mediaRecord.CachePath.Valid {
			handleError(w, fmt.Errorf("no cache"))
			return
		}

		type MediaObject struct {
			MediaId        string `json:"id"`
			ParentUrl      string `json:"parentUrl"`
			Type           string `json:"type"`
			Url            string `json:"url"`
			Timestamp      uint64 `json:"timestamp"`
			DurationMillis int32  `json:"durationMillis,omitempty"`
			VideoUrl       string `json:"videoUrl,omitempty"`
			ContentLength  int64  `json:"contentLength,omitempty"`
			CachePath      string `json:"cachePath,omitempty"`
			Removed        bool   `json:"removed"`
		}

		m := MediaObject{
			MediaId:   mediaRecord.MediaId,
			ParentUrl: mediaRecord.ParentUrl,
			Type:      mediaRecord.Type,
			Url:       mediaRecord.Url,
			Timestamp: mediaRecord.Timestamp,
			Removed:   mediaRecord.Removed,
		}

		if mediaRecord.DurationMillis.Valid {
			m.DurationMillis = mediaRecord.DurationMillis.Int32
		}
		if mediaRecord.VideoUrl.Valid {
			m.VideoUrl = mediaRecord.VideoUrl.String
		}
		if mediaRecord.ContentLength.Valid {
			m.ContentLength = mediaRecord.ContentLength.Int64
		}

		o, err := json.Marshal(m)
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(o))
	})

	router.RegistorEndpoint("GET /"+selfName+"/thumbnail/:id", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		id := values["id"]

		thumbnail, err := conn.GetThumbnailByID(id)
		if err != nil {
			handleError(w, err)
			return
		}

		if len(thumbnail) == 0 {
			handleError(w, fmt.Errorf("no thumbnail"))
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		w.Write(thumbnail)
	})

	router.RegistorEndpoint("POST /"+selfName+"/media", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			handleError(w, err)
			return
		}

		media, err := mediadata.ParseMediaData(body)
		if err != nil {
			handleError(w, err)
			return
		}

		columns := []string{"media_id", "parent_url", "type", "url", "timestamp", "duration_millis", "video_url"}
		var valueTable [][]any
		for _, m := range media {
			row := []any{
				m.Id,
				m.ParentUrl,
				m.Type,
				m.Url,
				m.Timestamp,
				m.DurationMillis,
				m.VideoUrl,
			}
			if row[5] == 0 {
				row[5] = nil
			}
			if row[6] == "" {
				row[6] = nil
			}

			valueTable = append(valueTable, row)
		}

		if len(valueTable) > 0 {
			err = conn.UpsertMedia(columns, valueTable)
			if err != nil {
				handleError(w, err)
				return
			}
		}

		mediaList, err := conn.GetMedia()
		if err != nil {
			handleError(w, err)
			return
		}

		lines := mediaRecordSet(mediaList)
		o, err := json.Marshal(lines)
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(o))
	})

	router.RegistorEndpoint("DELETE /"+selfName+"/media", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		err := conn.DeleteMediaAll()
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	router.RegistorEndpoint("DELETE /"+selfName+"/media/cached", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		err := conn.DeleteMediaCached()
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	router.RegistorEndpoint("DELETE /"+selfName+"/media/:id", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		id := values["id"]

		err := conn.DeleteMedia(id)
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	router.RegistorEndpoint("DELETE /"+selfName+"/cache-file/:id", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		id := values["id"]

		err := deleteCacheFileCore(conn, id)
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	cgi.Serve(router.Router)
	return
}

func handleError(w http.ResponseWriter, err error) {
	log.Println(err)

	msg := err.Error()
	code := http.StatusInternalServerError

	if cerr, ok := err.(*DaemonError); ok {
		msg = cerr.Text()
		code = cerr.Code()
	}

	http.Error(w, msg, code)
}

func deleteCacheFileCore(conn *Database, id string) error {
	mediaRecord, err := conn.GetMediaByID(id)
	if err != nil {
		return NewDaemonError(err, http.StatusNotFound, "")
	}

	if !mediaRecord.CachePath.Valid {
		return NewDaemonError(nil, http.StatusNoContent, "No content")
	}

	cachePath := mediaRecord.CachePath.String
	mediaType := mediaRecord.Type
	err = mediadata.DeleteCacheFile(cachePath, mediaType)
	if err != nil {
		return NewDaemonError(err, 0, "Failed to delete file")
	}

	err = conn.DeleteCacheFile(id)
	if err != nil {
		return NewDaemonError(err, 0, "")
	}

	return nil
}

func mediaRecordSet(mediaList []map[string]any) []mediadata.MediaData {
	set := []mediadata.MediaData{}
	for _, media := range mediaList {
		m := mediadata.MediaData{
			Id:        media["mediaId"].(string),
			ParentUrl: media["parentUrl"].(string),
			Type:      media["type"].(string),
			Url:       media["url"].(string),
			Timestamp: media["timestamp"].(uint64),
			HasCache:  media["hasCache"].(bool),
		}
		durationMillis := media["durationMillis"].(sql.NullInt32)
		if durationMillis.Valid {
			m.DurationMillis = uint(durationMillis.Int32)
		}
		videoUrl := media["videoUrl"].(sql.NullString)
		if videoUrl.Valid {
			m.VideoUrl = videoUrl.String
		}
		mediaPath := media["mediaPath"].(sql.NullString)
		if mediaPath.Valid {
			m.MediaPath = mediaPath.String
		}
		thumbPath := media["thumbPath"].(sql.NullString)
		if thumbPath.Valid {
			m.ThumbPath = thumbPath.String
		}
		set = append(set, m)
	}
	return set
}

func mediaRecordSetListToJson(duplicatedMediaList [][]map[string]any) (string, error) {
	nodeRoot := [][]mediadata.MediaData{}
	for _, mediaSet := range duplicatedMediaList {
		nodeRoot = append(nodeRoot, mediaRecordSet(mediaSet))
	}

	o, err := json.Marshal(nodeRoot)
	if err != nil {
		return "", err
	}

	return string(o), nil
}

func DeleteCacheFile(id string) error {
	conn, err := GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return deleteCacheFileCore(conn, id)
}
