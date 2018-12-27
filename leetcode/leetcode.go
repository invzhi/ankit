package leetcode

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

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

	l := LeetCode{
		db:     db,
		path:   path,
		lang:   lang,
		CodeFn: codeFn,
	}
	l.init()

	return &l
}

// init create sqlite3 table if not exists for leetcode questions, then get their id and title_slug from leetcode.
func (l *LeetCode) init() {
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

func (l *LeetCode) questionInfo() error {
	const url = "https://leetcode.com/api/problems/all/"

	log.Print("fetching id and title_slug from leetcode.com api...")

	resp, err := l.client.Get(url)
	if err != nil {
		return errors.Wrap(err, "cannot get questions from leetcode")
	}
	defer resp.Body.Close()

	var questions struct {
		StatStatusPairs []struct {
			Stat struct {
				FrontendQuestionID int    `json:"frontend_question_id"`
				QuestionTitleSlug  string `json:"question__title_slug"`
			} `json:"stat"`
		} `json:"stat_status_pairs"`
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&questions); err != nil {
		return errors.Wrap(err, "cannot decode questions from json")
	}

	stmt, err := l.db.Prepare("INSERT OR IGNORE INTO questions(id, title_slug) VALUES(?, ?)")
	if err != nil {
		return errors.Wrap(err, "cannot prepare stmt")
	}

	for _, pair := range questions.StatStatusPairs {
		_, err = stmt.Exec(pair.Stat.FrontendQuestionID, pair.Stat.QuestionTitleSlug)
		if err != nil {
			return errors.Wrap(err, "cannot exec stmt")
		}
	}

	return nil
}
