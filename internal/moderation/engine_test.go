package moderation

import (
	"testing"

	"github.com/enekos/errolan/internal/models"
)

func base() models.ModerationSettings {
	return models.DefaultModerationSettings(1)
}

func TestEvaluate_OpenAllowsNormalComment(t *testing.T) {
	got := Evaluate(base(), nil, Input{Body: "Hello world, this is fine."})
	if got.Action != ActionAllow {
		t.Fatalf("want allow, got %v (%s)", got.Action, got.Reason)
	}
}

func TestEvaluate_MinBodyLengthRejects(t *testing.T) {
	s := base()
	s.MinBodyLength = 5
	got := Evaluate(s, nil, Input{Body: "hi"})
	if got.Action != ActionReject {
		t.Fatalf("want reject, got %v", got.Action)
	}
}

func TestEvaluate_PreModerationHolds(t *testing.T) {
	s := base()
	s.Mode = "pre_moderation"
	got := Evaluate(s, nil, Input{Body: "anything"})
	if got.Action != ActionHold {
		t.Fatalf("want hold, got %v", got.Action)
	}
}

func TestEvaluate_LinkPolicyHold(t *testing.T) {
	s := base()
	s.LinkPolicy = "hold"
	got := Evaluate(s, nil, Input{Body: "see https://example.com"})
	if got.Action != ActionHold {
		t.Fatalf("want hold, got %v", got.Action)
	}
}

func TestEvaluate_AnonymousLinkRejects(t *testing.T) {
	s := base()
	s.AnonymousLinkPolicy = "reject"
	got := Evaluate(s, nil, Input{Body: "free stuff https://spam.example", Anonymous: true})
	if got.Action != ActionReject {
		t.Fatalf("want reject, got %v", got.Action)
	}
}

func TestEvaluate_MaxLinksHolds(t *testing.T) {
	s := base()
	s.MaxLinks = 1
	got := Evaluate(s, nil, Input{Body: "https://a.example https://b.example"})
	if got.Action != ActionHold {
		t.Fatalf("want hold, got %v", got.Action)
	}
}

func TestEvaluate_BlocklistKeywordHolds(t *testing.T) {
	rules := CompileRules([]*models.BlocklistEntry{{Kind: "keyword", Pattern: "buy now", Action: "hold"}})
	got := Evaluate(base(), rules, Input{Body: "Hey, BUY NOW, big sale!"})
	if got.Action != ActionHold {
		t.Fatalf("want hold, got %v", got.Action)
	}
}

func TestEvaluate_BlocklistRegexRejects(t *testing.T) {
	rules := CompileRules([]*models.BlocklistEntry{{Kind: "regex", Pattern: `\bcasino\b`, Action: "reject"}})
	got := Evaluate(base(), rules, Input{Body: "best Casino in town"})
	if got.Action != ActionReject {
		t.Fatalf("want reject, got %v", got.Action)
	}
}

func TestEvaluate_NewUserHold(t *testing.T) {
	s := base()
	s.HoldNewUsers = 3
	got := Evaluate(s, nil, Input{Body: "hello", AuthorCommentCount: 1})
	if got.Action != ActionHold {
		t.Fatalf("want hold, got %v", got.Action)
	}
	got = Evaluate(s, nil, Input{Body: "hello", AuthorCommentCount: 5})
	if got.Action != ActionAllow {
		t.Fatalf("want allow, got %v", got.Action)
	}
}

func TestEvaluate_NewUserHoldSkippedForAnonymous(t *testing.T) {
	s := base()
	s.HoldNewUsers = 3
	got := Evaluate(s, nil, Input{Body: "hello", Anonymous: true, AuthorCommentCount: 0})
	if got.Action != ActionAllow {
		t.Fatalf("anonymous shouldn't trigger new-user hold, got %v", got.Action)
	}
}

func TestEvaluate_RejectBeatsHold(t *testing.T) {
	s := base()
	s.Mode = "pre_moderation" // would hold
	rules := CompileRules([]*models.BlocklistEntry{{Kind: "keyword", Pattern: "spam", Action: "reject"}})
	got := Evaluate(s, rules, Input{Body: "spam spam"})
	if got.Action != ActionReject {
		t.Fatalf("reject must win over hold, got %v", got.Action)
	}
}

func TestValidateRule(t *testing.T) {
	if err := ValidateRule("regex", "[", "hold"); err == nil {
		t.Fatal("expected invalid regex error")
	}
	if err := ValidateRule("keyword", "ok", "ban"); err == nil {
		t.Fatal("expected invalid action error")
	}
	if err := ValidateRule("keyword", "ok", "hold"); err != nil {
		t.Fatalf("expected valid rule, got %v", err)
	}
}
