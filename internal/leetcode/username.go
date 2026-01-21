package leetcode

import (
	"fmt"
	"regexp"
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

func ValidateUsername(username string) error {
	if !usernameRe.MatchString(username) {
		return fmt.Errorf("invalid username %q", username)
	}
	return nil
}
