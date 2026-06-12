package mention

import "testing"

func TestExtractUsernames(t *testing.T) {
	result := ExtractUsernames("Hi @Alice, @bob_123 and @Alice again. email a@example.com @@ignored @ab @toolongtoolongtoolongtoolongtoolong")

	if len(result) != 2 {
		t.Fatalf("expected two usernames, got %#v", result)
	}
	if result[0].String() != "alice" || result[1].String() != "bob_123" {
		t.Fatalf("unexpected usernames: %#v", result)
	}
}

func TestAddedUsernames(t *testing.T) {
	result := AddedUsernames("hello @alice @bob", "hello @alice @carol @bob @Carol")

	if len(result) != 1 || result[0].String() != "carol" {
		t.Fatalf("expected only newly added carol mention, got %#v", result)
	}
}
