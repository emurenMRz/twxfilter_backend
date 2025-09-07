package datasource

import (
	"database/sql"
	"strings"
)

type MediaRecord struct {
	MediaId        string
	ParentUrl      string
	Type           string
	Url            string
	Timestamp      uint64
	DurationMillis sql.NullInt32
	VideoUrl       sql.NullString
	ContentLength  sql.NullInt64
	ContentHash    sql.NullInt64
	CachePath      sql.NullString
	HasThumbnail   bool
	Removed        bool
}

type MediaRecordComplement struct {
	MediaRecord
	HasCache      bool
	MediaPath     sql.NullString
	ThumbnailPath sql.NullString
}

func (m MediaRecord) getRelativePath() string {
	index := strings.Index(m.CachePath.String, ".cache")
	return m.CachePath.String[index:]
}

func (m MediaRecord) HasCache() bool {
	return m.ContentLength.Valid && m.ContentLength.Int64 > 0
}

func (m MediaRecord) GetMediaPath() sql.NullString {
	var mediaPath sql.NullString
	if m.CachePath.Valid {
		mediaPath.String = m.getRelativePath()
		mediaPath.Valid = true
	}
	return mediaPath
}

func (m MediaRecord) GetThumbnailPath() sql.NullString {
	var thumbPath sql.NullString
	if m.CachePath.Valid {
		thumbPath.String = m.getRelativePath()
		thumbPath.Valid = true
		if m.HasThumbnail {
			thumbPath.String = GetSelfName() + "/thumbnail/" + m.MediaId
		} else if m.Type != "photo" {
			ext := strings.LastIndex(thumbPath.String, ".")
			thumbPath.String = thumbPath.String[:ext] + "_thumb.jpg"
		}
	}
	return thumbPath
}

func (m MediaRecord) ToCompletion() MediaRecordComplement {
	return MediaRecordComplement{
		MediaRecord:   m,
		HasCache:      m.HasCache(),
		MediaPath:     m.GetMediaPath(),
		ThumbnailPath: m.GetThumbnailPath(),
	}
}
