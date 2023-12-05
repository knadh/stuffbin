package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/knadh/stuffbin"
)

const helpTxt = `
compress and embed static assets into Go binaries.
Usage: stuffbin -a build -in yourbinary.bin -out stuffed.bin /path/asset1 /path/asset2:/asset2 ...

The file paths to embed can be suffixed by a colon and an 
target (alias) path, for instance /original/local/path:/virtual/path.
When compressed and stuffed, the original path is overwritten
with the alias, which in turn can be used to access the file
from within the application.`

var (
	aID      = "id"
	aStuff   = "stuff"
	aUnstuff = "unstuff"
	aStrip   = "strip"

	logger = log.New(os.Stdout, "", 0)
)

// id shows the ID and stuffed files in a given binary.
func id(path string, l *log.Logger) error {
	id, err := stuffbin.GetFileID(path)
	if err != nil {
		if err == stuffbin.ErrNoID {
			return fmt.Errorf("%s: %v", path, err)
		}
		return fmt.Errorf("error reading file: %v", err)
	}

	l.Printf("%s: %s (%0.2f KB binary, %0.2f KB stuff)\n\n",
		path, id.Name, float64(id.BinSize)/1024, float64(id.ZipSize)/1024)

	// Get stuffed zip data.
	b, err := stuffbin.GetStuff(path)
	if err != nil {
		return err
	}

	// Unzip and list files.
	fs, err := stuffbin.UnZip(b)
	if err != nil {
		return err
	}

	l.Printf("%d files totalling %0.2f KB\n", fs.Len(), float64(fs.Size())/1024)
	for _, p := range fs.List() {
		f, _ := fs.Get(p)
		info, err := f.Stat()
		if err != nil {
			return fmt.Errorf("error reading %s: %v", p, err)
		}
		l.Printf("%0.2f KB \t\t %s", float64(info.Size())/1024, p)
	}

	return nil
}

// unstuff extracts the ZIP from a stuffed binary.
func unstuff(in, out string, l *log.Logger) error {
	id, err := stuffbin.GetFileID(in)
	if err != nil {
		if err == stuffbin.ErrNoID {
			return fmt.Errorf("%s: %v", in, err)
		}
		return fmt.Errorf("error reading file: %v", err)
	}

	l.Printf("%s: %s (%v bytes original binary, %v bytes zipped stuff)\n\n",
		in, id.Name, id.BinSize, id.ZipSize)

	// Get stuffed zip data.
	b, err := stuffbin.GetStuff(in)
	if err != nil {
		return err
	}

	// Write out.
	to, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, bytes.NewReader(b))
	if err != nil {
		to.Close()
		return err
	}
	l.Printf("wrote to %s", out)

	return nil
}

// strip strips the binary of stuffed files.
func strip(in, out string, l *log.Logger) error {
	id, err := stuffbin.GetFileID(in)
	if err != nil {
		if err == stuffbin.ErrNoID {
			return fmt.Errorf("%s: %v", in, err)
		}
		return fmt.Errorf("error reading file: %v", err)
	}

	l.Printf("%s: %s (%v bytes original binary, %v bytes zipped stuff)\n\n", in, id.Name, id.BinSize, id.ZipSize)

	from, err := os.Open(in)
	if err != nil {
		return err
	}
	defer from.Close()

	// Write out.
	to, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		to.Close()
		return err
	}

	// Truncate the file to its original length, losing the stuffed zip.
	if err := to.Truncate(int64(id.BinSize)); err != nil {
		l.Fatalf("error stripping binary: %v", err)
	}

	l.Printf("wrote stripped binary '%s'", out)

	return to.Sync()
}

func main() {
	var (
		fAction = flag.String("a", "", fmt.Sprintf("action (%s, %s, %s, %s)", aID, aStuff, aUnstuff, aStrip))
		fIn     = flag.String("in", "", "path to the input binary")
		fRoot   = flag.String("root", "/", "(optional) root path to bind all files to")
		fOut    = flag.String("out", "", "path to the output binary (stuff) or zip file (unstuff)")
	)

	// Usage help.
	flag.Usage = func() {
		logger.Printf("stuffbin\n")
		logger.Println(helpTxt)
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NFlag() == 0 {
		flag.Usage()
		return
	}

	// Validate actions.
	if *fAction != aID && *fAction != aStuff && *fAction != aUnstuff && *fAction != aStrip {
		logger.Fatal("unknown action")
	}

	// Validate input binary path.
	if *fIn == "" {
		logger.Fatal("provide an input path")
	}

	// Show the file ID.
	if *fAction == aID {
		if err := id(*fIn, logger); err != nil {
			logger.Fatal(err)
		}
		return
	}

	// Validate output binary path.
	if *fOut == "" {
		logger.Fatalf("provide an output path")
	}

	// Unstuff bundled files.
	if *fAction == aUnstuff {
		if err := unstuff(*fIn, *fOut, logger); err != nil {
			logger.Fatal(err)
		}
		return
	}

	// Strip binary of zip files.
	if *fAction == aStrip {
		if err := strip(*fIn, *fOut, logger); err != nil {
			logger.Fatal(err)
		}
		return
	}

	// Valid the list of files to embed.
	if flag.NArg() == 0 {
		logger.Fatalf("provide one or more files to embed")
	}

	// Build.
	binLen, zipLen, err := stuffbin.Stuff(*fIn, *fOut, *fRoot, flag.Args()...)
	if err != nil {
		logger.Fatalf("stuffing failed: %v", err)
	}
	logger.Printf("stuffing complete. binary size is %0.2f KB and stuffed zip size is %0.2f KB.",
		float64(binLen)/1024, float64(zipLen)/1024)
}
