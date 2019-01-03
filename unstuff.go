package stuffbin

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
)

// UnStuff takes the path to a stuffed binary, unstuffs it, and returns
// a FileSystem.
func UnStuff(path string) (FileSystem, error) {
	// Get stuffed zip data.
	b, err := GetStuff(path)
	if err != nil {
		return nil, err
	}

	// Unzip files into a FileSystem.
	fs, err := UnZip(b)
	if err != nil {
		return nil, err
	}

	return fs, nil
}

// GetStuff takes the path to a stuffed binary and extracts
// the packed data.
func GetStuff(in string) ([]byte, error) {
	id, err := GetFileID(in)
	if err != nil {
		return nil, err
	}

	// Read the zip data from the binary.
	b, err := getZipBytes(in, int64(id.BinSize), int64(id.ZipSize))
	if err != nil {
		return nil, err
	}

	return b, nil
}

// UnZip unzips zipped bytes and returns a FileSystem
// with the files mapped to it.
func UnZip(b []byte) (FileSystem, error) {
	r, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, err
	}

	fs, _ := NewFS()
	for _, f := range r.File {
		// Read the file.
		rd, err := f.Open()
		if err != nil {
			return nil, err
		}

		b := new(bytes.Buffer)
		if _, err := io.Copy(b, rd); err != nil {
			return nil, err
		}

		if err := fs.Add(NewFile(f.FileHeader.Name, f.FileInfo(), b.Bytes())); err != nil {
			return nil, err
		}
	}

	return fs, nil
}

// getZipBytes gets the embedded ZIP data from a binary
// given offset (from) and zipLen positions extracted
// from the embedded ID.
func getZipBytes(fName string, offset, zipLen int64) ([]byte, error) {
	f, err := os.Open(fName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var b = make([]byte, zipLen)
	_, err = f.ReadAt(b, offset)
	if err != nil {
		return nil, err
	}

	return b, nil
}
