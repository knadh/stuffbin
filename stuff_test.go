package stuffbin

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sort"
	"strings"
	"testing"
)

const mockBin = "mock/mock.exe"
const mockBinStuffed = "mock/mock.exe.stuffed"
const mockBinStuffed2 = "mock/mock.exe.stuffed.temp"
const mockExeSize = 512
const mockZipSize = 338

var mockID = ID{
	Name:    [8]byte{'s', 't', 'u', 'f', 'f', 'b', 'i', 'n'},
	BinSize: mockExeSize,
	ZipSize: mockZipSize,
}

var (
	localFiles   = []string{"mock/bar.txt", "mock/foo.txt"}
	stuffedFiles = []string{"/mock/bar.txt", "/mock/foo.txt"}
)

func TestMain(m *testing.M) {
	setup()
	retCode := m.Run()
	teardown()
	os.Exit(retCode)
}

func TestMakeIDBytes(t *testing.T) {
	b := makeIDBytes(mockID)

	assert(t, "makeID returned unexpected bytes",
		[]byte{115, 116, 117, 102, 102, 98, 105, 110, 0, 0, 0, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 1, 82},
		b)
}

func TestStuff(t *testing.T) {
	exeSize, zipSize, err := Stuff(mockBin, mockBinStuffed2, "/", localFiles...)
	if runtime.GOOS == "windows" {
		// reduce length by one to compensate for \r line ending byte on windows
		zipSize--
	}
	assert(t, "error stuffing", nil, err)
	assert(t, "exe size", mockExeSize, exeSize)
	assert(t, "zip size", mockZipSize, zipSize)
	_ = os.Remove(mockBinStuffed2)
}

func TestStuffCustomRoot(t *testing.T) {
	_, _, err := Stuff(mockBin, mockBinStuffed2, "/root/", localFiles...)
	assert(t, "error stuffing", nil, err)

	fs, err := UnStuff(mockBinStuffed2)
	assert(t, "error unstuffing", nil, err)

	f := []string{"/root/mock/bar.txt", "/root/mock/foo.txt"}
	f2 := fs.List()
	sort.Strings(f2)
	assert(t, "mismatch in stuffed file paths with custom /root/", f, f2)
	_ = os.Remove(mockBinStuffed2)
}

func TestGetFileID(t *testing.T) {
	id, err := GetFileID(mockBinStuffed)
	assert(t, "error getting file ID", nil, err)
	if runtime.GOOS == "windows" {
		// reduce length by one to compensate for \r line ending byte on windows
		id.ZipSize--
	}
	assert(t, "error matching file ID", mockID, id)
}

func TestZipFiles(t *testing.T) {
	// Zip some files including a file with an alias.
	f := []string{"mock/foo.txt:/test/foo.txt"}
	f = append(f, localFiles...)
	b, err := zipFiles("/", f...)
	assert(t, "error zipping files", nil, err)

	// Unzip the files and check if they're all there including
	// the alias.
	fs, err := UnZip(b.Bytes())
	assert(t, "error unzipping", nil, err)

	f = []string{"/test/foo.txt", "/mock/foo.txt", "/mock/bar.txt"}
	sort.Strings(f)
	f2 := fs.List()
	sort.Strings(f2)
	assert(t, "mismatch in zipped file paths", f, f2)
}

func setup() {
	// Generate a fake EXE file with random bytes.
	b := make([]byte, mockExeSize)
	_, _ = rand.Read(b)
	err := ioutil.WriteFile(mockBin, b, 0755)
	if err != nil {
		panic(err)
	}

	if _, _, err := Stuff(mockBin, mockBinStuffed, "/", localFiles...); err != nil {
		panic(fmt.Sprintf("error stuffing: %v", err))
	}
}

func teardown() {
	_ = os.Remove(mockBin)
	_ = os.Remove(mockBinStuffed)
	_ = os.Remove(mockBinStuffed2)
}

func assert(t *testing.T, msg string, a interface{}, b interface{}) {
	// TODO: replace with strings.ReplaceAll once we drop support for GO 1.11
	if strings.Replace(fmt.Sprintf("%v", a), "\r\n", "\n", -1) == strings.Replace(fmt.Sprintf("%v", b), "\r\n", "\n", -1) {
		return
	}
	t.Fatalf("%s: %v != %v", msg, a, b)
}
