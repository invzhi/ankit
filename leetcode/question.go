package leetcode

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

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

func (q *question) fetch() error {
	const (
		url  = "https://leetcode.com/graphql"
		data = `{"operationName":"question","variables":{"titleSlug":"???"},"query":"query question($titleSlug: String!) {\n  question(titleSlug: $titleSlug) {\n    questionId\n    questionFrontendId\n    boundTopicId\n    title\n    content\n    translatedTitle\n    translatedContent\n    isPaidOnly\n    difficulty\n    likes\n    dislikes\n    isLiked\n    similarQuestions\n    contributors {\n      username\n      profileUrl\n      avatarUrl\n      __typename\n    }\n    langToValidPlayground\n    topicTags {\n      name\n      slug\n      translatedName\n      __typename\n    }\n    companyTagStats\n    codeSnippets {\n      lang\n      langSlug\n      code\n      __typename\n    }\n    stats\n    hints\n    solution {\n      id\n      canSeeDetail\n      __typename\n    }\n    status\n    sampleTestCase\n    metaData\n    judgerAvailable\n    judgeType\n    mysqlSchemas\n    enableRunCode\n    enableTestMode\n    envInfo\n    __typename\n  }\n}\n"}`
	)

	log.Printf("fetching question %d from leetcode.com api...", q.ID)

	r := strings.NewReader(strings.Replace(data, "???", q.TitleSlug, 1))

	req, err := http.NewRequest(http.MethodPost, url, r)
	if err != nil {
		return errors.Wrap(err, "cannot create a http request")
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := q.l.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "cannot send a http request")
	}
	defer resp.Body.Close()

	var body struct {
		Data struct {
			Question struct {
				Title      string `json:"title"`
				Content    string `json:"content"`
				Difficulty string `json:"difficulty"`
				TopicTags  []struct {
					Slug string `json:"slug"`
				} `json:"topicTags"`
				CodeSnippets []struct {
					Lang string `json:"langSlug"`
					Code string `json:"code"`
				} `json:"codeSnippets"`
			} `json:"question"`
		} `json:"data"`
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&body); err != nil {
		return errors.Wrap(err, "cannot decode json")
	}

	q.Title = body.Data.Question.Title
	q.Content = body.Data.Question.Content
	q.Difficulty = body.Data.Question.Difficulty

	tags := make([]string, len(body.Data.Question.TopicTags))
	for i, tag := range body.Data.Question.TopicTags {
		tags[i] = tag.Slug
	}
	q.Tags = strings.Join(tags, " ")

	for _, snippet := range body.Data.Question.CodeSnippets {
		if snippet.Lang == string(q.l.lang) {
			q.CodeSnippet = snippet.Code
			break
		}
	}

	return nil
}
