package models

type MediaCatalog struct {
	Id             string `json:"id"`
	ParentUrl      string `json:"parentUrl"`
	Type           string `json:"type"`
	Url            string `json:"url"`
	Timestamp      uint64 `json:"timestamp"`
	HasCache       bool   `json:"hasCache"`
	ContentLength  uint64 `json:"contentLength"`
	DurationMillis uint   `json:"durationMillis,omitempty"`
	VideoUrl       string `json:"videoUrl,omitempty"`
	MediaPath      string `json:"mediaPath,omitempty"`
	ThumbPath      string `json:"thumbPath,omitempty"`
}
