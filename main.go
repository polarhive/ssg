package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"gopkg.in/yaml.v2"
)

// frontmatter struct (metadata at the top of a md file)
type Frontmatter struct {
	Title string `yaml:"title"`
}

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
	helpFlag := flag.Bool("help", false, "show help message")
	serveFlag := flag.Bool("serve", false, "only run ssg serve after copying files")

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	srcDir := "static"
	dstDir := "rendered"

	// copy files from static/* to rendered/*
	if err := copyDir(srcDir, dstDir); err != nil {
		fmt.Println("error copying directory:", err)
		os.Exit(1)
	}

	// read markdown content
	markdownContent, err := os.ReadFile("content/post.md")
	if err != nil {
		fmt.Println("error reading markdown file:", err)
		os.Exit(1)
	}

	// parse frontmatter
	var fm Frontmatter
	content := string(markdownContent)
	if strings.HasPrefix(content, "---") {
		endIdx := strings.Index(content[3:], "---")
		if endIdx == -1 {
			fmt.Println("error: frontmatter end not found")
			os.Exit(1)
		}
		frontmatterContent := content[3 : endIdx+3]
		if err := yaml.Unmarshal([]byte(frontmatterContent), &fm); err != nil {
			fmt.Println("error parsing frontmatter:", err)
			os.Exit(1)
		}
		content = content[endIdx+6:]
		// rm frontmatter and newline
	}

	// setup goldmark
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
		),
	)

	// convert markdown to HTML
	var htmlContent bytes.Buffer
	if err := md.Convert([]byte(content), &htmlContent); err != nil {
		fmt.Println("error converting markdown to HTML:", err)
		os.Exit(1)
	}

	// read layout file
	layoutContent, err := os.ReadFile("theme/layout.html")
	if err != nil {
		fmt.Println("error reading layout file:", err)
		os.Exit(1)
	}

	// create template from layout
	tmpl, err := template.New("layout").Parse(string(layoutContent))
	if err != nil {
		fmt.Println("error parsing layout template:", err)
		os.Exit(1)
	}

	var finalOutput bytes.Buffer
	if err := tmpl.Execute(&finalOutput, struct {
		Title   string
		Content string
	}{
		Title:   fm.Title,
		Content: htmlContent.String(),
	}); err != nil {
		fmt.Println("error executing template:", err)
		os.Exit(1)
	}

	// write file to disk
	outputFile, err := os.Create("rendered/index.html")
	if err != nil {
		fmt.Println("error creating output file:", err)
		os.Exit(1)
	}
	defer outputFile.Close()
	if _, err := io.Copy(outputFile, &finalOutput); err != nil {
		fmt.Println("error writing HTML content to output file:", err)
		os.Exit(1)
	}

	if *serveFlag {
		http.Handle("/", http.FileServer(http.Dir("rendered")))
		fmt.Println("check: http://localhost:8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("error starting server:", err)
			os.Exit(1)
		}
	}
}
