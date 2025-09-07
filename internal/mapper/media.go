package mapper

import (
	"datasource"
	"mediadata"

	"github.com/emurenMRz/twxfilter_backend/internal/models"
)

func MediaRecordToMediaCatalog(m datasource.MediaRecord) models.MediaCatalog {
	c := models.MediaCatalog{
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

func MediaRecordListToMediaCatalogList(records []datasource.MediaRecord) []models.MediaCatalog {
	var catalogs []models.MediaCatalog
	for _, v := range records {
		catalogs = append(catalogs, MediaRecordToMediaCatalog(v))
	}
	return catalogs
}

func MediaRecordToMediaData(m datasource.MediaRecord) mediadata.MediaData {
	md := mediadata.MediaData{
		Id:        m.MediaId,
		ParentUrl: m.ParentUrl,
		Type:      m.Type,
		Url:       m.Url,
		Timestamp: m.Timestamp,
		HasCache:  m.HasCache(),
	}

	if m.DurationMillis.Valid {
		md.DurationMillis = uint(m.DurationMillis.Int32)
	}
	if m.VideoUrl.Valid {
		md.VideoUrl = m.VideoUrl.String
	}

	mediaPath := m.GetMediaPath()
	if mediaPath.Valid {
		md.MediaPath = mediaPath.String
	}
	thumbnailPath := m.GetThumbnailPath()
	if thumbnailPath.Valid {
		md.ThumbPath = thumbnailPath.String
	}

	return md
}

func MediaRecordListToMediaDataList(mediaList []datasource.MediaRecord) []mediadata.MediaData {
	set := []mediadata.MediaData{}
	for _, media := range mediaList {
		set = append(set, MediaRecordToMediaData(media))
	}
	return set
}

func MediaRecordSetListToMediaDataSetList(mediaSetList [][]datasource.MediaRecord) [][]mediadata.MediaData {
	setList := [][]mediadata.MediaData{}
	for _, mediaSet := range mediaSetList {
		setList = append(setList, MediaRecordListToMediaDataList(mediaSet))
	}
	return setList
}
