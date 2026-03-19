package cmd

import "os"

func createTempFile(pattern string) (*os.File, error) {
	return os.CreateTemp("", pattern)
}
