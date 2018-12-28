package leetcode

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"

	"github.com/invzhi/ankit"
)

// KeyFunc is the type of function called for each file or directory visited by filepath.Walk.
// The path argument is a relative path of Repo.path.
type KeyFunc func(path string, info os.FileInfo) (Key, error)

// CodeFunc is the type of function called for get leetcode question's code.
type CodeFunc func(path string, lang Lang) (string, error)

// Repo represents a repo which store leetcode solution code.
type Repo struct {
	path   string
	db     *sqlx.DB
	lang   Lang
	client http.Client

	KeyFn  KeyFunc
	CodeFn CodeFunc
}

// NewRepo create a anki deck for leetcode repo.
func NewRepo(path, dbfile string, lang Lang, codeFn CodeFunc, keyFn KeyFunc) ankit.Deck {
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

	db := sqlx.MustOpen("sqlite3", dbfile)
	db.MustExec(schema)

	r := Repo{
		db:     db,
		path:   path,
		lang:   lang,
		CodeFn: codeFn,
		KeyFn:  keyFn,
	}
	r.mustLoadKeys()

	return &r
}

// Note returns questions which can be retrieved by paths in LeetCode repo.
func (r *Repo) Note(paths ...interface{}) <-chan ankit.Note {
	notes := make(chan ankit.Note)

	go func() {
		for _, p := range paths {
			path, ok := p.(string)
			if !ok {
				log.Printf("%v is not string", p)
				continue
			}

			info, err := os.Lstat(path)
			if err != nil {
				log.Print(err)
				continue
			}

			rel, err := filepath.Rel(r.path, path)
			if err != nil {
				log.Print(err)
				continue
			}

			key, err := r.KeyFn(rel, info)
			if key != nil {
				notes <- r.note(path, key)
			}
			if err != nil && err != filepath.SkipDir {
				log.Printf("KeyFn error: %v", err)
			}
		}
		close(notes)
	}()

	return notes
}

// Notes returns all questions in LeetCode repo.
func (r *Repo) Notes() <-chan ankit.Note {
	notes := make(chan ankit.Note)

	go func() {
		err := filepath.Walk(r.path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			rel, _ := filepath.Rel(r.path, path)

			key, err := r.KeyFn(rel, info)
			if key != nil {
				notes <- r.note(path, key)
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

func (r *Repo) note(path string, key Key) ankit.Note {
	q := &question{repo: r}
	key(q)

	var err error

	switch {
	case q.ID != 0:
		err = q.getByID()
	case q.TitleSlug != "":
		err = q.getByTitleSlug()
	default:
		err = errors.New("leetcode.Question has no ID either TitleSlug")
	}

	if err != nil {
		q.err = err
		return q
	}

	if q.empty() {
		if err = q.fetch(); err != nil {
			q.err = err
			return q
		}
		if err = q.update(); err != nil {
			q.err = err
			return q
		}
	}

	q.Code, err = r.CodeFn(path, r.lang)
	if err != nil {
		q.err = err
		return q
	}

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

	if err := json.NewDecoder(resp.Body).Decode(&questions); err != nil {
		return errors.Wrap(err, "cannot decode questions from json")
	}

	stmt, err := r.db.Prepare("INSERT OR IGNORE INTO questions(id, title_slug) VALUES(?, ?)")
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
