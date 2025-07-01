package jd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// JohnnyDecimalFilePattern matches a Johnny Decimal filename prefix like "15.23" or "15.23+JEM".
var JohnnyDecimalFilePattern = regexp.MustCompile(`^(\d{2})\.(\d{2})(\+\S+)?`)

// JohnnyDecimal represents a parsed Johnny Decimal ID with optional sub-ID.
type JohnnyDecimal struct {
	Area     string // Area range, e.g. "10-19"
	Category string // Category number, e.g. "15"
	ID       string // Full ID, e.g. "15.23"
	SubID    string // Optional sub-ID, e.g. "+JEM" or "+0001"
}

// EnsureFolders ensures the folder structure for the JohnnyDecimal object exists under root.
// It creates folders for Area, Category, ID, and optionally the SubID (extension).
// Returns the final folder path.
func (jd *JohnnyDecimal) EnsureFolders(root string) (string, error) {
	// Ensure Area folder
	areaPath, err := findOrCreatePrefixedFolder(root, jd.Area)
	if err != nil {
		return "", fmt.Errorf("could not ensure area folder: %w", err)
	}
	// Ensure Category folder
	categoryPath, err := findOrCreatePrefixedFolder(areaPath, jd.Category)
	if err != nil {
		return "", fmt.Errorf("could not ensure category folder: %w", err)
	}
	// Ensure ID folder
	idPath, err := findOrCreatePrefixedFolder(categoryPath, jd.ID)
	if err != nil {
		return "", fmt.Errorf("could not ensure ID folder: %w", err)
	}

	finalPath := idPath

	// Ensure SubID (extension) folder if present
	if jd.SubID != "" {
		extPath, err := findOrCreatePrefixedFolder(idPath, jd.ID+jd.SubID)
		if err != nil {
			return "", fmt.Errorf("could not ensure extension folder: %w", err)
		}

		finalPath = extPath
	}

	return finalPath, nil
}

// FolderPath returns the folder path segments for this JohnnyDecimal object (Area, Category, ID).
func (jd *JohnnyDecimal) FolderPath() []string {
	return []string{jd.Area, jd.Category, jd.ID}
}

// Parse parses a filename prefix like "15.23" or "15.23+JEM" and returns a JohnnyDecimal object.
// Returns an error if the filename does not match the Johnny Decimal pattern.
func Parse(filename string) (*JohnnyDecimal, error) {
	matches := JohnnyDecimalFilePattern.FindStringSubmatch(filename)
	if len(matches) < 3 {
		return nil, fmt.Errorf("filename does not match Johnny Decimal pattern")
	}

	category := matches[1]
	id := fmt.Sprintf("%s.%s", matches[1], matches[2])

	subid := ""
	if len(matches) >= 4 && matches[3] != "" {
		subid = matches[3] // includes the "+"
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
		SubID:    subid,
	}, nil
}

// String returns a formatted string representation of the JohnnyDecimal object for debugging.
func (jd *JohnnyDecimal) String() string {
	str := fmt.Sprintf("Area: %s, Category: %s, ID: %s", jd.Area, jd.Category, jd.ID)

	if jd.SubID != "" {
		str = fmt.Sprintf("%s, Sub-ID: %s", str, jd.SubID)
	}

	return str
}

// findOrCreatePrefixedFolder looks for a folder in parentDir starting with prefix.
// If found, returns its path. Otherwise, creates the folder and returns its path.
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
