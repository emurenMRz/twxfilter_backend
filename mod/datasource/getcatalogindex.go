package datasource

func (conn *Database) GetCatalogIndex(minSize uint64) (dates []string, err error) {
	query := `SELECT DISTINCT TO_CHAR(TO_TIMESTAMP(timestamp / 1000), 'YYYY-MM-DD')
			FROM media
			WHERE content_length > $1 AND cache_path IS NOT NULL
			ORDER BY TO_CHAR(TO_TIMESTAMP(timestamp / 1000), 'YYYY-MM-DD') DESC`
	rows, err := conn.db.Query(query, minSize)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var dateStr string
		if err = rows.Scan(&dateStr); err != nil {
			return
		}
		dates = append(dates, dateStr)
	}
	return
}
