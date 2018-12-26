package leetcode

import (
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/invzhi/ankit"
)

// CodeFunc is the type of function called for get leetcode question's code.
type CodeFunc func(dir string, lang Lang) (string, error)

// LeetCode ...
type LeetCode struct {
	db     *sqlx.DB
	path   string
	lang   Lang
	client http.Client
	CodeFn CodeFunc

	// CodeFromPath func(path string)

	// NoteDirs     func(path string) ([]string, error)
	// NoteFromPath func(path string) (ankit.Note, error)
}

// New create a anki deck for leetcode repo.
func New(path, dbfile string, lang Lang, codeFn CodeFunc) *LeetCode {
	db := sqlx.MustOpen("sqlite3", filepath.Join(path, dbfile))

	return &LeetCode{
		db:     db,
		path:   path,
		lang:   lang,
		CodeFn: codeFn,
	}
}

// Init create sqlite3 table if not exists for leetcode questions, then get their id and title_slug from leetcode.
func (l *LeetCode) Init() {
	const schema = `
		CREATE TABLE IF NOT EXISTS questions (
			id           INTEGER PRIMARY KEY,
			title_slug   TEXT,
			title        TEXT DEFAULT '',
			content      TEXT DEFAULT '',
			difficulty   TEXT DEFAULT '',
			tags         TEXT DEFAULT '',
			code_snippet TEXT DEFAULT ''
		)`
	l.db.MustExec(schema)
	l.questionInfo()
}

// Notes implements the ankit.Deck interface.
func (l *LeetCode) Notes() ([]ankit.Note, error) {
	dirs, err := ioutil.ReadDir(l.path)
	if err != nil {
		return nil, err
	}

	var notes []ankit.Note

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		id, err := strconv.Atoi(dir.Name())
		if err != nil {
			continue
		}

		note, err := l.Question(id, filepath.Join(dir.Name(), "code.go"))
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}

	return notes, nil
}
