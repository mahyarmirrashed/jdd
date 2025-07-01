package jd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var JohnnyDecimalFilePattern = regexp.MustCompile(`^(\d{2})\.(\d{2})(\+\S+)?`)

type JohnnyDecimal struct {
	Area      string // e.g. "10-19"
	Category  string // e.g. "15"
	ID        string // e.g. "15.23"
	Extension string // e.g. "+JEM" or "+0001" (optional)
}

// Ensure that path for JohnnyDecimal object is created
func (jd *JohnnyDecimal) EnsureFolders(root string) (string, error) {
	finalPath := ""

	// Area
	areaPath, err := findOrCreatePrefixedFolder(root, jd.Area)
	if err != nil {
		return "", fmt.Errorf("could not ensure area folder: %w", err)
	}
	// Category
	categoryPath, err := findOrCreatePrefixedFolder(areaPath, jd.Category)
	if err != nil {
		return "", fmt.Errorf("could not ensure category folder: %w", err)
	}
	// ID
	idPath, err := findOrCreatePrefixedFolder(categoryPath, jd.ID)
	if err != nil {
		return "", fmt.Errorf("could not ensure ID folder: %w", err)
	}
	// Extension
	if jd.Extension != "" {
		extPath, err := findOrCreatePrefixedFolder(idPath, jd.ID+jd.Extension)
		if err != nil {
			return "", fmt.Errorf("could not ensure extension folder: %w", err)
		}
		finalPath = extPath
	} else {
		finalPath = idPath
	}

	return finalPath, nil
}

// Return the folder path segments for this JD object
func (jd *JohnnyDecimal) FolderPath() []string {
	return []string{jd.Area, jd.Category, jd.ID}
}

// Parse a filename prefix like "15.23" and return a JohnnyDecimal object
func Parse(filename string) (*JohnnyDecimal, error) {
	matches := JohnnyDecimalFilePattern.FindStringSubmatch(filename)
	if len(matches) <= 3 {
		return nil, fmt.Errorf("filename does not match Johnny Decimal pattern")
	}
	category := matches[1]
	id := fmt.Sprintf("%s.%s", matches[1], matches[2])

	extension := ""
	if len(matches) >= 4 && matches[3] != "" {
		extension = matches[3] // includes the "+"
	}

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

		Extension: extension,
	}, nil
}

// Return a formatted string for debugging
func (jd *JohnnyDecimal) String() string {
	str := fmt.Sprintf("Area: %s, Category: %s, ID: %s", jd.Area, jd.Category, jd.ID)

	if jd.Extension != "" {
		str = fmt.Sprintf("%s, Extension: %s", str, jd.Extension)
	}

	return str
}

// Checks for a folder in the parent directory that starts with the defined prefix.
// If found, returns its path. If not, creates prefix as a folder and returns its path.
func findOrCreatePrefixedFolder(parentDir, prefix string) (string, error) {
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			return filepath.Join(parentDir, entry.Name()), nil
		}
	}

	// Not found, create it
	fullPath := filepath.Join(parentDir, prefix)
	if err := os.Mkdir(fullPath, 0755); err != nil && !os.IsExist(err) {
		return "", err
	}
	return fullPath, nil
}
