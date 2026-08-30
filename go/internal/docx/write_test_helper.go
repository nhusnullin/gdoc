package docx

import "os"

// writeFile exists so a test can lay down a fixture archive without importing
// os itself into every test file.
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}
