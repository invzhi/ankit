package leetcode

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"github.com/invzhi/ankit"
)

// KeyFunc is the type of function to indicate questoin in Repo.
// The path argument is the relative path of Repo.path.
// The info argument is the os.FileInfo for the named path.
// See also https://golang.org/pkg/path/filepath/#WalkFunc
type KeyFunc func(path string, info os.FileInfo) (Key, error)

// CodeFunc is the type of function called for get question's code.
type CodeFunc func(path string, lang Lang) (string, error)

// Config is the config for Repo.
type Config struct {
	Path   string
	Source string
	Lang   string
}

// Valid check cfg.Path and cfg.Lang.
func (cfg Config) Valid() error {
	info, err := os.Lstat(cfg.Path)
	if os.IsNotExist(err) {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", cfg.Path)
	}

	if !Lang(cfg.Lang).Valid() {
		return fmt.Errorf("%s is not supported on leetcode", cfg.Lang)
	}

	return nil
}

// Repo represents a repo which store leetcode solution code.
type Repo struct {
	cfg Config

	db     *sqlx.DB
	lang   Lang
	client http.Client

	KeyFn  KeyFunc
	CodeFn CodeFunc
}

// NewRepo create a anki deck for leetcode repo.
func NewRepo(cfg Config, codeFn CodeFunc, keyFn KeyFunc) *Repo {
	const schema = `
	CREATE TABLE IF NOT EXISTS questions (
		id           INTEGER PRIMARY KEY,
		title_slug   TEXT,
		title        TEXT DEFAULT '',
		content      TEXT DEFAULT '',
		difficulty   TEXT DEFAULT '',
		tags         TEXT DEFAULT '',
		code_snippet TEXT DEFAULT ''
	);
	CREATE UNIQUE INDEX IF NOT EXISTS questions_title_slug_index ON questions (title_slug)`

	db := sqlx.MustOpen("sqlite3", cfg.Source)
	db.MustExec(schema)

	r := Repo{
		cfg:    cfg,
		db:     db,
		lang:   Lang(cfg.Lang),
		CodeFn: codeFn,
		KeyFn:  keyFn,
	}
	r.mustLoadKeys()

	return &r
}

// Notes returns all questions in Repo.
func (r *Repo) Notes() <-chan ankit.Note {
	notes := make(chan ankit.Note)

	go func() {
		err := filepath.Walk(r.cfg.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			rel, _ := filepath.Rel(r.cfg.Path, path)

			key, err := r.KeyFn(rel, info)
			if key != nil {
				notes <- r.Note(path, key)
			}

			return err
		})
		if err != nil {
			log.Printf("filepath.Walk error: %v", err)
		}

		close(notes)
	}()

	return notes
}

// Note returns a question in Repo with specific path and key.
// The path argument is the argument used to call r.CodeFn.
func (r *Repo) Note(path string, key Key) ankit.Note {
	q := &question{repo: r}
	if q.err = key(q); q.err != nil {
		return q
	}

	if q.empty() {
		if q.err = q.fetch(); q.err != nil {
			return q
		}
		if q.err = q.update(); q.err != nil {
			return q
		}
	}

	q.Code, q.err = r.CodeFn(path, r.lang)
	return q
}

func (r *Repo) mustLoadKeys() {
	if err := r.loadKeys(); err != nil {
		panic(err)
	}
}

func (r *Repo) loadKeys() error {
	const url = "https://leetcode.com/api/problems/all/"

	log.Print("fetching id and title_slug from leetcode api...")

	resp, err := r.client.Get(url)
	if err != nil {
		return err
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

	if err := json.NewDecoder(resp.Body).Decode(&questions); err != nil {
		return err
	}

	stmt, err := r.db.Prepare("INSERT OR IGNORE INTO questions(id, title_slug) VALUES(?, ?)")
	if err != nil {
		return err
	}

	for _, pair := range questions.StatStatusPairs {
		_, err = stmt.Exec(pair.Stat.FrontendQuestionID, pair.Stat.QuestionTitleSlug)
		if err != nil {
			return err
		}
	}

	return nil
}
