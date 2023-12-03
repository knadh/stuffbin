<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" /></a>

# stuffbin

stuffbin is a utility + package to compress and embed static files and assets into Go binaries for distribution. It supports falling back to the local file system when no embedded assets are available, for instance, in development mode. stuffbin is inspired by [zgok](https://github.com/srtkkou/zgok) but is simpler and has better abstractions.

## stuffbin vs. Go 1.16 embed
Go 1.16 introduced the [`//go:embed`](https://golang.org/pkg/embed/) directive that allows embedding of files into Go binaries without any external utilities. stuffbin offers a few key advantages over native embedding in its current form.

- All files are ZIP compressed.
- Custom path aliases (eg: embed `/home/local/path/file.txt` as `/app/file.txt`).
- Dynamically embed files instead of static `//go:embed` directives.
- Embed parent directories, sub-directories, and arbitrary paths (eg: `../../path`).
  Go embed does not permit embedding of files outside of a .go file's directory. This makes it difficult for programs structured in a `cmd` directory to embed files outside it.
- Better [filesystem abstraction](https://github.com/knadh/stuffbin/blob/e80f23deca3fbc201ae7fb5f59a9e4ea6f17878e/fs.go#L17) for virtual filesystem manipulation, including merging.

## How does it work?

![stuffbin](https://user-images.githubusercontent.com/547147/50650557-caa04680-0fa6-11e9-9f8e-4d76cf331dc6.png)

stuffbin compresses and embeds arbitrary files to the end of Go binaries. This does not affect the normal execution of the binary as the compressed data that is appended beyond the binary's original size is simply ignored by the operating system. When a stuffed application is executed, stuffbin reads the compressed bytes from self (the executable), uncompresses the files on the fly into an in-memory filesystem, and provides a FileSystem interface to access them. This enables Go applications that have external file dependencies to be shipped a single _fat_ binary, commonly, web applications that have static file and template dependencies.

- Built in ZIP compression
- A virtual filesystem abstraction to access embedded files
- Add static assets from nested directories recursively
- Re-path files and whole directories with the :suffix format, eg: ../my/original/file.txt:/my/virtual/file.txt and /my/nested/dir:/virtual/dir
- Template parsing helper similar to template.ParseGlob() to parse templates from the virtual filesystem
- Launch an http.FileServer for serving static files
- Gracefully failover to the local file system in the absence of embedded assets
- CLI to stuff, unstuff and extract, and list stuffed files in binaries

## Installation

```shell
go install github.com/knadh/stuffbin/...@latest
```

### Homebrew

For macOS/Linux users, you can install via [brew](https://brew.sh/)

```sh
$ brew install stuffbin
```

## Usage

#### Stuffing and embedding

```shell
# -a, -in, and -out params followed by the paths of files to embed.
# To normalize paths, aliases can be suffixed with a colon.
stuffbin -a stuff -in /path/to/exe -out /path/to/new.exe \
    static/file1.css static/file2.pdf /somewhere/else/file3.txt:/static/file3.txt
```

#### List files in a stuffed binary

```shell
stuffbin -a id -in /path/to/new/exe
```

#### Extract stuffed files from a binary

```shell
stuffbin -a unstuff -in /path/to/new/exe -out assets.zip
```

## In the application

To test this, `cd` into `./mock` and run `go run mock.go`

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/knadh/stuffbin"
)

func main() {
	// Get self executable path.
	path, err := os.Executable()
	if err != nil {
		log.Fatalf("error getting executable path: %v", err)
	}
	// Read stuffed data from self.
	fs, err := stuffbin.UnStuff(path)
	if err != nil {
		// Binary is unstuffed or is running in dev mode.
		// Can halt here or fall back to the local filesystem.
		if err == stuffbin.ErrNoID {
			// First argument is to the root to mount the files in the FileSystem
			// and the rest of the arguments are paths to embed.
			fs, err = stuffbin.NewLocalFS("/",
				"./", "bar.txt:/virtual/path/bar.txt")
			if err != nil {
				log.Fatalf("error falling back to local filesystem: %v", err)
			}
		} else {
			log.Fatalf("error reading stuffed binary: %v", err)
		}
	}

	fmt.Println("loaded files", fs.List())
	// Read the file 'foo'.
	f, err := fs.Get("foo.txt")
	if err != nil {
		log.Fatalf("error reading foo.txt: %v", err)
	}
	log.Println("foo.txt =", string(f.ReadBytes()))

	// Read the file 'bar'.
	f, err = fs.Get("/virtual/path/bar.txt")
	if err != nil {
		log.Fatalf("error reading /virtual/path/bar.txt: %v", err)
	}
	log.Println("/virtual/path/bar.txt =", string(f.ReadBytes()))

	fmt.Println("stuffed files:")
	for _, f := range fs.List() {
		fmt.Println("\t", f)
	}

	// Compile templates with the helpers:
	// err, tpl := stuffbin.ParseTemplatesGlob(nil, fs, "/templates/*.html")
	//
	// Template func map.
	// mp := map[string]interface{}{
	// 	"Foo": func() string {
	// 		return "func"
	// 	},
	// }
	// err, tpl := stuffbin.ParseTemplates(mp, fs, "/templates/index.html", "/templates/hello.html")

	// Expose an HTTP file server.
	// Try http://localhost:8000/static/foo.txt
	// Try http://localhost:8000/static/virtual/path/bar.txt
	// Try http://localhost:8000/static/subdir/baz.txt
	http.Handle("/static/", http.StripPrefix("/static/", fs.FileServer()))
	log.Println("listening on :8000")
	http.ListenAndServe(":8000", nil)
}
```

### License

Licensed under the MIT License.
