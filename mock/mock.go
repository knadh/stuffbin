package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/knadh/stuffbin"
)

func main() {
	// Read stuffed data from self.
	fs, err := stuffbin.UnStuff(os.Args[0])
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

	// Expose an HTTP file server.
	// Try http://localhost:8000/static/foo.txt
	// Try http://localhost:8000/static/virtual/path/bar.txt
	// Try http://localhost:8000/static/subdir/baz.txt
	http.Handle("/static/", http.StripPrefix("/static/", fs.FileServer()))
	log.Println("listening on :8000")
	_ = http.ListenAndServe(":8000", nil)
}
