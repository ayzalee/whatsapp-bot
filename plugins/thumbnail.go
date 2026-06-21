package plugins

import _ "embed"

//go:embed assets/black.jpg
var defaultThumbnailBytes []byte

func defaultThumbnail() []byte {
return defaultThumbnailBytes
}
