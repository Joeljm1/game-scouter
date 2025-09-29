package helpers

import (
	"bytes"
	"encoding/gob"
)

// go data structure to []bytes
func SerializeGoB(data any) ([]byte, error) {
	buff := new(bytes.Buffer)
	err := gob.NewEncoder(buff).Encode(data)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// []bytes to go data structure
// dest should be pointer to the dat structure
func DeserializeGoB(src []byte, dest any) error {
	reader := bytes.NewReader(src)
	err := gob.NewDecoder(reader).Decode(dest)
	return err
}
