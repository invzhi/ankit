package leetcode

// Lang resprents a programming language on leetcode.
type Lang string

// programming languages supported on leetcode.
const (
	C          Lang = "c"
	Cpp        Lang = "cpp"
	CSharp     Lang = "csharp"
	Java       Lang = "java"
	Kotlin     Lang = "kotlin"
	Scala      Lang = "scala"
	Python     Lang = "python"
	Python3    Lang = "python3"
	Ruby       Lang = "ruby"
	JavaScript Lang = "javascript"
	Swift      Lang = "swift"
	Go         Lang = "golang"
	Rust       Lang = "rust"
)

// Valid check if Lang is supported on leetcode.
func (l Lang) Valid() bool {
	switch l {
	case C, Cpp, CSharp:
		return true
	case Java, Kotlin, Scala:
		return true
	case Python, Python3, Ruby:
		return true
	case JavaScript, Swift:
		return true
	case Go, Rust:
		return true
	default:
		return false
	}
}
