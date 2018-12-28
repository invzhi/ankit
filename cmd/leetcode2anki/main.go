package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/invzhi/ankit"
	"github.com/invzhi/ankit/leetcode"
)

var (
	path    string
	dbfile  string
	csvfile string
	lang    string
	spec    string
)

func init() {
	flag.StringVar(&path, "path", ".", "leetcode repo path")
	flag.StringVar(&dbfile, "db", "leetcode.db", "sqlite3 filename")
	flag.StringVar(&csvfile, "file", "notes.txt", "exported csv filename")
	flag.StringVar(&lang, "lang", "golang", "programming language")

	flag.StringVar(&spec, "spec", "", "optional: the path of leetcode question that should be exported only")
}

func question(path string, info os.FileInfo) (leetcode.Key, error) {
	if path != "." && info.IsDir() {
		id, err := strconv.Atoi(path)
		if err != nil {
			return nil, filepath.SkipDir
		}

		return leetcode.KeyID(id), filepath.SkipDir
	}
	return nil, nil
}

func code(path string, _ leetcode.Lang) (string, error) {
	f, err := os.Open(filepath.Join(path, "code.go"))
	if err != nil {
		return "", err
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func main() {
	flag.Parse()

	lang := leetcode.Lang(lang)
	if !lang.Valid() {
		log.Fatalf("%s is unsupported on leetcode", lang)
	}

	f, err := os.Create(csvfile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	repo := leetcode.NewRepo(path, dbfile, lang, code, question)

	var keys []interface{}
	if spec != "" {
		keys = append(keys, spec)
	}

	if err := ankit.Export(f, repo, keys...); err != nil {
		log.Fatal(err)
	}
}
