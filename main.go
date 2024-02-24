package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// copy logic
func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			srcFile, err := os.Open(srcPath)
			if err != nil {
				return err
			}
			defer srcFile.Close()
			dstFile, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			defer dstFile.Close()
			if _, err := io.Copy(dstFile, srcFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	// serve flag
	serveFlag := flag.Bool("serve", false, "only run ssg serve after copying files")
	flag.Parse()

	// copy stuff from static/* -> rendered/* (for now)
	srcDir := "static"
	dstDir := "rendered"
	if err := copyDir(srcDir, dstDir); err != nil {
		fmt.Println("error copying directory:", err)
		os.Exit(1)
	}
	fmt.Println("site built successfully!")

	// serve the rendered/* dir
	if *serveFlag {
		http.Handle("/", http.FileServer(http.Dir("rendered")))
		fmt.Println("server started at http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("Error starting server:", err)
			os.Exit(1)
		}
	}
}
