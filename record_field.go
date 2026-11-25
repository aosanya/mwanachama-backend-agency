package mwanachamaagency

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/aosanya/mwanachama-backend-agency/models"
)

const (
	fieldKeyPrefix    = "f_"
	fieldKeyDigits    = 16
	fieldCodeLimit    = 40
	fieldCodeFallback = "field"
)

var (
	fieldCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	fieldKeyPattern  = regexp.MustCompile(`^f_[0-9a-f]{16}$`)
	notCodeCharacter = regexp.MustCompile(`[^a-z0-9]+`)
)

type fieldKeyMaker func(recordCode string, q models.StepQuestion, sameBefore int) string

func randomFieldKey(string, models.StepQuestion, int) string {
	raw := make([]byte, fieldKeyDigits/2)
	if _, err := rand.Read(raw); err != nil {
		panic(fmt.Sprintf("mwanachamaagency: no randomness for a field key: %v", err))
	}
	return fieldKeyPrefix + hex.EncodeToString(raw)
}

func legacyFieldKey(recordCode string, q models.StepQuestion, sameBefore int) string {
	sum := sha256.Sum256([]byte(recordCode + "\x00" + q.Prompt + "\x00" + strconv.Itoa(sameBefore)))
	return fieldKeyPrefix + hex.EncodeToString(sum[:])[:fieldKeyDigits]
}

func codeFromText(text, fallback string) string {
	code := strings.Trim(notCodeCharacter.ReplaceAllString(strings.ToLower(text), "_"), "_")
	if len(code) > fieldCodeLimit {
		code = strings.Trim(code[:fieldCodeLimit], "_")
	}
	if code == "" || code[0] < 'a' || code[0] > 'z' {
		code = strings.Trim(fallback+"_"+code, "_")
	}
	return code
}

func FieldCodeOf(prompt string) string {
	return codeFromText(prompt, fieldCodeFallback)
}

func uniqueFieldCode(proposed string, taken map[string]bool) string {
	code := proposed
	for n := 2; taken[code]; n++ {
		code = proposed + "_" + strconv.Itoa(n)
	}
	return code
}

func validFieldCode(code string) bool {
	return fieldCodePattern.MatchString(code) && !fieldKeyPattern.MatchString(code)
}

func ProposeFieldCodes(prompts []string) []string {
	return proposeFieldCodesAvoiding(prompts, map[string]bool{})
}

func proposeFieldCodesAvoiding(prompts []string, taken map[string]bool) []string {
	codes := make([]string, len(prompts))
	for i, prompt := range prompts {
		codes[i] = uniqueFieldCode(FieldCodeOf(prompt), taken)
		taken[codes[i]] = true
	}
	return codes
}

type fieldSettlement struct {
	recordCode string
	before     []models.StepQuestion
	known      bool
	makeKey    fieldKeyMaker
}

func settleFields(s fieldSettlement, questions []models.StepQuestion) ([]models.StepQuestion, error) {
	out := append([]models.StepQuestion(nil), questions...)
	if err := s.settleKeys(out); err != nil {
		return nil, err
	}
	if err := s.settleCodes(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s fieldSettlement) settleKeys(out []models.StepQuestion) error {
	held := map[string]bool{}
	for _, q := range s.before {
		held[q.Key] = true
	}
	used := map[string]bool{}
	for i := range out {
		key := out[i].Key
		if key == "" {
			continue
		}
		switch {
		case strings.ContainsAny(key, " \t\r\n,;") || len(key) > 64:
			return invalidRecord("the field %q has a key that is not one word of at most 64 characters", out[i].Prompt)
		case s.known && !held[key]:
			return invalidRecord("the field %q names the key %q, which the record never had", out[i].Prompt, key)
		case used[key]:
			return invalidRecord("two fields of the record share the key %q", key)
		}
		used[key] = true
	}
	claimed := map[string]bool{}
	for i := range out {
		if out[i].Key != "" {
			claimed[out[i].Key] = true
		}
	}
	for i := range out {
		if out[i].Key != "" {
			continue
		}
		previous, found := s.previousOf(out[i], claimed)
		if found {
			out[i].Key = previous.Key
			if out[i].Code == "" {
				out[i].Code = previous.Code
			}
			claimed[previous.Key] = true
			continue
		}
		out[i].Key = s.freshKey(out, i, used)
		used[out[i].Key] = true
	}
	return nil
}

func (s fieldSettlement) previousOf(q models.StepQuestion, claimed map[string]bool) (models.StepQuestion, bool) {
	if q.Code != "" {
		for _, p := range s.before {
			if p.Key != "" && !claimed[p.Key] && p.Code == q.Code {
				return p, true
			}
		}
	}
	for _, p := range s.before {
		if p.Key != "" && !claimed[p.Key] && p.Prompt == q.Prompt {
			return p, true
		}
	}
	return models.StepQuestion{}, false
}

func (s fieldSettlement) freshKey(out []models.StepQuestion, i int, used map[string]bool) string {
	sameBefore := 0
	for _, earlier := range out[:i] {
		if earlier.Prompt == out[i].Prompt {
			sameBefore++
		}
	}
	for {
		key := s.makeKey(s.recordCode, out[i], sameBefore)
		if !used[key] {
			return key
		}
		sameBefore++
	}
}

func (s fieldSettlement) settleCodes(out []models.StepQuestion) error {
	taken := map[string]bool{}
	for _, q := range out {
		if q.Code == "" {
			continue
		}
		switch {
		case !validFieldCode(q.Code):
			return invalidRecord("the field %q has the code %q; a code is lower case letters, digits and underscores, starting with a letter", q.Prompt, q.Code)
		case taken[q.Code]:
			return invalidRecord("two fields of the record share the code %q", q.Code)
		}
		taken[q.Code] = true
	}
	var unnamed []int
	var prompts []string
	for i := range out {
		if out[i].Code == "" {
			unnamed = append(unnamed, i)
			prompts = append(prompts, out[i].Prompt)
		}
	}
	for n, code := range proposeFieldCodesAvoiding(prompts, taken) {
		out[unnamed[n]].Code = code
	}
	return nil
}

func keyedFieldsOrRefused(recordCode string, questions []models.StepQuestion) ([]models.StepQuestion, error) {
	return settleFields(fieldSettlement{recordCode: recordCode, makeKey: legacyFieldKey}, questions)
}

func keyedFields(recordCode string, questions []models.StepQuestion) []models.StepQuestion {
	settled, err := keyedFieldsOrRefused(recordCode, questions)
	if err != nil {
		return questions
	}
	return settled
}

func withoutFieldIdentity(questions []models.StepQuestion) []models.StepQuestion {
	out := append([]models.StepQuestion(nil), questions...)
	for i := range out {
		out[i].Key, out[i].Code = "", ""
	}
	return out
}

func withoutDefaultFieldIdentity(questions []models.StepQuestion) []models.StepQuestion {
	out := withoutFieldIdentity(questions)
	prompts := make([]string, len(questions))
	for i, q := range questions {
		prompts[i] = q.Prompt
	}
	proposed := ProposeFieldCodes(prompts)
	for i := range out {
		if questions[i].Code != proposed[i] {
			out[i].Code = questions[i].Code
		}
	}
	return out
}

type recordFields struct {
	byKey    map[string]models.StepQuestion
	byCode   map[string]models.StepQuestion
	byPrompt map[string]models.StepQuestion
}

func recordFieldsOf(r models.Record) recordFields {
	f := recordFields{
		byKey: map[string]models.StepQuestion{}, byCode: map[string]models.StepQuestion{}, byPrompt: map[string]models.StepQuestion{},
	}
	for _, q := range keyedFields(r.Identity(), r.Questions) {
		f.byKey[q.Key] = q
		f.byCode[q.Code] = q
		if _, seen := f.byPrompt[q.Prompt]; !seen {
			f.byPrompt[q.Prompt] = q
		}
	}
	return f
}

func (f recordFields) named(name string) (models.StepQuestion, bool) {
	if q, ok := f.byKey[name]; ok {
		return q, true
	}
	if q, ok := f.byCode[name]; ok {
		return q, true
	}
	q, ok := f.byPrompt[name]
	return q, ok
}

func (f recordFields) keyed(named map[string]string) map[string]string {
	if len(named) == 0 {
		return named
	}
	out := make(map[string]string, len(named))
	for name, value := range named {
		if q, ok := f.named(name); ok {
			out[q.Key] = value
			continue
		}
		out[name] = value
	}
	return out
}
