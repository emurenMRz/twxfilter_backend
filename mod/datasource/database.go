package datasource

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

type ConnectConfig struct {
	User     string
	Password string
	Host     string
	Port     uint16
	Dbname   string
}

type Database struct {
	ConnectConfig

	db *sql.DB
}

func Connect(config ConnectConfig) (conn *Database, err error) {
	conn = &Database{config, nil}
	conn.db, err = sql.Open("postgres", "user="+conn.User+" dbname="+conn.Dbname+" sslmode=disable")
	if err != nil {
		return
	}

	sql := `CREATE TABLE IF NOT EXISTS media(
				media_id        TEXT PRIMARY KEY,
				parent_url      TEXT NOT NULL,
				type            TEXT NOT NULL,
				url             TEXT NOT NULL,
				timestamp       NUMERIC NOT NULL DEFAULT (EXTRACT(epoch FROM now()) * 1000::numeric)::bigint::numeric,
				duration_millis NUMERIC,
				video_url       TEXT,

				content_length  BIGINT,
				content_hash    BIGINT,
				cache_path      TEXT,
				thumbnail       BYTEA,

				removed         BOOLEAN NOT NULL DEFAULT FALSE,
				created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			)`
	_, err = conn.db.Query(sql)

	return
}

func Create(config ConnectConfig) (err error) {
	db, err := sql.Open("postgres", "user="+config.User+" dbname=postgres")
	if err != nil {
		return
	}
	defer db.Close()

	_, err = db.Query("CREATE DATABASE " + config.Dbname + " TEMPLATE 'template0' ENCODING 'UTF-8' LC_COLLATE 'C.UTF-8' LC_CTYPE 'C.UTF-8'")

	return
}

func (conn *Database) Close() {
	conn.db.Close()
	log.Println("Close database: " + conn.Dbname)
}

func (conn *Database) UpsertMedia(columns []string, valueTable [][]any) (err error) {
	if len(valueTable) == 0 {
		err = fmt.Errorf("no data")
		return
	}

	colLen := len(columns)
	for i, row := range valueTable {
		if colLen != len(row) {
			err = fmt.Errorf("diferent table row / columns: %d row", i)
			return
		}
	}

	var placeholder []string
	var values []any

	for i, row := range valueTable {
		rowPlaceholder := ""
		for j, col := range row {
			if j > 0 {
				rowPlaceholder += ","
			}
			rowPlaceholder += `$` + strconv.Itoa(i*colLen+j+1)
			values = append(values, col)
		}
		placeholder = append(placeholder, rowPlaceholder)
	}

	query := fmt.Sprintf(`INSERT INTO media (%s)
			VALUES (%s)
			ON CONFLICT (media_id)
			DO NOTHING
			`,
		strings.Join(columns, ","),
		strings.Join(placeholder, "),("))
	_, err = conn.db.Exec(query, values...)
	return
}

func (conn *Database) GetMedia() (mediaList []map[string]any, err error) {
	query := `SELECT
				media_id,
				parent_url,
				type,
				url,
				timestamp,
				duration_millis,
				video_url,
				content_length,
				cache_path,
				CASE WHEN thumbnail IS NOT NULL THEN true ELSE false END AS thumbnail
			FROM
				media
			WHERE
				removed='f'
			ORDER BY
				timestamp DESC
			`
	rows, err := conn.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mediaRecord MediaRecord
		err = rows.Scan(
			&mediaRecord.MediaId,
			&mediaRecord.ParentUrl,
			&mediaRecord.Type,
			&mediaRecord.Url,
			&mediaRecord.Timestamp,
			&mediaRecord.DurationMillis,
			&mediaRecord.VideoUrl,
			&mediaRecord.ContentLength,
			&mediaRecord.CachePath,
			&mediaRecord.HasThumbnail,
		)
		if err != nil {
			return
		}

		mediaList = append(mediaList, mediaRecordToMap(mediaRecord))
	}

	return
}

func (conn *Database) GetMediaByID(id string) (mediaRecord MediaRecord, err error) {
	query := `SELECT
				media_id,
				parent_url,
				type,
				url,
				timestamp,
				duration_millis,
				video_url,
				content_length,
				content_hash,
				cache_path,
				CASE WHEN thumbnail IS NOT NULL THEN true ELSE false END AS thumbnail,
				removed
			FROM
				media
			WHERE
				media_id=$1
			`
	row := conn.db.QueryRow(query, id)

	err = row.Scan(
		&mediaRecord.MediaId,
		&mediaRecord.ParentUrl,
		&mediaRecord.Type,
		&mediaRecord.Url,
		&mediaRecord.Timestamp,
		&mediaRecord.DurationMillis,
		&mediaRecord.VideoUrl,
		&mediaRecord.ContentLength,
		&mediaRecord.ContentHash,
		&mediaRecord.CachePath,
		&mediaRecord.HasThumbnail,
		&mediaRecord.Removed,
	)

	return
}

func (conn *Database) GetMediaByQuery(where string, args ...any) (mediaRecordList []MediaRecord, err error) {
	query := fmt.Sprintf(`SELECT
				media_id,
				parent_url,
				type,
				url,
				timestamp,
				duration_millis,
				video_url,
				content_length,
				content_hash,
				cache_path,
				CASE WHEN thumbnail IS NOT NULL THEN true ELSE false END AS thumbnail,
				removed
			FROM
				media
			WHERE
				%s
			`, where)
	rows, err := conn.db.Query(query, args...)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mediaRecord MediaRecord
		err = rows.Scan(
			&mediaRecord.MediaId,
			&mediaRecord.ParentUrl,
			&mediaRecord.Type,
			&mediaRecord.Url,
			&mediaRecord.Timestamp,
			&mediaRecord.DurationMillis,
			&mediaRecord.VideoUrl,
			&mediaRecord.ContentLength,
			&mediaRecord.ContentHash,
			&mediaRecord.CachePath,
			&mediaRecord.HasThumbnail,
			&mediaRecord.Removed,
		)
		if err != nil {
			return
		}

		mediaRecordList = append(mediaRecordList, mediaRecord)
	}

	return
}

func (conn *Database) GetThumbnailByID(id string) (thumbnail []byte, err error) {
	query := `SELECT thumbnail FROM media WHERE media_id=$1`
	row := conn.db.QueryRow(query, id)

	err = row.Scan(&thumbnail)

	return
}

func (conn *Database) DeleteMediaAll() (err error) {
	_, err = conn.db.Exec("UPDATE media SET removed='t', updated_at=CURRENT_TIMESTAMP WHERE removed='f'")
	return
}

func (conn *Database) DeleteMediaCached() (err error) {
	_, err = conn.db.Exec("UPDATE media SET removed='t', updated_at=CURRENT_TIMESTAMP WHERE removed='f' AND content_length IS NOT NULL")
	return
}

func (conn *Database) DeleteMedia(id string) (err error) {
	_, err = conn.db.Exec("UPDATE media SET removed='t', updated_at=CURRENT_TIMESTAMP WHERE removed='f' AND media_id=$1", id)
	return
}

func (conn *Database) DeleteCacheFile(id string) (err error) {
	_, err = conn.db.Exec("UPDATE media SET content_length=0, cache_path=NULL, removed='t', updated_at=CURRENT_TIMESTAMP WHERE media_id=$1", id)
	return
}

func (conn *Database) GetNoCacheMedia() (mediaList []map[string]any, err error) {
	query := `SELECT
				media_id,
				type,
				url,
				video_url
			FROM
				media
			WHERE
				removed='f' AND content_length IS NULL
			ORDER BY
				timestamp DESC
			`
	rows, err := conn.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mediaId string
		var mediaType string
		var url string
		var videoUrl sql.NullString
		err = rows.Scan(&mediaId, &mediaType, &url, &videoUrl)
		if err != nil {
			return
		}
		mediaList = append(mediaList, map[string]any{
			"mediaId":  mediaId,
			"type":     mediaType,
			"url":      url,
			"videoUrl": videoUrl,
		})
	}

	return
}

type CachedVideoMedia struct {
	Id   string
	Path string
}

func (conn *Database) GetCachedVideoMedia() (cachedVideoMediaList []CachedVideoMedia, err error) {
	query := `SELECT
				media_id,
				cache_path
			FROM
				media
			WHERE
				type='video' AND removed='f' AND content_length IS NOT NULL
			ORDER BY
				timestamp DESC
			`
	rows, err := conn.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mediaId string
		var cachePath sql.NullString
		err = rows.Scan(&mediaId, &cachePath)
		if err != nil {
			return
		}
		if cachePath.Valid {
			cachedVideoMediaList = append(cachedVideoMediaList, CachedVideoMedia{
				Id:   mediaId,
				Path: cachePath.String,
			})
		}
	}

	return
}

type UnhashedMedia struct {
	MediaId   string
	Thumbnail []byte
}

func (conn *Database) GetUnhashedMedia() (unhashedMediaList []UnhashedMedia, err error) {
	query := `SELECT
				media_id,
				type,
				thumbnail
			FROM
				media
			WHERE
				content_length > 0 AND content_hash IS NULL AND thumbnail IS NOT NULL
			`
	rows, err := conn.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mediaId string
		var mediaType string
		var thumbnail []byte
		err = rows.Scan(&mediaId, &mediaType, &thumbnail)
		if err != nil {
			return
		}

		unhashedMediaList = append(unhashedMediaList, UnhashedMedia{
			MediaId:   mediaId,
			Thumbnail: thumbnail,
		})
	}

	return
}

type DuplicatedHash struct {
	ContentHash int64
	Count       int
}

func (conn *Database) GetMediaDataSet(where string, args ...any) ([]map[string]any, error) {
	result, err := conn.GetMediaByQuery(where, args...)
	if err != nil {
		return nil, err
	}

	return mediaListToSet(result), nil
}

func GetSelfName() string {
	t := strings.Split(os.Args[0], "/")
	return t[len(t)-1]
}

func mediaRecordToMap(mediaRecord MediaRecord) map[string]any {
	return map[string]any{
		"mediaId":        mediaRecord.MediaId,
		"parentUrl":      mediaRecord.ParentUrl,
		"type":           mediaRecord.Type,
		"url":            mediaRecord.Url,
		"timestamp":      mediaRecord.Timestamp,
		"durationMillis": mediaRecord.DurationMillis,
		"videoUrl":       mediaRecord.VideoUrl,
		"hasCache":       mediaRecord.HasCache(),
		"mediaPath":      mediaRecord.GetMediaPath(),
		"thumbPath":      mediaRecord.GetThumbnailPath(),
	}
}

func mediaListToSet(mediaRecordList []MediaRecord) []map[string]any {
	duplicatedMediaSet := []map[string]any{}
	for _, mediaRecord := range mediaRecordList {
		duplicatedMediaSet = append(duplicatedMediaSet, mediaRecordToMap(mediaRecord))
	}
	return duplicatedMediaSet
}

func (conn *Database) SetCacheData(mediaId string, contentLength uint64, contentHash uint64, cachePath string) (err error) {
	_, err = conn.db.Exec("UPDATE media SET content_length=$2, content_hash=$3, cache_path=$4, updated_at=CURRENT_TIMESTAMP WHERE media_id=$1", mediaId, contentLength, int64(contentHash), cachePath)
	return
}

func (conn *Database) SetThumbnail(mediaId string, thumbnail []byte) (err error) {
	_, err = conn.db.Exec("UPDATE media SET thumbnail=$2, updated_at=CURRENT_TIMESTAMP WHERE media_id=$1", mediaId, thumbnail)
	return
}

func (conn *Database) SetContentHashData(mediaId string, contentHash uint64) (err error) {
	_, err = conn.db.Exec("UPDATE media SET content_hash=$2, updated_at=CURRENT_TIMESTAMP WHERE media_id=$1", mediaId, contentHash)
	return
}
