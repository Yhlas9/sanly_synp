package store

import (
	"encoding/json"
	"math"
	"strings"
	"unicode"
)

// Виды заданий из раздела 6 технического задания.
const (
	KindChoice    = "choice"    // jogap saýlamak
	KindShort     = "short"     // gysga jogap
	KindNumeric   = "numeric"   // sanly jogap
	KindMatch     = "match"     // gabat getirmek
	KindOrder     = "order"     // yzygiderlilik düzmek
	KindOpen      = "open"      // açyk jogap — проверяет учитель
	KindPractical = "practical" // amaly iş — проверяет учитель
)

// Grade возвращает долю набранного балла [0..1], признак полностью верного
// ответа и признак того, что задание требует ручной проверки учителем.
func Grade(kind string, answerKey, value json.RawMessage) (ratio float64, correct bool, needsManual bool) {
	switch kind {
	case KindOpen, KindPractical:
		return 0, false, true
	case KindChoice:
		r := gradeChoice(answerKey, value)
		return r, r >= 1, false
	case KindShort:
		r := gradeShort(answerKey, value)
		return r, r >= 1, false
	case KindNumeric:
		r := gradeNumeric(answerKey, value)
		return r, r >= 1, false
	case KindMatch:
		r := gradeMatch(answerKey, value)
		return r, r >= 1, false
	case KindOrder:
		r := gradeOrder(answerKey, value)
		return r, r >= 1, false
	default:
		return 0, false, true
	}
}

func gradeChoice(key, val json.RawMessage) float64 {
	var k struct {
		Correct []string `json:"correct"`
	}
	var v struct {
		Selected []string `json:"selected"`
	}
	if json.Unmarshal(key, &k) != nil || json.Unmarshal(val, &v) != nil {
		return 0
	}
	if len(k.Correct) == 0 {
		return 0
	}

	want := toSet(k.Correct)
	got := toSet(v.Selected)
	if len(want) != len(got) {
		return 0
	}
	for c := range want {
		if _, ok := got[c]; !ok {
			return 0
		}
	}
	return 1
}

func gradeShort(key, val json.RawMessage) float64 {
	var k struct {
		Accept []string `json:"accept"`
	}
	var v struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(key, &k) != nil || json.Unmarshal(val, &v) != nil {
		return 0
	}

	got := normalize(v.Text)
	if got == "" {
		return 0
	}
	for _, a := range k.Accept {
		if normalize(a) == got {
			return 1
		}
	}
	return 0
}

func gradeNumeric(key, val json.RawMessage) float64 {
	var k struct {
		Value     float64 `json:"value"`
		Tolerance float64 `json:"tolerance"`
		Unit      string  `json:"unit"`
	}
	var v struct {
		Value *float64 `json:"value"`
		Unit  string   `json:"unit"`
	}
	if json.Unmarshal(key, &k) != nil || json.Unmarshal(val, &v) != nil || v.Value == nil {
		return 0
	}
	// Единица измерения учитывается, только если задана в ключе.
	if k.Unit != "" && normalize(k.Unit) != normalize(v.Unit) {
		return 0
	}
	if math.Abs(*v.Value-k.Value) <= math.Abs(k.Tolerance) {
		return 1
	}
	return 0
}

// gradeMatch и gradeOrder дают частичный балл: ученик, собравший половину
// соответствий, получает половину балла.
func gradeMatch(key, val json.RawMessage) float64 {
	var k struct {
		Pairs map[string]string `json:"pairs"`
	}
	var v struct {
		Pairs map[string]string `json:"pairs"`
	}
	if json.Unmarshal(key, &k) != nil || json.Unmarshal(val, &v) != nil || len(k.Pairs) == 0 {
		return 0
	}

	hit := 0
	for left, right := range k.Pairs {
		if v.Pairs[left] == right {
			hit++
		}
	}
	return float64(hit) / float64(len(k.Pairs))
}

func gradeOrder(key, val json.RawMessage) float64 {
	var k struct {
		Order []string `json:"order"`
	}
	var v struct {
		Order []string `json:"order"`
	}
	if json.Unmarshal(key, &k) != nil || json.Unmarshal(val, &v) != nil || len(k.Order) == 0 {
		return 0
	}

	hit := 0
	for i, want := range k.Order {
		if i < len(v.Order) && v.Order[i] == want {
			hit++
		}
	}
	return float64(hit) / float64(len(k.Order))
}

func toSet(xs []string) map[string]struct{} {
	set := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		set[strings.TrimSpace(x)] = struct{}{}
	}
	return set
}

// normalize сводит ответ к сравнимому виду: регистр, пробелы и типографские
// знаки не должны влиять на оценку короткого ответа.
func normalize(s string) string {
	var b strings.Builder
	prevSpace := true

	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case unicode.IsSpace(r):
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		case r == '’' || r == '‘' || r == '`':
			b.WriteRune('\'')
			prevSpace = false
		default:
			b.WriteRune(r)
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}
