package lib

import (
	"fmt"
	"os"
)

// FileSize returns the size of the file specified by name
func FileSize(name string) (uint32, error) {
	fileInfo, err := os.Stat(name)
	if err != nil {
		return 0, fmt.Errorf("failed to obtain fileinfo: %w", err)
	}
	return uint32(fileInfo.Size()), nil
}
