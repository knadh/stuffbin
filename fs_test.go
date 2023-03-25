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

	f := []string{"/mock/foo.txt",
		"/test/bar.txt",
		"/test/foo.txt",
		"/test/foofunc.txt",
		"/test/mock.exe",
		"/test/mock.exe.stuffed",
		"/test/mock.go",
		"/test/subdir/baz.txt"}
	sort.Strings(f)

	f2 := fs.List()
	sort.Strings(f2)
	assert(t, "mismatch in local FS", f, f2)
}

func TestNewLocalFS(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/",
		"mock/foo.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	f := []string{"/foo.txt",
		"/mock/bar.txt",
		"/mock/foo.txt",
		"/mock/foofunc.txt",
		"/mock/mock.exe",
		"/mock/mock.exe.stuffed",
		"/mock/mock.go",
		"/mock/subdir/baz.txt"}
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

	g, err := fs.Glob("/foo.txt")
	assert(t, "glob creation failed", nil, err)
	assert(t, "glob match failed", []string{"/foo.txt"}, g)

	g, err = fs.Glob("/mock/*.exe")
	assert(t, "glob creation failed", nil, err)
	assert(t, "glob match failed", []string{"/mock/mock.exe"}, g)
}

func TestParseTemplates(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/", "mock/bar.txt:/bar.txt", "mock/foo.txt:/foo.txt", "mock/foofunc.txt:/foofunc.txt")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	tpl, err := ParseTemplates(nil, fs, "/bar.txt")
	assert(t, "error parsing template", nil, err)

	b := bytes.Buffer{}
	err = tpl.Execute(&b, nil)
	assert(t, "template execute failed", nil, err)
	assert(t, "mismatch in executed template", "bar", b.String())

	// Template func map.
	mp := map[string]interface{}{
		"Foo": func() string {
			return "func"
		},
	}
	tpl, err = ParseTemplates(mp, fs, "/foo.txt", "/foofunc.txt")
	assert(t, "error parsing template", nil, err)
	b.Reset()
	err = tpl.Execute(&b, nil)
	assert(t, "template execute failed", nil, err)
	assert(t, "mismatch in executed template", "foo\nfoo - func\n", b.String())
}

func TestParseTemplatesGlob(t *testing.T) {
	// Template func map.
	mp := map[string]interface{}{
		"Foo": func() string {
			return "func"
		},
	}

	fs, err := NewLocalFS("/", "mock/", "mock/foo.txt:/foo.txt", "mock/foofunc.txt:/foofunc.txt")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	tpl, err := ParseTemplatesGlob(mp, fs, "/*.txt")
	assert(t, "error parsing template", nil, err)

	b := bytes.Buffer{}
	err = tpl.Execute(&b, nil)
	assert(t, "template execute failed", nil, err)
	assert(t, "mismatch in executed template", "foo\nfoo - func\n", b.String())
}

func TestExecuteTemplate(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/:/")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	tpl, err := ParseTemplates(nil, fs, "/foo.txt", "/bar.txt", "/subdir/baz.txt")
	assert(t, "error parsing template", nil, err)

	b := bytes.Buffer{}
	err = tpl.ExecuteTemplate(&b, "subdir/baz.txt", nil)
	assert(t, "template execute failed", nil, err)
	assert(t, "mismatch in executed template", "baz\n", b.String())
}

func TestExecuteTemplateGlob(t *testing.T) {
	// Template func map.
	mp := map[string]interface{}{
		"Foo": func() string {
			return "func"
		},
	}

	fs, err := NewLocalFS("/", "mock/:/")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	tpl, err := ParseTemplatesGlob(mp, fs, "/*.txt")
	assert(t, "error parsing template", nil, err)

	b := bytes.Buffer{}
	err = tpl.ExecuteTemplate(&b, "bar.txt", nil)
	assert(t, "template execute failed", nil, err)
	assert(t, "mismatch in executed template", "bar", b.String())
}

func TestNamedTemplates(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/", "mock/bar.txt:/bar.txt", "mock/foo.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)
	if fs == nil {
		return
	}

	tpl, err := ParseTemplatesGlob(nil, fs, "/*.txt")
	assert(t, "error parsing template", nil, err)

	names := make([]string, len(tpl.Templates()))
	for i, template := range tpl.Templates() {
		names[i] = template.Name()
	}
	sort.Strings(names)

	assert(t, "template execute failed", nil, err)
	assert(t, "mismatch in executed template", []string{"", "bar.txt", "foo.txt"}, names)
}

func TestMerge(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/", "mock/foo.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)

	fs2, err := NewLocalFS("/", "mock/bar.txt:/bar.txt", "mock/foofunc.txt:/foofunc.txt")
	assert(t, "error creating local FS", nil, err)

	err = fs.Merge(fs2)
	assert(t, "error merging FS", nil, err)

	_, err = fs.Get("/bar.txt")
	assert(t, "merged file not found", nil, err)
	_, err = fs.Get("/foofunc.txt")
	assert(t, "merged file not found", nil, err)
}

func TestMergeOverwrite(t *testing.T) {
	fs, err := NewLocalFS("/", "mock/", "mock/foo.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)

	// Load baz.txt as foo.txt
	fs2, err := NewLocalFS("/", "mock/subdir/baz.txt:/foo.txt")
	assert(t, "error creating local FS", nil, err)

	err = fs.Merge(fs2)
	assert(t, "error merging FS", nil, err)

	_, err = fs.Get("/foo.txt")
	assert(t, "merged file not found", nil, err)

	// "foo" should've been written with "baz" after the merge.
	b, err := fs.Get("/foo.txt")
	assert(t, "merged value doesn't match", "baz\n", string(b.ReadBytes()))
}
