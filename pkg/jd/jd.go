package jd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var JohnnyDecimalFilePattern = regexp.MustCompile(`^(\d{2})\.(\d{2})`)

type JohnnyDecimal struct {
	Area     string // e.g. "10-19"
	Category string // e.g. "15"
	ID       string // e.g. "15.23"
}

// Ensure that path for JohnnyDecimal object is created
func (jd *JohnnyDecimal) EnsureFolders(root string) (string, error) {
	// Compose the path: root/area/category/id
	fullPath := filepath.Join(root, jd.Area, jd.Category, jd.ID)
	err := os.MkdirAll(fullPath, 0755)
	if err != nil {
		return "", fmt.Errorf("could not create folders: %w", err)
	}
	return fullPath, nil
}

// Return the folder path segments for this JD object
func (jd *JohnnyDecimal) FolderPath() []string {
	return []string{jd.Area, jd.Category, jd.ID}
}

// Parse a filename prefix like "15.23" and return a JohnnyDecimal object
func Parse(filename string) (*JohnnyDecimal, error) {
	matches := JohnnyDecimalFilePattern.FindStringSubmatch(filename)
	if len(matches) != 3 {
		return nil, fmt.Errorf("filename does not match Johnny Decimal pattern")
	}
	category := matches[1]
	id := fmt.Sprintf("%s.%s", matches[1], matches[2])

	firstDigit, err := strconv.Atoi(string(category[0]))
	if err != nil {
		return nil, err
	}
	areaStart := firstDigit * 10
	areaEnd := areaStart + 9
	area := fmt.Sprintf("%02d-%02d", areaStart, areaEnd)

	return &JohnnyDecimal{
		Area:     area,
		Category: category,
		ID:       id,
	}, nil
}

// Return a formatted string for debugging
func (jd *JohnnyDecimal) String() string {
	return fmt.Sprintf("Area: %s, Category: %s, ID: %s", jd.Area, jd.Category, jd.ID)
}
