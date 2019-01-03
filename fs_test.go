package stuffbin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

func TestFileServer(t *testing.T) {
	fs, err := UnStuff(mockBinStuffed)
	assert(t, "error unstuffing", nil, err)

	ts := httptest.NewServer(fs.FileServer())
	defer ts.Close()

	uri := "/" + localFiles[0]
	res, err := http.Get(ts.URL + uri)
	assert(t, "error in GET "+uri, nil, err)
	assert(t, "status error in GET "+uri, 200, res.StatusCode)

	uri = "/" + localFiles[1]
	res, err = http.Get(ts.URL + uri)
	assert(t, "error in GET "+uri, nil, err)
	assert(t, "status error in GET "+uri, 200, res.StatusCode)

	uri = "/nope"
	res, err = http.Get(ts.URL + uri)
	assert(t, "error in GET "+uri, nil, err)
	assert(t, "status error in GET "+uri, 404, res.StatusCode)
}

func TestNewLocalFSWithAlias(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/:test/", "mock/foo.txt")
	assert(t, "error creating local FS", nil, err)

	f := []string{"/mock/foo.txt", "/test/bar.txt", "/test/foo.txt", "/test/mock.exe", "/test/mock.exe.stuffed", "/test/mock.go", "/test/subdir/baz.txt"}
	sort.Strings(f)

	f2 := fs.List()
	sort.Strings(f2)
	assert(t, "mismatch in local FS", f, f2)
}

func TestNewLocalFS(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/", "mock/foo.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	f := []string{"/foo.txt", "/mock/bar.txt", "/mock/foo.txt", "/mock/mock.exe", "/mock/mock.exe.stuffed", "/mock/mock.go", "/mock/subdir/baz.txt"}
	sort.Strings(f)

	f2 := fs.List()
	sort.Strings(f2)
	assert(t, "mismatch in local FS", f, f2)
}

func TestGlob(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/", "mock/foo.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	g, _ := fs.Glob("/foo.txt")
	assert(t, "glob match failed", []string{"/foo.txt"}, g)

	g, _ = fs.Glob("/mock/*.exe")
	assert(t, "glob match failed", []string{"/mock/mock.exe"}, g)
}

func TestParseTemplates(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/", "mock/foo.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	tpl, err := ParseTemplates(fs, "/foo.txt")
	assert(t, "error parsing template", nil, err)

	b := bytes.Buffer{}
	tpl.Execute(&b, nil)
	assert(t, "mismatch in executed template", "foo", string(b.Bytes()))
}

func TestParseTemplatesGlob(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/", "mock/foo.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	tpl, err := ParseTemplatesGlob(fs, "/*.txt")
	assert(t, "error parsing template", nil, err)

	b := bytes.Buffer{}
	tpl.Execute(&b, nil)
	assert(t, "mismatch in executed template", "foo", string(b.Bytes()))
}
