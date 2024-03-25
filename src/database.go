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
				timestamp       NUMERIC NOT NULL,
				duration_millis NUMERIC,
				video_url       TEXT,

				content_length  BIGINT,
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

	query := "INSERT INTO media(" + strings.Join(columns, ",") + ") VALUES (" + strings.Join(placeholder, "),(") + ") ON CONFLICT (media_id) DO UPDATE SET updated_at = EXCLUDED.updated_at"
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
				video_url
			FROM
				media
			WHERE
				removed='f'
			ORDER BY
				media_id
			`
	rows, err := conn.db.Query(query)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var mediaId string
		var parentUrl string
		var mediaType string
		var url string
		var timestamp uint64
		var durationMillis sql.NullInt32
		var videoUrl sql.NullString
		err = rows.Scan(&mediaId, &parentUrl, &mediaType, &url, &timestamp, &durationMillis, &videoUrl)
		if err != nil {
			return
		}
		mediaList = append(mediaList, map[string]any{
			"mediaId":        mediaId,
			"parentUrl":      parentUrl,
			"type":           mediaType,
			"url":            url,
			"timestamp":      timestamp,
			"durationMillis": durationMillis,
			"videoUrl":       videoUrl,
		})
	}

	return
}

func (conn *Database) DeleteMedia(id string) (err error) {
	_, err = conn.db.Exec("UPDATE media SET removed='t' WHERE media_id=$1", id)
	return
}
