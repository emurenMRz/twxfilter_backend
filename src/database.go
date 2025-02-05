package main

import (
	"database/sql"
	"fmt"
	"log"
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

type MediaRecord struct {
	MediaId        string         `json:"id"`
	ParentUrl      string         `json:"parentUrl"`
	Type           string         `json:"type"`
	Url            string         `json:"url"`
	Timestamp      uint64         `json:"timestamp"`
	DurationMillis sql.NullInt32  `json:"durationMillis,omitempty"`
	VideoUrl       sql.NullString `json:"videoUrl,omitempty"`
	ContentLength  sql.NullInt64  `json:"ContentLength,omitempty"`
	ContentHash    sql.NullInt64  `json:"ContentHash,omitempty"`
	CachePath      sql.NullString `json:"CachePath,omitempty"`
	Removed        bool           `json:"Removed"`
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
			DO UPDATE SET
				removed = EXCLUDED.removed,
				updated_at = EXCLUDED.updated_at
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
				cache_path
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
		&mediaRecord.Removed,
	)

	return
}

func (conn *Database) GetMediaByContentHash(contentHash uint64) (mediaRecordList []MediaRecord, err error) {
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
				removed
			FROM
				media
			WHERE
				content_hash=$1
			`
	rows, err := conn.db.Query(query, contentHash)
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
			&mediaRecord.Removed,
		)
		if err != nil {
			return
		}

		mediaRecordList = append(mediaRecordList, mediaRecord)
	}

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

func (conn *Database) GetCachedVideoMedia() (cachePathList []string, err error) {
	query := `SELECT
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
		var cachePath sql.NullString
		err = rows.Scan(&cachePath)
		if err != nil {
			return
		}
		if cachePath.Valid {
			cachePathList = append(cachePathList, cachePath.String)
		}
	}

	return
}

type UnhashedMedia struct {
	MediaId       string
	ThumbnailPath string
}

func (conn *Database) GetUnhashedMedia() (unhashedMediaList []UnhashedMedia, err error) {
	query := `SELECT
				media_id,
				type,
				cache_path
			FROM
				media
			WHERE
				content_length > 0 AND content_hash IS NULL AND cache_path IS NOT NULL
			`
	rows, err := conn.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mediaId string
		var mediaType string
		var cachePath string
		err = rows.Scan(&mediaId, &mediaType, &cachePath)
		if err != nil {
			return
		}

		if mediaType != "photo" {
			ext := strings.LastIndex(cachePath, ".")
			cachePath = cachePath[:ext] + "_thumb.jpg"
		}

		unhashedMediaList = append(unhashedMediaList, UnhashedMedia{
			MediaId:       mediaId,
			ThumbnailPath: cachePath,
		})
	}

	return
}

type DuplicatedHash struct {
	ContentHash uint64
	Count       int
}

func (conn *Database) GetDuplicatedHash() (duplicatedHashList []DuplicatedHash, err error) {
	query := `SELECT
				content_hash,
				COUNT(*)
			FROM
				media
			WHERE
				content_length > 0 AND content_hash > 0
			GROUP BY
				content_hash
			HAVING
				COUNT(*) >= 2
			`
	rows, err := conn.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var contentHash uint64
		var count int
		err = rows.Scan(&contentHash, &count)
		if err != nil {
			return
		}

		duplicatedHashList = append(duplicatedHashList, DuplicatedHash{
			ContentHash: contentHash,
			Count:       count,
		})
	}

	return
}

func (conn *Database) GetDuplicatedMedia() (duplicatedMediaList [][]map[string]any, err error) {
	duplicatedHashList, err := conn.GetDuplicatedHash()
	if err != nil {
		return
	}

	for _, duplicatedHash := range duplicatedHashList {
		mediaRecordList, err := conn.GetMediaByContentHash(duplicatedHash.ContentHash)
		if err != nil {
			return nil, err
		}

		duplicatedMediaList = append(duplicatedMediaList, mediaListToSet(mediaRecordList))
	}

	return
}

func mediaRecordToMap(mediaRecord MediaRecord) map[string]any {
	var mediaPath sql.NullString
	var thumbPath sql.NullString
	if mediaRecord.CachePath.Valid {
		index := strings.Index(mediaRecord.CachePath.String, ".cache")
		relativePath := mediaRecord.CachePath.String[index:]
		mediaPath.String = relativePath
		mediaPath.Valid = true
		thumbPath.String = relativePath
		thumbPath.Valid = true
		if mediaRecord.Type != "photo" {
			ext := strings.LastIndex(thumbPath.String, ".")
			thumbPath.String = thumbPath.String[:ext] + "_thumb.jpg"
		}
	}

	return map[string]any{
		"mediaId":        mediaRecord.MediaId,
		"parentUrl":      mediaRecord.ParentUrl,
		"type":           mediaRecord.Type,
		"url":            mediaRecord.Url,
		"timestamp":      mediaRecord.Timestamp,
		"durationMillis": mediaRecord.DurationMillis,
		"videoUrl":       mediaRecord.VideoUrl,
		"hasCache":       mediaRecord.ContentLength.Valid && mediaRecord.ContentLength.Int64 > 0,
		"mediaPath":      mediaPath,
		"thumbPath":      thumbPath,
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

func (conn *Database) SetContentHashData(mediaId string, contentHash uint64) (err error) {
	_, err = conn.db.Exec("UPDATE media SET content_hash=$2, updated_at=CURRENT_TIMESTAMP WHERE media_id=$1", mediaId, contentHash)
	return
}
