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
	"router"
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
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nodeRoot := [][]mediadata.MediaData{}
		for _, mediaSet := range duplicatedMediaList {
			set := []mediadata.MediaData{}
			for _, media := range mediaSet {
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
			nodeRoot = append(nodeRoot, set)
		}

		o, err := json.Marshal(nodeRoot)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(o))
	})

	router.RegistorEndpoint("GET /"+selfName+"/media/:id", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		id := values["id"]

		mediaRecord, err := conn.GetMediaByID(id)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, err)
			return
		}

		if !mediaRecord.CachePath.Valid {
			err := "No cache"
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, err)
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
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintln(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(o))
	})

	router.RegistorEndpoint("POST /"+selfName+"/media", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, err)
			return
		}

		media, err := mediadata.ParseMediaData(body)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, err)
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
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				fmt.Fprint(w, err)
				return
			}
		}

		mediaList, err := conn.GetMedia()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, err)
			return
		}

		lines := []mediadata.MediaData{}
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
			lines = append(lines, m)
		}

		o, err := json.Marshal(lines)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintln(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(o))
	})

	router.RegistorEndpoint("DELETE /"+selfName+"/media", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		err := conn.DeleteMediaAll()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	router.RegistorEndpoint("DELETE /"+selfName+"/media/cached", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		err := conn.DeleteMediaCached()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	router.RegistorEndpoint("DELETE /"+selfName+"/media/:id", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		id := values["id"]

		err := conn.DeleteMedia(id)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprint(w, err)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	router.RegistorEndpoint("DELETE /"+selfName+"/cache-file/:id", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		id := values["id"]

		mediaRecord, err := conn.GetMediaByID(id)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		if !mediaRecord.CachePath.Valid {
			log.Println(err)
			http.Error(w, "No content", http.StatusNoContent)
			return
		}

		cachePath := mediaRecord.CachePath.String
		mediaType := mediaRecord.Type
		err = mediadata.DeleteCacheFile(cachePath, mediaType)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to delete file", http.StatusInternalServerError)
			return
		}

		err = conn.DeleteCacheFile(id)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	cgi.Serve(router.Router)
	return
}
