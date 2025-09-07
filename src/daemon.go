package main

import (
	"datasource"
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

	"github.com/emurenMRz/twxfilter_backend/internal/mapper"
)

func daemon() (err error) {
	conn, err := GetConnection()
	if err != nil {
		return
	}
	defer conn.Close()

	selfName := datasource.GetSelfName()

	router.RegistorEndpoint("GET /"+selfName+"/media/duplicated", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		duplicatedMediaList, err := conn.GetHashCluster()
		if err != nil {
			handleError(w, err)
			return
		}

		o, err := json.Marshal(mapper.MediaRecordSetListToMediaDataSetList(duplicatedMediaList))
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(o))
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

		mediaSetList := [][]datasource.MediaRecord{}
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

		o, err := json.Marshal(mapper.MediaRecordSetListToMediaDataSetList(mediaSetList))
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(o))
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

	router.RegistorEndpoint("GET /"+selfName+"/catalog/index", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		dates, err := conn.GetCatalogIndex()
		if err != nil {
			handleError(w, err)
			return
		}

		o, err := json.Marshal(dates)
		if err != nil {
			handleError(w, err)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, string(o))
	})

	router.RegistorEndpoint("GET /"+selfName+"/catalog/:date", func(w http.ResponseWriter, r *http.Request, values router.PathValues) {
		date := values["date"]
		mediaCatalog, err := conn.GetCatalog(date)
		if err != nil {
			handleError(w, err)
			return
		}

		o, err := json.Marshal(mapper.MediaRecordListToMediaCatalogList(mediaCatalog))
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

		o, err := json.Marshal(mapper.MediaRecordListToMediaDataList(mediaList))
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

func deleteCacheFileCore(conn *datasource.Database, id string) error {
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

func DeleteCacheFile(id string) error {
	conn, err := GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	return deleteCacheFileCore(conn, id)
}
