package leetcode

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

const (
	graphqlAPIURL = "https://leetcode.com/graphql"
	listAPIURL    = "https://leetcode.com/api/problems/all/"

	data = `{"operationName":"question","variables":{"titleSlug":"???"},"query":"query question($titleSlug: String!) {\n  question(titleSlug: $titleSlug) {\n    questionId\n    questionFrontendId\n    boundTopicId\n    title\n    content\n    translatedTitle\n    translatedContent\n    isPaidOnly\n    difficulty\n    likes\n    dislikes\n    isLiked\n    similarQuestions\n    contributors {\n      username\n      profileUrl\n      avatarUrl\n      __typename\n    }\n    langToValidPlayground\n    topicTags {\n      name\n      slug\n      translatedName\n      __typename\n    }\n    companyTagStats\n    codeSnippets {\n      lang\n      langSlug\n      code\n      __typename\n    }\n    stats\n    hints\n    solution {\n      id\n      canSeeDetail\n      __typename\n    }\n    status\n    sampleTestCase\n    metaData\n    judgerAvailable\n    judgeType\n    mysqlSchemas\n    enableRunCode\n    enableTestMode\n    envInfo\n    __typename\n  }\n}\n"}`
)

func (l *LeetCode) questionInfo() error {
	log.Print("fetching id and title_slug from leetcode.com api...")

	resp, err := l.client.Get(listAPIURL)
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

func (q *question) fetch() error {
	log.Printf("fetching question %d from leetcode.com api...", q.ID)

	r := strings.NewReader(strings.Replace(data, "???", q.TitleSlug, 1))

	req, err := http.NewRequest(http.MethodPost, graphqlAPIURL, r)
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
