package anthropic

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestClient() *AnthropicClient {
	return &AnthropicClient{location: time.UTC}
}

func TestParseWorkoutFromToolInput_ValidPayload(t *testing.T) {
	input := json.RawMessage(`{"exercises":[{"exercise_id":"barbell_row","display_name":"Remo con Barra","sets":4,"reps":8,"weight":0}]}`)

	workout, err := newTestClient().parseWorkoutFromToolInput(input, "user-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(workout.Exercises) != 1 || workout.Exercises[0].Weight != 0 {
		t.Fatalf("unexpected exercises: %+v", workout.Exercises)
	}
}

func TestParseWorkoutFromToolInput_NegativeValuesAreRecoverable(t *testing.T) {
	input := json.RawMessage(`{"exercises":[{"exercise_id":"barbell_row","display_name":"Remo","sets":-1,"reps":-8,"weight":-5.5}]}`)

	_, err := newTestClient().parseWorkoutFromToolInput(input, "user-1")

	var validationErr *workoutValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected *workoutValidationError, got %v", err)
	}
	for _, want := range []string{"sets cannot be negative", "reps cannot be negative", "weight cannot be negative"} {
		if !strings.Contains(validationErr.Error(), want) {
			t.Errorf("expected message to contain %q, got %q", want, validationErr.Error())
		}
	}
}

func TestParseWorkoutFromToolInput_MalformedJSONIsNotRecoverable(t *testing.T) {
	_, err := newTestClient().parseWorkoutFromToolInput(json.RawMessage(`{not json`), "user-1")

	var validationErr *workoutValidationError
	if err == nil || errors.As(err, &validationErr) {
		t.Fatalf("expected a plain unmarshal error, got %v", err)
	}
}

func TestUnsupportedLanguageMessage_IsSpanishOnly(t *testing.T) {
	if !strings.Contains(unsupportedLanguageMessage, "español") {
		t.Errorf("expected message to state that only Spanish is supported, got %q", unsupportedLanguageMessage)
	}
	for _, english := range []string{"I'm sorry", "Spanish", "English", "inglés"} {
		if strings.Contains(unsupportedLanguageMessage, english) {
			t.Errorf("message must be Spanish only, found %q in %q", english, unsupportedLanguageMessage)
		}
	}
}

func TestSystemPrompt_SpanishOnlyAndNutritionScope(t *testing.T) {
	for _, want := range []string{
		"NUTRITION SCOPE RULE",
		"You support ONLY Spanish",
		`Respond EXACTLY and ONLY with: "UNSUPPORTED_LANGUAGE"`,
		"match the 'name_es'",
	} {
		if !strings.Contains(systemPrompt, want) {
			t.Errorf("expected system prompt to contain %q", want)
		}
	}
	for _, gone := range []string{"name_en", "TWO languages", "Spanish or English"} {
		if strings.Contains(systemPrompt, gone) {
			t.Errorf("system prompt must not contain %q anymore", gone)
		}
	}
}
