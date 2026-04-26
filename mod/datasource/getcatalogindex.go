package datasource

import (
	"strings"
)

type CatalogIndexEntry struct {
	Date  string   `json:"date"`
	Types []string `json:"types"`
}

func (conn *Database) GetCatalogIndex(minSize uint64) (entries []CatalogIndexEntry, err error) {
	query := `SELECT DISTINCT TO_CHAR(TO_TIMESTAMP(timestamp / 1000), 'YYYY-MM-DD') AS date,
			string_agg(DISTINCT type, ',') AS types
			FROM media
			WHERE content_length > $1 AND cache_path IS NOT NULL
			GROUP BY date
			ORDER BY date DESC`
	rows, err := conn.db.Query(query, minSize)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var entry CatalogIndexEntry
		var typesStr string
		if err = rows.Scan(&entry.Date, &typesStr); err != nil {
			return
		}
		entry.Types = strings.Split(typesStr, ",")
		entries = append(entries, entry)
	}
	return
}
