package stuffbin

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// lenID is the length of the byte ID that's appended to binaries.
const lenID = 24

// WalkFunc is an abstraction over filepath.WalkFunc that's used as
// a callback to receive the real file path and their corresponding
// target (alias) paths from a real filepath.Walk() traversal of a list of
// file and directory paths.
type WalkFunc func(srcPath, targetPath string, fInfo os.FileInfo) error

// ID represents an identifier that is appended to binaries for identifying
// stuffbin binaries. The fields are appended as bytes totalling
// 8 + 12 + 8 + 8 = 36 bytes in the order Name BinSize ZipSize.
type ID struct {
	Name    [8]byte
	BinSize uint64
	ZipSize uint64
}

// ErrNoID is used to indicate if an ID was found in a file or not.
var ErrNoID = errors.New("no ID found in the file")

// buildName is the name of the app that's injected
var buildName = [8]byte{'s', 't', 'u', 'f', 'f', 'b', 'i', 'n'}

// Stuff takes the path to a binary, a list of file paths to stuff, and compresses
// the files and appends them to the end of the binary's body and writes everything
// to a new binary.
func Stuff(in, out, rootPath string, files ...string) (int64, int64, error) {
	z, err := zipFiles(rootPath, files...)
	if err != nil {
		return 0, 0, err
	}

	// Copy the binary and get the handle to append remaining data.
	outFile, origSize, err := copyFile(in, out)
	if err != nil {
		return 0, 0, err
	}
	defer outFile.Close()

	// Write compressed data and get the length.
	zLen, err := io.Copy(outFile, z)
	if err != nil {
		return 0, 0, err
	}

	// Write the ID at end.
	id := makeID(buildName, uint64(origSize), uint64(zLen))
	if _, err := outFile.Write(makeIDBytes(id)); err != nil {
		return 0, 0, err
	}

	return origSize, zLen, nil
}

// GetFileID attempts to get the stuffbin identifier from
// the end of the file and returns the identifier name
// and file sizes.
func GetFileID(fName string) (ID, error) {
	var id ID
	f, err := os.Open(fName)
	if err != nil {
		return id, err
	}
	defer f.Close()

	stat, err := os.Stat(fName)
	if err != nil {
		return id, err
	}

	var (
		buf   = make([]byte, lenID)
		start = stat.Size() - lenID
	)
	if start < 0 {
		return id, ErrNoID
	}

	_, err = f.ReadAt(buf, start)
	if err != nil {
		return id, err
	}

	if !bytes.Equal(buf[0:8], buildName[:]) {
		return id, ErrNoID
	}

	var name [8]byte
	copy(name[:], buf[0:8])
	return ID{
		Name:    name,
		BinSize: binary.BigEndian.Uint64(buf[8:16]),
		ZipSize: binary.BigEndian.Uint64(buf[16:24]),
	}, nil
}

// zipFiles takes a list of files and ZIPs them and returns the zipped bytes. It optionally
// flattens the paths (eg: /some/path/file.txt becomes /file.txt) and adds
// a base path (eg: /some/path/file.txt becomes /custombase/some/path/file.txt).
// The files list can have targetsaliases separated by a :, for instance
// /tmp/something/x:/assets/x, where the target followed by the colon is used as
// the file path when stuffing. This is useful to unify assets into a common path where  during
// the build process, the original assets can be scattered across different paths.
func zipFiles(rootPath string, paths ...string) (*bytes.Buffer, error) {
	var (
		buf = &bytes.Buffer{}
		zw  = zip.NewWriter(buf)
	)
	defer zw.Close()

	if err := walkPaths(func(srcPath, targetPath string, fInfo os.FileInfo) error {
		return zipFile(srcPath, targetPath, zw)
	}, rootPath, paths...); err != nil {
		return nil, err
	}

	return buf, nil
}

// zipFile reads and adds a single file from the local file system to a given zip.Writer
// while optionally losing the real path information (flattening)
// or subsituting it with an alias.
func zipFile(srcPath, targetPath string, zw *zip.Writer) error {
	z, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer z.Close()

	info, err := z.Stat()
	if err != nil {
		return err
	}

	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Append the optional alias.
	hdr.Name = targetPath
	hdr.Method = zip.Deflate

	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	if _, err = io.Copy(w, z); err != nil {
		return err
	}

	return nil
}

// copyFile takes an input file path, copies it to an output path
// and returns the size of the original file and the file handler
// of the new copy for further writing.
func copyFile(in string, out string) (*os.File, int64, error) {
	from, err := os.Open(in)
	if err != nil {
		return nil, 0, err
	}
	defer from.Close()

	// Get the source file's size.
	s, err := from.Stat()
	if err != nil {
		return nil, 0, err
	}
	curSize := s.Size()

	to, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return nil, 0, err
	}
	_, err = io.Copy(to, from)
	if err != nil {
		to.Close()
		return nil, 0, err
	}

	// Check if the binary is already stuffed. If yes, seek to the original
	// size of the bin so that the stuffed blob gets overwritten with the
	// new blob on write.
	old, _ := GetFileID(in)
	if old.BinSize > 0 {
		curSize = int64(old.BinSize)

		// Truncate the file to its original binary size.
		if err := to.Truncate(curSize); err != nil {
			return nil, 0, err
		}
		if _, err := to.Seek(curSize, 0); err != nil {
			return nil, 0, err
		}
	}

	return to, curSize, nil
}

func walkPaths(cb WalkFunc, rootPath string, paths ...string) error {
	for _, fp := range paths {
		var (
			chunks     = strings.Split(fp, ":")
			srcPath    = filepath.Clean(chunks[0])
			targetPath = ""
		)

		// Is there an alias (eg: /real/path:/alias/path)
		if len(chunks) > 2 {
			return fmt.Errorf("invalid alias format '%s'", fp)
		} else if len(chunks) == 2 {
			targetPath = cleanPath("/", chunks[1])
		}

		// If it's a directory, find its children.
		stat, err := os.Stat(srcPath)
		if err != nil {
			return err
		}

		if stat.IsDir() {
			if err := filepath.Walk(srcPath, func(p string, fInfo os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if fInfo.IsDir() {
					return nil
				}

				// If there's an alias, replace the whole dirpath with it.
				tp := p
				if targetPath != "" {
					tp = filepath.Join(targetPath, strings.TrimPrefix(p, srcPath))
				}

				return cb(p, filepath.Join(rootPath, tp), fInfo)
			}); err != nil {
				return err
			}

			continue
		}

		// Single file.
		if targetPath == "" {
			targetPath = cleanPath(rootPath, srcPath)
		}
		if err := cb(srcPath, targetPath, stat); err != nil {
			return err
		}
	}

	return nil
}

// makeID takes the individual ID fields and returns an ID.
func makeID(name [8]byte, binLen, zipLen uint64) ID {
	return ID{
		Name:    name,
		BinSize: binLen,
		ZipSize: zipLen,
	}
}

// makeIDBytes takes the values of an ID and returns them as a byte slice.
func makeIDBytes(id ID) []byte {
	b := make([]byte, lenID)
	copy(b[0:8], id.Name[:])
	binary.BigEndian.PutUint64(b[8:16], id.BinSize)
	binary.BigEndian.PutUint64(b[16:24], id.ZipSize)

	return b
}
