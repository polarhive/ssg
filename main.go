package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// copy logic
func copyDir(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source directory %s does not exist", src)
	}
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
	serveFlag := flag.Bool("serve", false, "only run ssg serve after copying files")
	flag.Parse()

	// copy files from static/* to rendered/* (for now)
	srcDir := "static"
	dstDir := "rendered"

	// check for files in the static folder
	if err := copyDir(srcDir, dstDir); err != nil {
		fmt.Println("error copying directory:", err)
		os.Exit(1)
	}

	// parse markdown
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)

	// read content
	markdownContent, err := os.ReadFile("content/post.md")
	if err != nil {
		fmt.Println("error reading markdown file:", err)
		os.Exit(1)
	}

	// parse markdown content
	var htmlContent bytes.Buffer
	if err := md.Convert(markdownContent, &htmlContent); err != nil {
		fmt.Println("error converting markdown to HTML:", err)
		os.Exit(1)
	}

	// write HTML to disk
	outputFile, err := os.Create("rendered/index.html")
	if err != nil {
		fmt.Println("error creating output file:", err)
		os.Exit(1)
	}
	defer outputFile.Close()
	if _, err := io.Copy(outputFile, &htmlContent); err != nil {
		fmt.Println("error writing HTML content to output file:", err)
		os.Exit(1)
	}

	// serve the rendered/* dir
	if *serveFlag {
		http.Handle("/", http.FileServer(http.Dir("rendered")))
		fmt.Println("Server started at http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("Error starting server:", err)
			os.Exit(1)
		}
	}
}
