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

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, "Succeed")
	})

	cgi.Serve(router.Router)
	return
}
