package collect

import "regexp"

// taskPatterns match, in priority order, a task reference embedded in a branch
// name. First match wins.
var taskPatterns = []*regexp.Regexp{
	regexp.MustCompile(`#(\d+)`),                     // "#675", "fix-#42"
	regexp.MustCompile(`\b([A-Z]{2,}-\d+)\b`),        // "AUTH-12", "OBS-3"
	regexp.MustCompile(`(?:^|/)(\d{2,})(?:[-/_]|$)`), // "feat/675-s3", "675-x", "675"
}

// TaskFromBranch infers a task reference from a branch name, or "" if none.
// Examples: "feat/675-s3" → "#675", "AUTH-12-login" → "AUTH-12", "dev" → "".
func TaskFromBranch(branch string) string {
	for i, re := range taskPatterns {
		if m := re.FindStringSubmatch(branch); m != nil {
			if i == 1 { // prefixed keys (AUTH-12) are already canonical
				return m[1]
			}
			return "#" + m[1]
		}
	}
	return ""
}
