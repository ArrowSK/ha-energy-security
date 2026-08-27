package topology

import (
	"fmt"
	"testing"
	"time"
)

func TestRetentionNeverExceeds100CalendarDays(t *testing.T) {
	now := time.Date(2026, 8, 27, 12, 0, 0, 0, time.UTC)
	rel := &relationData{
		ParentDeviceID: "parent",
		ChildDeviceID:  "child",
		Days:           map[string]*dailyEvidence{},
	}
	for i := 0; i < 110; i++ {
		day := now.AddDate(0, 0, -i).Format("2006-01-02")
		rel.Days[day] = &dailyEvidence{Matches: 1, RatioSum: 1, RatioCount: 1}
	}
	l := &Learner{store: persistedStore{Version: 1, Relations: map[string]*relationData{"parent|child": rel}}}
	l.pruneLocked(now)
	if got := len(rel.Days); got != RetentionDays {
		t.Fatalf("retained %d days, want %d", got, RetentionDays)
	}
	oldestKept := now.AddDate(0, 0, -(RetentionDays - 1)).Format("2006-01-02")
	if _, ok := rel.Days[oldestKept]; !ok {
		t.Fatalf("expected oldest retained day %s", oldestKept)
	}
	tooOld := now.AddDate(0, 0, -RetentionDays).Format("2006-01-02")
	if _, ok := rel.Days[tooOld]; ok {
		t.Fatalf("day older than retention window was retained: %s", tooOld)
	}
}

func TestRelationshipPromotionRequiresRepeatedEvidence(t *testing.T) {
	makeRel := func(matches, on, off, streak int) *relationData {
		return &relationData{
			ParentDeviceID: "parent", ChildDeviceID: "child", CurrentStreak: streak, BestStreak: streak,
			Days: map[string]*dailyEvidence{"2026-08-27": {
				Matches: matches, OnMatches: on, OffMatches: off, RatioSum: float64(matches) * 1.07, RatioCount: matches,
			}},
		}
	}
	if got := relationshipStats(makeRel(3, 3, 0, 3)).Status; got != "suspected" {
		t.Fatalf("3 clean matches should be suspected, got %s", got)
	}
	if got := relationshipStats(makeRel(5, 5, 0, 5)).Status; got != "strong" {
		t.Fatalf("5 clean matches should be strong, got %s", got)
	}
	confirmed := relationshipStats(makeRel(8, 4, 4, 8))
	if confirmed.Status != "confirmed" {
		t.Fatalf("8 bidirectional clean matches should be confirmed, got %s", confirmed.Status)
	}
	if confirmed.LearnedFactor != 1.07 {
		t.Fatalf("learned factor = %.3f, want 1.070", confirmed.LearnedFactor)
	}
}

func TestContradictionsPreventFalsePromotion(t *testing.T) {
	rel := &relationData{
		ParentDeviceID: "parent", ChildDeviceID: "child", CurrentStreak: 0, BestStreak: 3,
		Days: map[string]*dailyEvidence{"2026-08-27": {
			Matches: 3, Contradictions: 3, OnMatches: 3, RatioSum: 3, RatioCount: 3,
		}},
	}
	got := relationshipStats(rel)
	if got.Status != "learning" {
		t.Fatalf("50%% support must not promote a relationship, got %s", got.Status)
	}
}

func TestConfirmedTransitiveAncestorIsNotDirect(t *testing.T) {
	rels := []Relationship{
		{ChildDeviceID: "child", ParentDeviceID: "ups", Status: "confirmed", Direct: true},
		{ChildDeviceID: "ups", ParentDeviceID: "wall", Status: "confirmed", Direct: true},
		{ChildDeviceID: "child", ParentDeviceID: "wall", Status: "confirmed", Direct: true},
	}
	markTransitiveAncestors(rels)
	for _, rel := range rels {
		if rel.ChildDeviceID == "child" && rel.ParentDeviceID == "wall" {
			if rel.Direct {
				t.Fatal("transitive wall ancestor should not be marked direct")
			}
			return
		}
	}
	t.Fatal(fmt.Sprintf("transitive relationship not found: %#v", rels))
}
