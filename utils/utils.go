package utils

import "errors"

func StrSliceContains(slice []string, target string) bool {
	for _, s := range slice {
		if s == target {
			return true
		}
	}
	return false
}

var excludedTags = map[string]struct{}{
	"latest":     {},
	"prod":       {},
	"stg":        {},
	"production": {},
	"staging":    {},
}

func GetValidImageTag(imageTags []string) (string, error) {
	for _, t := range imageTags {
		if _, exists := excludedTags[t]; !exists {
			return t, nil
		}
	}

	return "", errors.New("valid image tag not found")
}
