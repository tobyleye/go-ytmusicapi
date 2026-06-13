package parsers

import "fmt"

const (
	DOT_SEPARATOR = "•"
	AND_SEPARATOR = "&"
	COMMA         = ","
)

// ReadValue navigates through nested JSON structure using a path
func ReadValue(data interface{}, path []interface{}) interface{} {
	current := data
	for _, key := range path {
		switch v := current.(type) {
		case map[string]interface{}:
			if strKey, ok := key.(string); ok {
				current = v[strKey]
			} else {
				return nil
			}
		case []interface{}:
			if idx, ok := key.(int); ok && idx >= 0 && idx < len(v) {
				current = v[idx]
			} else {
				return nil
			}
		default:
			return nil
		}
	}
	return current
}

// ReadValueString reads a string value from nested JSON
func ReadValueString(data interface{}, path []interface{}) string {
	value := ReadValue(data, path)
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func CreatePlaylistLink(playlistId string) string {
	return fmt.Sprintf("https://music.youtube.com/playlist?list=%s", playlistId)
}

func CreateTrackLink(trackId string) string {
	return fmt.Sprintf("https://music.youtube.com/watch?v=%s", trackId)
}
