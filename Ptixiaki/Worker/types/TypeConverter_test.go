package types

import "testing"

func TestParseViewAndFacadeAreCaseInsensitive(t *testing.T) {
	t.Parallel()

	forest, mountain, sea, unlimited, noView := ParseView([]string{"Forest", "SEA", "unlimited"})
	if forest != 1 || mountain != 0 || sea != 1 || unlimited != 1 || noView != 0 {
		t.Fatalf("unexpected view flags: forest=%d mountain=%d sea=%d unlimited=%d noView=%d", forest, mountain, sea, unlimited, noView)
	}

	corner, frontage, interior, through, none := ParseFacade([]string{"Corner", "FRONTAGE", "through"})
	if corner != 1 || frontage != 1 || interior != 0 || through != 1 || none != 0 {
		t.Fatalf("unexpected facade flags: corner=%d frontage=%d interior=%d through=%d none=%d", corner, frontage, interior, through, none)
	}
}
