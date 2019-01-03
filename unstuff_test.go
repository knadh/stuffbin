package stuffbin

import (
	"sort"
	"testing"
)

func TestUnStuff(t *testing.T) {
	fs, err := UnStuff(mockBinStuffed)
	assert(t, "error unstuffing", nil, err)
	f := fs.List()
	sort.Strings(f)
	assert(t, "mismatch in unstuffed file paths", stuffedFiles, f)
}

func TestGetStuff(t *testing.T) {
	b, err := GetStuff(mockBinStuffed)
	assert(t, "error getting stuff", nil, err)
	assert(t, "mismatch in stuff byte size", mockZipSize, len(b))
}

func TestUnzipFiles(t *testing.T) {
	b, err := GetStuff(mockBinStuffed)
	assert(t, "error getting stuff", nil, err)
	assert(t, "mismatch in stuff byte size", mockZipSize, len(b))

	// Unzip the files and check if they're all there including
	// the alias.
	fs, err := UnZip(b)
	assert(t, "error unzipping", nil, err)
	f := fs.List()
	sort.Strings(f)
	assert(t, "mismatch in zipped file paths", stuffedFiles, f)
}
