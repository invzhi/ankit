# ankit
:hammer: CSV export tool for Anki.

![Anki](https://i.imgur.com/kVyHst0.png)

## leetcode2anki

`leetcode2anki` will print CSV data from your LeetCode repository and corresponding question data.
It use SQLite to store LeetCode question data. If data is not exist, it will be fetched from LeetCode API.

Therefore, you need to import CSV to your Anki.

### Export LeetCode Repository to CSV

```sh
leetcode2anki -lang golang > leetcode.csv
```

#### Customize for your repository structure

If your LeetCode repository is different from mine, you need to change `question` and `code` function in `leetcode2anki/main.go`:

My [LeetCode](https://github.com/invzhi/LeetCode) Repository structure:
```
LeetCode
├── 0003
│   ├── code.go
│   └── code_test.go
├── 0004
│   ├── code.go
│   └── code_test.go
└── 0007
    ├── code.go
    └── code_test.go
```

```go
func question(path string, info os.FileInfo) (leetcode.Key, error) {
	if path == "." || !info.IsDir() {
		return nil, nil
	}
	// only handle directory in repository
	id, err := strconv.Atoi(path)
	if err != nil {
		return nil, filepath.SkipDir
	}
	// identify leetcode question by id
	return leetcode.KeyID(id), filepath.SkipDir
}

func code(path string, _ leetcode.Lang) (string, error) {
	fset := token.NewFileSet()

	f, err := parser.ParseFile(fset, filepath.Join(path, "code.go"), nil, parser.ParseComments)
	if err != nil {
		return "", err
	}
	// start from import declarations, then format code
	var w strings.Builder
	if err := format.Node(&w, fset, f.Decls); err != nil {
		return "", err
	}

	return w.String(), nil
}
```

Python example:
```
LeetCode
├──add-two-numbers.py
├──reverse-integer.py
└──two-sum.py
```

```go
func question(path string, info os.FileInfo) (leetcode.Key, error) {
	if path == "." {
		return nil, nil
	}
	// skip directory in repository
	if info.IsDir() {
		return nil, filepath.SkipDir
	}

	filename := filepath.Base(path)
	ext := filepath.Ext(filename)
	// only handle python file
	if ext != ".py" {
		return nil, nil
	}
	// identify leetcode question by title slug: filename
	slug := strings.TrimSuffix(filename, ext)
	return leetcode.KeyTitleSlug(slug), nil
}

func code(path string, _ leetcode.Lang) (string, error) {
	b, err := ioutil.ReadFile(path)
	return string(b), err
}
```
### Import CSV to Anki

1. Get Anki note type from [Shared Deck](https://ankiweb.net/shared/info/1000896729).
2. Import CSV file in Anki.

![Import CSV](https://i.imgur.com/Gye2EVk.png)
