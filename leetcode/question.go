package leetcode

import (
	"strconv"

	"github.com/pkg/errors"

	"github.com/invzhi/ankit"
)

type question struct {
	l *LeetCode

	ID          int    `db:"id"`
	TitleSlug   string `db:"title_slug"`
	Title       string `db:"title"`
	Content     string `db:"content"`
	Difficulty  string `db:"difficulty"`
	Tags        string `db:"tags"`
	CodeSnippet string `db:"code_snippet"`
	Code        string
}

// Fields implements the ankit.Note interface.
func (q *question) Fields() []string {
	return []string{
		strconv.Itoa(q.ID),
		q.TitleSlug,
		q.Title,
		q.Content,
		q.Difficulty,
		q.Tags,
		q.CodeSnippet,
		q.Code,
	}
}

// Question get question info by id from db, get question solution code from dir.
// If question's info is empty in db, fetch info from leetcode.com api.
func (l *LeetCode) Question(id int, dir string) (ankit.Note, error) {
	q := question{l: l}
	if err := q.get(id); err != nil {
		return nil, errors.Wrapf(err, "cannot get question from db by id %d", id)
	}

	if q.empty() {
		if err := q.fetch(); err != nil {
			return nil, errors.Wrap(err, "cannot fetch question info from leetcode.com")
		}
		if err := q.update(); err != nil {
			return nil, errors.Wrap(err, "cannot update question info to db")
		}
	}

	var err error
	q.Code, err = l.CodeFn(dir, l.lang)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get question's code")
	}

	return &q, nil
}

func (q *question) empty() bool {
	return q.Title == ""
}

func (q *question) get(id int) error {
	const query = "SELECT * FROM questions WHERE id=?"
	return q.l.db.Get(q, query, id)
}

func (q *question) update() error {
	const query = "UPDATE questions SET title=?, content=?, difficulty=?, tags=?, code_snippet=? WHERE id=?"
	_, err := q.l.db.Exec(query, q.Title, q.Content, q.Difficulty, q.Tags, q.CodeSnippet, q.ID)
	return err
}
