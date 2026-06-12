package mention

import (
	"strings"

	"github.com/Versifine/cumt-nexus-api/internal/user/userdomain"
)

func ExtractUsernames(text string) []userdomain.Username {
	runes := []rune(text)
	seen := make(map[string]bool)
	usernames := make([]userdomain.Username, 0)

	for index := 0; index < len(runes); index++ {
		if runes[index] != '@' {
			continue
		}
		if index > 0 && (isUsernameRune(runes[index-1]) || runes[index-1] == '@') {
			continue
		}

		start := index + 1
		end := start
		for end < len(runes) && isUsernameRune(runes[end]) {
			end++
		}
		if start == end {
			continue
		}

		username, err := userdomain.NewUsername(string(runes[start:end]))
		if err != nil {
			index = end - 1
			continue
		}
		key := username.String()
		if !seen[key] {
			usernames = append(usernames, username)
			seen[key] = true
		}
		index = end - 1
	}

	return usernames
}

func AddedUsernames(oldText string, newText string) []userdomain.Username {
	oldMentions := ExtractUsernames(oldText)
	oldSet := make(map[string]bool, len(oldMentions))
	for _, username := range oldMentions {
		oldSet[username.String()] = true
	}

	added := make([]userdomain.Username, 0)
	for _, username := range ExtractUsernames(newText) {
		if oldSet[username.String()] {
			continue
		}
		added = append(added, username)
	}
	return added
}

func isUsernameRune(value rune) bool {
	return strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_", value)
}
