package anthropic

// Tool represents a function that Claude can invoke during the conversation
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema InputSchema `json:"input_schema"`
}

// InputSchema defines the JSON Schema for tool parameters
type InputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required"`
}

// Property defines a single parameter in the tool's input schema
type Property struct {
	Type        string              `json:"type"`
	Description string              `json:"description"`
	Items       *Items              `json:"items,omitempty"`
	Properties  map[string]Property `json:"properties,omitempty"`
}

// Items defines the type of elements in an array property
type Items struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties,omitempty"`
	Required   []string            `json:"required,omitempty"`
}

// GetGimpyTools returns the predefined tools for Gimpy's AI agent
func GetGimpyTools() []Tool {
	return []Tool{
		{
			Name: "update_user_status",
			Description: `Updates the current physical status of the user before generating a workout routine. 
This tool MUST be called when the user reports any of the following:
- Current energy level (scale 1-10, where 1 = No energy/Completely exhausted and 10 = Maximum energy/Fully rested)
- New injuries or pain (muscle strains, joint pain, previous injuries that are still healing)
- Available training time for the session (in minutes)
- Recent illness or recovery status
Use this tool to gather critical safety information that will inform workout intensity and exercise selection.`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"user_id": {
						Type:        "string",
						Description: "The unique identifier of the user",
					},
					"energy_level": {
						Type:        "integer",
						Description: "Current energy level exactly as reported by the user on a scale of 1-10, where 1 = No energy/Completely exhausted and 10 = Maximum energy/Fully rested. Higher means MORE energy. Do NOT invert or convert the user's number.",
					},
					"reported_injuries": {
						Type:        "array",
						Description: "List of current injuries or pain points. CRITICAL RULE: If the user states they are 'cured', 'recovered', 'fine', or explicitly says they have no pain to train a specific muscle group, you MUST send an empty array [] to clear their medical record, ignoring past injuries in the conversation history.",
						Items: &Items{
							Type: "string",
						},
					},
					"available_time_minutes": {
						Type:        "integer",
						Description: "Available time for the workout session in minutes",
					},
					"additional_notes": {
						Type:        "string",
						Description: "Any additional context about the user's current state (illness, stress, sleep quality, etc.)",
					},
				},
				Required: []string{"user_id", "energy_level"},
			},
		},
		{
			Name: "get_exercise_history",
			Description: `Retrieves the user's exercise history to inform progressive overload and exercise selection.
This tool should be called to:
- Check what exercises the user has performed recently
- Verify previous weights, sets, and reps to ensure proper progression
- Identify exercise patterns and preferences
- Avoid repeating the same exercises too frequently
- Ensure balanced muscle group training across the week
The history helps maintain scientific training principles like progressive overload, adequate recovery, and periodization.

This tool CAN and SHOULD be used standalone in Advisor Mode if the user simply asks to remember their past workouts.`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"user_id": {
						Type:        "string",
						Description: "The unique identifier of the user whose history to retrieve",
					},
					"days_back": {
						Type:        "integer",
						Description: "Number of days to look back in history (default: 7 days for weekly overview)",
					},
					"muscle_group": {
						Type:        "string",
						Description: "Optional filter for specific muscle group (e.g., 'chest', 'legs', 'back', 'shoulders', 'arms')",
					},
				},
				Required: []string{"user_id"},
			},
		},
		{
			Name: "generate_workout_json",
			Description: `Generates the final structured workout routine in JSON format. This is the FINAL tool that must be called after gathering all necessary context.
This tool should ONLY be called after:
1. Checking the user's current status (fatigue, injuries) via update_user_status
2. Reviewing exercise history via get_exercise_history
3. Analyzing the user's fitness goals and any contraindications

The workout must follow evidence-based training principles:
- Progressive overload: Gradually increase weight, reps, or sets from previous sessions
- Exercise selection: Choose exercises appropriate for the user's goals, experience level, and any injuries
- Volume and intensity: Adjust based on fatigue level and recovery status
- Exercise order: Compound movements first, isolation exercises later
- Rest periods: Appropriate for the training goal (strength: 2-5min, hypertrophy: 1-2min, endurance: 30-60s)
- Safety: Avoid exercises that could aggravate reported injuries

The generated routine must be scientifically sound and personalized to the user's current state.`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"user_id": {
						Type:        "string",
						Description: "The unique identifier of the user this workout is for",
					},
					"exercises": {
						Type:        "array",
						Description: "List of exercises in the workout, ordered from most to least demanding",
						Items: &Items{
							Type: "object",
							Properties: map[string]Property{
								"exercise_id": {
									Type:        "string",
									Description: "Technical exercise code in English (e.g., 'barbell_back_squat', 'dumbbell_bench_press')",
								},
								"display_name": {
									Type:        "string",
									Description: "The name of the exercise to display in the UI. This MUST be in Spanish and match the 'name_es' of the catalog (e.g., 'Press de Banca').",
								},
								"sets": {
									Type:        "integer",
									Description: "Number of sets to perform",
								},
								"reps": {
									Type:        "integer",
									Description: "Number of repetitions per set",
								},
								"weight": {
									Type:        "number",
									Description: "The weight to be used for the exercise in Kilograms (kg). If there is no previous data for this exercise in the user's history, you MUST set this strictly to 0.",
								},
							},
							Required: []string{"exercise_id", "display_name", "sets", "reps", "weight"},
						},
					},
				},
				Required: []string{"user_id", "exercises"},
			},
		},
		{
			Name:        "log_workout_feedback",
			Description: "Logs the user's feedback after they complete a workout. Use this ONLY when the user tells you how their recent workout went, how tired they are after it, or gives specific notes about exercises.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"fatigue_level": {
						Type:        "integer",
						Description: "The perceived post-workout fatigue from 1 to 10, where 1 = Not tired at all and 10 = Completely exhausted. This is NOT the pre-workout energy scale.",
					},
					"notes": {
						Type:        "string",
						Description: "A brief summary of the user's feedback, performance, or struggles during the workout. Write it in Spanish.",
					},
				},
				Required: []string{"fatigue_level", "notes"},
			},
		},
		{
			Name: "get_allowed_exercises",
			Description: `Retrieves the catalog of allowed exercises for a specific muscle group. This tool MUST be called before 'generate_workout_json' in Generator Mode.
Every exercise used in the generated workout MUST come from this catalog - you are STRICTLY FORBIDDEN from inventing exercise IDs or names.
Use the 'id' from the catalog as the 'exercise_id', and the 'name_es' as the 'display_name'.`,
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"muscle_group": {
						Type:        "string",
						Description: "The muscle group to target. Must be exactly one of: 'chest', 'back', 'legs', 'shoulders', 'arms', 'core'. For 'abdomen', 'abs', 'abdominales' or 'core' requests, ALWAYS pass exactly 'core' (never 'abdomen'). If the user requests a full body routine or multiple muscle groups, strictly pass the exact string 'full_body' to retrieve the entire catalog.",
					},
				},
				Required: []string{"muscle_group"},
			},
		},
		{
			Name:        "search_exercise_video",
			Description: "Searches YouTube for an exercise tutorial. The query must be written entirely in Spanish (no English words, no Spanglish).",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"query": {
						Type:        "string",
						Description: "The exact search phrase to send to YouTube. You must include the exercise name and helpful keywords like 'tutorial' or 'técnica', in Spanish only. For example, 'press militar técnica tutorial' or 'curl de bíceps técnica correcta tutorial'.",
					},
				},
				Required: []string{"query"},
			},
		},
	}
}
