package main

import (
	"flag"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/invzhi/ankit"
	"github.com/invzhi/ankit/leetcode"
)

var (
	cfg  leetcode.Config
	spec string
)

func init() {
	flag.StringVar(&cfg.Path, "path", ".", "leetcode repo path")
	flag.StringVar(&cfg.Source, "db", "leetcode.db", "sqlite3 filename")
	flag.StringVar(&cfg.Lang, "lang", "golang", "programming language")

	flag.StringVar(&spec, "spec", "", "optional: the relative path of leetcode question that should be exported only")
}

func question(path string, info os.FileInfo) (leetcode.Key, error) {
	if path == "." || !info.IsDir() {
		return nil, nil
	}

	id, err := strconv.Atoi(path)
	if err != nil {
		return nil, filepath.SkipDir
	}

	return leetcode.KeyID(id), filepath.SkipDir
}

func code(path string, _ leetcode.Lang) (string, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filepath.Join(path, "code.go"), nil, parser.ParseComments)
	if err != nil {
		return "", err
	}

	var w strings.Builder
	if err := format.Node(&w, fset, f.Decls); err != nil {
		return "", err
	}

	return w.String(), nil
}

func main() {
	flag.Parse()

	if err := cfg.Valid(); err != nil {
		log.Fatal(err)
	}

	repo := leetcode.NewRepo(cfg, code, question)
	defer repo.Close()

	var r ankit.Reader = repo
	if spec != "" {
		path := filepath.Join(cfg.Path, spec)

		info, err := os.Lstat(path)
		if err != nil {
			log.Fatal(err)
		}

		key, err := repo.KeyFn(spec, info)
		if err != nil && err != filepath.SkipDir {
			log.Fatal(err)
		}

		q, err := repo.Question(key, path)
		if err != nil {
			log.Fatal(err)
		}

		r = ankit.OneNoteReader(q)
	}

	if err := ankit.Copy(os.Stdout, r); err != nil {
		log.Fatal(err)
	}
}
