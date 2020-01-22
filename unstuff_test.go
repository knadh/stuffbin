package stuffbin

import (
	"runtime"
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
	expectedLength := len(b)
	if runtime.GOOS == "windows" {
		// reduce length by one to compensate for \r line ending byte on windows
		expectedLength--
	}
	assert(t, "mismatch in stuff byte size", mockZipSize, expectedLength)
}

func TestUnzipFiles(t *testing.T) {
	b, err := GetStuff(mockBinStuffed)
	assert(t, "error getting stuff", nil, err)
	expectedLength := len(b)
	if runtime.GOOS == "windows" {
		// reduce length by one to compensate for \r line ending byte on windows
		expectedLength--
	}
	assert(t, "mismatch in stuff byte size", mockZipSize, expectedLength)

	// Unzip the files and check if they're all there including
	// the alias.
	fs, err := UnZip(b)
	assert(t, "error unzipping", nil, err)
	f := fs.List()
	sort.Strings(f)
	assert(t, "mismatch in zipped file paths", stuffedFiles, f)
}
