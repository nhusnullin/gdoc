package drive

import (
	"bytes"
	"io"
	"net/http"
)

// readAndRestore drains a response body and puts an identical one back, so the
// caller downstream still gets to read it.
func readAndRestore(response *http.Response) ([]byte, error) {
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	response.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	return body, nil
}
