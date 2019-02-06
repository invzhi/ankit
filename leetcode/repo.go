package leetcode

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

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

// KeyFunc is the type of function to indicate question in Repo.
// The path argument is the relative path of Repo.path.
// The info argument is the os.FileInfo for the named path.
// See also https://golang.org/pkg/path/filepath/#WalkFunc
type KeyFunc func(path string, info os.FileInfo) (Key, error)

// CodeFunc is the type of function called for get question's code.
type CodeFunc func(path string, lang Lang) (string, error)

// Repo represents a repo which store leetcode solution code.
type Repo struct {
	ctx    context.Context
	cancel context.CancelFunc

	kpc chan keyPath

	cfg Config

	db     *sqlx.DB
	lang   Lang
	client http.Client

	KeyFn  KeyFunc
	CodeFn CodeFunc
}

// NewRepo represents a leetcode repo.
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

func (r *Repo) do() {
	r.kpc = make(chan keyPath)
	r.ctx, r.cancel = context.WithCancel(context.Background())

	go func() {
		err := filepath.Walk(r.cfg.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			rel, _ := filepath.Rel(r.cfg.Path, path)

			key, err := r.KeyFn(rel, info)
			if key != nil {
				select {
				case <-r.ctx.Done():
					return r.ctx.Err()
				case r.kpc <- keyPath{key: key, path: path}:
				}
			}

			return err
		})
		if err != nil {
			log.Printf("filepath.Walk error: %v", err)
		}

		close(r.kpc)
	}()
}

// Close close the Repo, stop filepath.Walk.
func (r *Repo) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	return nil
}

// Read reads fields from questions.
func (r *Repo) Read() ([]string, error) {
	if r.kpc == nil {
		r.do()
	}

	kp, ok := <-r.kpc
	if !ok {
		return nil, io.EOF
	}

	q, err := r.Question(kp.key, kp.path)
	if err != nil {
		return nil, err
	}

	return q.Fields(), nil
}

// Question returns a Question by key and path in Repo.
func (r *Repo) Question(key Key, path string) (*Question, error) {
	q := Question{repo: r}
	if err := key(&q); err != nil {
		return nil, err
	}

	if q.empty() {
		if err := q.fetch(); err != nil {
			return nil, err
		}
		if err := q.update(); err != nil {
			return nil, err
		}
	}

	code, err := r.CodeFn(path, r.lang)
	if err != nil {
		return nil, err
	}
	q.Code = html.EscapeString(code)

	return &q, nil
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
