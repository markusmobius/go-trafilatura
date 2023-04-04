package main

import (
	"fmt"
	"os"
	fp "path/filepath"
)

func openDataFile(name string) (*os.File, error) {
	// Try in comparison dir
	f, err := os.Open(fp.Join("test-files", "comparison", name))
	if err == nil {
		return f, nil
	}

	// Try in mock dir
	f, err = os.Open(fp.Join("test-files", "mock", name))
	if err == nil {
		return f, nil
	}

	return nil, fmt.Errorf("failed to open %s in comparison and mock dir", name)
}
