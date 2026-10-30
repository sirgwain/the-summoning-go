package summoning

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The game's data file names.
const (
	FileColors    = "COLORS"
	FileInit      = "INIT"
	FileInteract  = "INTERACT"
	FileKeywords  = "KEYWORDS"
	FileLevels    = "LEVELS"
	FileNPC       = "NPC"
	FileObjects   = "OBJECTS"
	FileResource  = "RESOURCE"
	FileText      = "TEXT"
	FileGraphIni  = "GRAPH.INI"
	FileClassData = "F"
)

// ReadDataFile reads the data file name from dir, matching the name case-insensitively.
func ReadDataFile(dir, name string) ([]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(e.Name(), name) {
			return os.ReadFile(filepath.Join(dir, e.Name()))
		}
	}
	return nil, fmt.Errorf("%s: no file %s", dir, name)
}
