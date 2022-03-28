package utils

func StrSliceContains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

var excludedTags = map[string]struct{}{
	"latest": {},
	"prod":   {},
	"stg":    {},
}

func GetValidImageTag(imageTags []string) string {
	for _, t := range imageTags {
		if _, exists := excludedTags[t]; !exists {
			return t
		}
	}

	return ""
}
