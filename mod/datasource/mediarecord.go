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

func (m MediaRecord) Complete() MediaRecordComplement {
	return MediaRecordComplement{
		MediaRecord:   m,
		HasCache:      m.HasCache(),
		MediaPath:     m.GetMediaPath(),
		ThumbnailPath: m.GetThumbnailPath(),
	}
}

func (m MediaRecord) ToMediaCatalog() MediaCatalog {
	c := MediaCatalog{
		Id:            m.MediaId,
		ParentUrl:     m.ParentUrl,
		Type:          m.Type,
		Url:           m.Url,
		Timestamp:     m.Timestamp,
		HasCache:      m.HasCache(),
		ContentLength: 0,
	}

	if m.ContentLength.Valid {
		c.ContentLength = uint64(m.ContentLength.Int64)
	}
	if m.DurationMillis.Valid {
		c.DurationMillis = uint(m.DurationMillis.Int32)
	}
	if m.VideoUrl.Valid {
		c.VideoUrl = m.VideoUrl.String
	}

	mediaPath := m.GetMediaPath()
	if mediaPath.Valid {
		c.MediaPath = mediaPath.String
	}
	thumbnailPath := m.GetThumbnailPath()
	if thumbnailPath.Valid {
		c.ThumbPath = thumbnailPath.String
	}

	return c
}
