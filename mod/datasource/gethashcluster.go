package datasource

import (
	"fmt"
	"strings"

	_ "github.com/lib/pq"
)

type Media struct {
	MediaID     string
	ContentHash uint64
}

func hammingDistance(a, b uint64, max int) bool {
	x := a ^ b
	count := 0
	for x != 0 {
		x &= x - 1
		count++
		if count > max {
			break
		}
	}
	return count <= max
}

func clusterMedia(mediaList []Media) map[string][]string {
	uf := NewUnionFind()
	n := len(mediaList)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if hammingDistance(mediaList[i].ContentHash, mediaList[j].ContentHash, 8) {
				uf.Union(mediaList[i].MediaID, mediaList[j].MediaID)
			}
		}
	}

	clusters := make(map[string][]string)
	for _, m := range mediaList {
		root := uf.Find(m.MediaID)
		clusters[root] = append(clusters[root], m.MediaID)
	}
	return clusters
}

func (conn *Database) GetHashCluster() (duplicatedMediaList [][]map[string]any, err error) {
	rows, err := conn.db.Query(`SELECT media_id, content_hash FROM media WHERE content_length > 0 AND content_hash != 0`)
	if err != nil {
		return
	}
	defer rows.Close()

	var mediaList []Media
	for rows.Next() {
		var m Media
		var sinedHash int64
		err = rows.Scan(&m.MediaID, &sinedHash)
		if err != nil {
			return
		}
		m.ContentHash = uint64(sinedHash)
		mediaList = append(mediaList, m)
	}

	clusters := clusterMedia(mediaList)

	for _, members := range clusters {
		if len(members) <= 1 {
			continue
		}

		placeholder := []string{}
		values := []any{}
		for no, id := range members {
			holder := fmt.Sprintf("$%d", no+1)
			placeholder = append(placeholder, holder)
			values = append(values, id)
		}

		mediaRecordList, err := conn.GetMediaByQuery("media_id IN ("+strings.Join(placeholder, ",")+")", values...)
		if err != nil {
			return nil, err
		}

		duplicatedMediaList = append(duplicatedMediaList, mediaListToSet(mediaRecordList))
	}

	return
}
