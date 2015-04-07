package common

import (
	"encoding/json"
	"os"
)

// Save the value d as json to the file f. The old content of f will be
// discarded
func Save(f *os.File, d interface{}) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := f.Seek(0, 0)
	if err != nil {
		return err
	}
	if err := json.NewEncoder(f).Encode(d); err != nil {
		return err
	}
	f.Sync()
	return nil
}
