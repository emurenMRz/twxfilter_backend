package datasource

func (conn *Database) GetCatalog(date string) (mediaRecordList []MediaCatalog, err error) {
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
				content_length > 0 AND cache_path IS NOT NULL AND TO_CHAR(TO_TIMESTAMP(timestamp / 1000), 'YYYY-MM-DD') = $1
			ORDER BY
				timestamp DESC
			`
	rows, err := conn.db.Query(query, date)
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

		mediaRecordList = append(mediaRecordList, mediaRecord.ToMediaCatalog())
	}

	return
}
