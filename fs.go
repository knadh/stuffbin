package stuffbin

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// FileSystem represents a simple filesystem abstraction
// that implements the http.fileSystem interface.
type FileSystem interface {
	Add(f *File) error
	List() []string
	Len() int
	Size() int64
	Get(path string) (*File, error)
	Glob(pattern string) ([]string, error)
	Read(path string) ([]byte, error)
	Open(path string) (http.File, error)
	Delete(path string) error
	Merge(f FileSystem) error
	FileServer() http.Handler
}

// memFS implements an in-memory FileSystem.
type memFS struct {
	files map[string]*File

	// size is the total size of all files in the filesystem.
	size int64
}

// localFS implements a passthrough to the local filesystem.
type localFS struct {
	files map[string]*File

	// size is the total size of all files in the filesystem.
	size int64
}

// File represents an abstraction over http.File.
type File struct {
	path string
	info os.FileInfo
	b    []byte
	rd   *bytes.Reader
}

// ErrNotSupported indicates interface methods
// that are implemented but not supported.
var ErrNotSupported = errors.New("this method is not supported")

// NewFS returns a new instance of FileSystem.
func NewFS() (FileSystem, error) {
	return &memFS{
		files: make(map[string]*File),
	}, nil
}

// NewLocalFS returns a new instance of FileSystem
// with the given list of local files and directories mapped to it.
func NewLocalFS(rootPath string, paths ...string) (FileSystem, error) {
	fs, _ := NewFS()
	if err := walkPaths(func(srcPath, targetPath string, fInfo os.FileInfo) error {
		f, err := os.Open(srcPath)
		if err != nil {
			return err
		}

		// Copy bytes.
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, f)
		if err != nil {
			return err
		}

		// Add the file to the filesystem.
		return fs.Add(NewFile(targetPath, fInfo, buf.Bytes()))
	}, rootPath, paths...); err != nil {
		return nil, err
	}

	return fs, nil
}

// Add adds a file to the FileSystem.
func (fs *memFS) Add(f *File) error {
	p := f.Path()
	if _, ok := fs.files[p]; ok {
		return fmt.Errorf("file already exists: %v", p)
	}

	// Clean the path. This also ensures that all files are
	// always mounted to /. For instance, /mock/foo and mock/bar
	// will be mounted as /mock/foo and /mock/bar respectively.
	fs.files[cleanPath("", f.Path())] = f

	// Append the filesize to the FileSystem.
	s, err := f.Stat()
	if err != nil {
		return err
	}
	fs.size += s.Size()

	return nil
}

// List returns the list of the file paths in the FileSystem.
func (fs *memFS) List() []string {
	var (
		out = make([]string, len(fs.files))
		i   = 0
	)
	for p := range fs.files {
		out[i] = p
		i++
	}
	return out
}

// Len returns the number of files in the FileSystem.
func (fs *memFS) Len() int {
	return len(fs.files)
}

// Size returns the total size of all the files in the FileSystem.
func (fs *memFS) Size() int64 {
	return fs.size
}

// Get returns a copy of a File from the FileSystem by its path.
func (fs *memFS) Get(fPath string) (*File, error) {
	f, ok := fs.files[cleanPath("/", fPath)]
	if !ok {
		return nil, os.ErrNotExist
	}
	return NewFile(f.path, f.info, f.b), nil
}

// Glob returns the file paths in the filesystem matching
// a pattern.
func (fs *memFS) Glob(pattern string) ([]string, error) {
	var out []string
	for _, f := range fs.List() {
		ok, err := filepath.Match(pattern, f)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, f)
		}
	}

	return out, nil
}

// Read returns a copy of a File's bytes from the FileSystem by its path.
func (fs *memFS) Read(fPath string) ([]byte, error) {
	f, err := fs.Get(fPath)
	if err != nil {
		return nil, err
	}
	return f.ReadBytes(), nil
}

// Open returns an http.File from the Filesystem given its path.
func (fs *memFS) Open(path string) (http.File, error) {
	return fs.Get(path)
}

// Delete deletes the given path.
func (fs *memFS) Delete(fPath string) error {
	fPath = cleanPath("/", fPath)
	_, ok := fs.files[fPath]
	if !ok {
		return os.ErrNotExist
	}
	delete(fs.files, fPath)
	return nil
}

// Merge merges a given source FileSystem into this instance.
func (fs *memFS) Merge(src FileSystem) error {
	return MergeFS(fs, src)
}

// FileServer returns an http.Handler that serves the files from
// the file system like http.FileServer.
func (fs *memFS) FileServer() http.Handler {
	return http.FileServer(fs)
}

// NewFile creates and returns a new instance of File.
func NewFile(path string, info os.FileInfo, b []byte) *File {
	f := &File{
		path: path,
		info: info,
		b:    make([]byte, len(b)),
	}
	copy(f.b, b)
	f.rd = bytes.NewReader(f.b)
	return f
}

// Path returns the path of the file.
func (f *File) Path() string {
	return f.path
}

// ReadBytes returns the bytes of the given file.
func (f *File) ReadBytes() []byte {
	b := make([]byte, len(f.b))
	copy(b, f.b)
	return b
}

// Close emulates http.File's Close but internally,
// it simply seeks the File's reader to 0.
func (f *File) Close() error {
	_, err := f.Seek(0, 0)
	return err
}

// Read reads the file contents.
func (f *File) Read(b []byte) (int, error) {
	return f.rd.Read(b)
}

// Readdir is a dud.
func (f *File) Readdir(count int) ([]os.FileInfo, error) {
	return nil, ErrNotSupported
}

// Seek seeks the given offset in the file.
func (f *File) Seek(offset int64, whence int) (int64, error) {
	return f.rd.Seek(offset, whence)
}

// Stat returns the file's os.FileInfo.
func (f *File) Stat() (os.FileInfo, error) {
	return f.info, nil
}

func cleanPath(rootPath, p string) string {
	if rootPath == "" {
		rootPath = "/"
	}

	// A preceding / is attempted to get rid of paths
	// that begins with dots.
	if filepath.Separator == '/' {
		p = "/" + p
	}

	p = filepath.Join(rootPath, filepath.Clean(p))
	p = strings.Replace(p, `\`, "/", -1)
	p = strings.Replace(p, filepath.VolumeName(p), "", -1)
	return p
}

// ParseTemplatesGlob takes a file system, a file path pattern,
// and parses matching files into a template.Template with an
// optional template.FuncMap that will be applied to the compiled
// templates.
func ParseTemplatesGlob(f template.FuncMap, fs FileSystem, pattern string) (*template.Template, error) {
	paths, err := fs.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("pattern %s matches no files", pattern)
	}
	return ParseTemplates(f, fs, paths...)
}

// ParseTemplates takes a file system, a list of file paths,
// and parses them into a template.Template.
func ParseTemplates(f template.FuncMap, fs FileSystem, path ...string) (*template.Template, error) {
	tpl := template.New(filepath.Base(path[0]))
	if f != nil {
		tpl = tpl.Funcs(f)
	}

	if len(path) == 0 {
		return nil, fmt.Errorf("no files named in call to ParseTemplates")
	}

	for _, p := range path {
		f, err := fs.Read(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %v", p, err)
		}

		_, err = tpl.Parse(string(f))
		if err != nil {
			return nil, err
		}
	}

	return tpl, nil
}

// MergeFS merges FileSystem b into a, overwriting conflicting paths.
func MergeFS(dest FileSystem, src FileSystem) error {
	for _, path := range src.List() {
		// Get from target.
		f, err := src.Get(path)
		if err != nil {
			return err
		}

		// Check if the path exists in the target. If yes, remove.
		if err, _ := dest.Get(path); err != nil {
			if err := dest.Delete(path); err != nil {
				return err
			}
		}

		// Add to destination.
		dest.Add(f)
	}
	return nil
}
