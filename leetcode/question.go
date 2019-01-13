package leetcode

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Question resprents a Question in leetcode repo.
type Question struct {
	repo *Repo

	ID          int    `db:"id"`
	TitleSlug   string `db:"title_slug"`
	Title       string `db:"title"`
	Content     string `db:"content"`
	Difficulty  string `db:"difficulty"`
	Tags        string `db:"tags"`
	CodeSnippet string `db:"code_snippet"`
	Code        string
}

// Fields returns the string fields of the question. If the question has a error, return nil.
func (q *Question) Fields() []string {
	return []string{
		strconv.Itoa(q.ID),
		q.TitleSlug,
		q.Title,
		q.Content,
		q.Difficulty,
		q.CodeSnippet,
		q.Code,
		q.Tags,
	}
}

// Key is the type of function. It use ID or TitleSlug to get question info from db.
type Key func(*Question) error

func KeyID(id int) Key {
	return func(q *Question) error {
		return q.getByID(id)
	}
}

func KeyTitleSlug(slug string) Key {
	return func(q *Question) error {
		return q.getByTitleSlug(slug)
	}
}

type keyPath struct {
	key  Key
	path string
}

func (q *Question) empty() bool {
	return q.Title == ""
}

func (q *Question) getByID(id int) error {
	const query = "SELECT * FROM questions WHERE id=?"
	return q.repo.db.Get(q, query, id)
}

func (q *Question) getByTitleSlug(slug string) error {
	const query = "SELECT * FROM questions WHERE title_slug=?"
	return q.repo.db.Get(q, query, slug)
}

func (q *Question) update() error {
	const query = "UPDATE questions SET title=?, content=?, difficulty=?, tags=?, code_snippet=? WHERE id=?"
	_, err := q.repo.db.Exec(query, q.Title, q.Content, q.Difficulty, q.Tags, q.CodeSnippet, q.ID)
	return err
}

func (q *Question) fetch() error {
	const (
		url  = "https://leetcode.com/graphql"
		data = `{"operationName":"question","variables":{"titleSlug":"???"},"query":"query question($titleSlug: String!) {\n  question(titleSlug: $titleSlug) {\n    questionId\n    questionFrontendId\n    boundTopicId\n    title\n    content\n    translatedTitle\n    translatedContent\n    isPaidOnly\n    difficulty\n    likes\n    dislikes\n    isLiked\n    similarQuestions\n    contributors {\n      username\n      profileUrl\n      avatarUrl\n      __typename\n    }\n    langToValidPlayground\n    topicTags {\n      name\n      slug\n      translatedName\n      __typename\n    }\n    companyTagStats\n    codeSnippets {\n      lang\n      langSlug\n      code\n      __typename\n    }\n    stats\n    hints\n    solution {\n      id\n      canSeeDetail\n      __typename\n    }\n    status\n    sampleTestCase\n    metaData\n    judgerAvailable\n    judgeType\n    mysqlSchemas\n    enableRunCode\n    enableTestMode\n    envInfo\n    __typename\n  }\n}\n"}`
	)

	log.Printf("fetching question %d. %s from leetcode api...", q.ID, q.TitleSlug)

	r := strings.NewReader(strings.Replace(data, "???", q.TitleSlug, 1))

	req, err := http.NewRequest(http.MethodPost, url, r)
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := q.repo.client.Do(req)
	if err != nil {
		return err
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
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
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
		if snippet.Lang == string(q.repo.lang) {
			q.CodeSnippet = snippet.Code
			break
		}
	}

	return nil
}
