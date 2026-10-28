package database

import (
	"strings"
	"testing"
)

func TestInferenceAllowsUnpricedTargetsWithoutWeakeningTraining(t *testing.T) {
	rules := Rules{}
	rules.Init()
	before := rules.BuildQuery("residence", "properties")
	inference := rules.BuildInferenceQuery("residence", "properties")
	for _, forbidden := range []string{"AND p.asking_price IS NOT NULL", "AND p.asking_price > 0", "OR p.asking_price > 1000"} {
		if strings.Contains(inference, forbidden) {
			t.Fatalf("inference still filters unpriced property: %s", forbidden)
		}
		if !strings.Contains(before, forbidden) {
			t.Fatalf("training lost label guard: %s", forbidden)
		}
	}
	if !strings.Contains(inference, "COALESCE(p.asking_price, 0) AS asking_price") {
		t.Fatal("nullable target must scan into legacy input struct")
	}
	if before != rules.BuildQuery("residence", "properties") {
		t.Fatal("inference mutated training rules")
	}
	if !strings.Contains(inference, "p.transaction_type = $1") {
		t.Fatal("inference lost transaction filter")
	}
}
