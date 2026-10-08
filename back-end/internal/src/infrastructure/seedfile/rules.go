// Package seedfile reads the JSON seed files under back-end/seeds.
package seedfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/domain/rule"
)

// ErrNoRules is returned when a rules seed file lists no rules.
var ErrNoRules = errors.New("seedfile: no rules")

// rulesFile is the shape of seeds/rules.json. Field names and values match the
// rules API (frequency: daily|weekly, scoreType: sum|decrease, score: 1-100).
type rulesFile struct {
	Rules []struct {
		Name      string         `json:"name"`
		Frequency rule.Frequency `json:"frequency"`
		ScoreType rule.ScoreType `json:"scoreType"`
		Score     int            `json:"score"`
	} `json:"rules"`
}

// ReadRules decodes a rules seed file. Unknown fields are rejected so typos
// fail loudly; the values themselves are validated by the domain when seeded.
func ReadRules(r io.Reader) ([]service.RuleInput, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var f rulesFile
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("seedfile: decode rules: %w", err)
	}
	if len(f.Rules) == 0 {
		return nil, ErrNoRules
	}
	rules := make([]service.RuleInput, len(f.Rules))
	for i, x := range f.Rules {
		rules[i] = service.RuleInput{Name: x.Name, Frequency: x.Frequency, ScoreType: x.ScoreType, Score: x.Score}
	}
	return rules, nil
}
