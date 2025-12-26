package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/andrejsstepanovs/go-litellm/models"
	"github.com/andrejsstepanovs/go-litellm/request"
	"github.com/andrejsstepanovs/storygen/pkg/story"
)

const ChapterPromptInstructions = "# Content writing instructions:\n" +
	"- Analyze previous chapters (if exists) before writing the next one.\n" +
	"- If story is for children then use shorter sentences, simple language and avoid complex words.\n" +
	"- If story is for children then write with respect for young readers. Include proper story development, meaningful plot progression, and clever twists.\n" +
	"- Avoid talking down or using overly childish language.\n" +
	"- Dont be cringe, skip overly childish and safe content.\n" +
	"- Avoid sugar-coating and predictable storylines.\n" +
	"- Proceed the storyline in a way that fits the chapter's place in the story.\n" +
	"- Use all provided story details (characters, setting, plot, morals, etc.) to create a rich, imaginative, and engaging chapter.\n" +
	"- Ensure the chapter aligns with the story's structure, timeline, themes, protagonist, villain, and overall plan.\n" +
	"- Take into consideration Story Suggestion.\n" +
	"- Write it using funny interactions between characters.\n" +
	"- Move plot forward without diving into surrounding details.\n" +
	"- Use Time-Related Transitions:\n" +
	"-- Instead of 'and then,' try:\n" +
	"-- After that\n" +
	"-- Meanwhile\n" +
	"-- Later\n" +
	"-- Shortly afterward\n" +
	"-- Moments later\n" +
	"-- Subsequently\n" +
	"-- In the meantime\n" +
	"- Use Cause-and-Effect Connections:\n" +
	"-- As a result\n" +
	"-- Consequently\n" +
	"-- Therefore\n" +
	"-- This led to\n" +
	"-- Because of this\n" +
	"- Replace with Action Verbs:\n" +
	"-- Instead of: 'She opened the door and then walked inside'\n" +
	"-- Try: 'She opened the door, stepping cautiously inside'\n" +
	"- Use Subordinate Clauses:\n" +
	"-- Instead of: 'He finished his homework and then he went to play'\n" +
	"-- Try: 'After finishing his homework, he went to play'\n" +
	"- Introduce Simultaneous Actions\n" +
	"-- Instead of: 'She heard the noise and then she turned around'\n" +
	"-- Try: 'Hearing the noise, she turned around'\n" +
	"- Connect Settings to Characters: make locations matter to your characters\n" +
	"- Be Specific About Location, Time and Weather\n" +
	"- Use minimal amount of adjectives.\n" +
	"- Restrain yourself from using cliché things like 'Whispering Woods', 'misty meadow', etc.\n" +
	"- Always place speaker name before quoting what they say. \n" +
	"-- Example: Max said \"That's amazing!\" NOT \"That's amazing!\" Max said.\n" +
	"-- Example: Johnny insisted \"I don't believe you\" NOT \"I don't believe you,\" Johnny insisted.\n" +
	"- Tell what happened and what happened next moving plot forward.\n\n" +
	"# Writing style Adjustments:\n" +
	"You often use descriptive phrases or clauses to extend sentences. " +
	"While they add great imagery, they can feel repetitive if overused. " +
	"Try mixing it up with shorter, punchier sentences or different ways of describing actions and settings! " +
	"It'll help keep the pacing fresh and engaging!" +
	"Another thing - laughing and dancing is nice but too much is cringe."

const ForceJson = "No yapping. Answer **only with raw JSON**. Dont wrap json with tags or quotes or anything else. Answer only with RAW JSON."

const GeneralInstruction = ""

func (a *AI) generate(systemPrompt, userPrompt string, useJSON bool) (string, error) {
	model, err := a.client.Model(a.ctx, models.ModelID(a.model))
	if err != nil {
		return "", fmt.Errorf("failed to get model: %w", err)
	}

	messages := request.Messages{
		request.SystemMessageSimple(systemPrompt),
		request.UserMessageSimple(userPrompt),
	}

	req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
	if useJSON {
		req.SetJSONMode()
	}

	resp, err := a.client.Completion(a.ctx, req)
	if err != nil {
		return "", fmt.Errorf("completion failed: %w", err)
	}

	return resp.String(), nil
}

func (a *AI) SuggestStoryFixes(storyEl story.Story, problem story.Problem, addressedSuggestions story.Suggestions) story.Suggestions {
	problemInjsonTxt := ""
	for i := 0; i < 10; i++ {
		suggestions, query, err := a.trySuggestStoryFixes(storyEl, problem, addressedSuggestions, problemInjsonTxt)
		if err == nil {
			return suggestions
		}
		log.Printf("Failed to suggest story fixes for chapter %d (attempt %d/10): %v", problem.Chapter, i+1, err)
		if query != "" {
			log.Printf("AI Response was: %s", query)
		}

		// Provide specific feedback based on the error
		if strings.Contains(err.Error(), "cannot unmarshal object") {
			problemInjsonTxt = fmt.Sprintf("\n\n**ERROR: You returned a single object instead of an array!**\n"+
				"You MUST return an array starting with [ and ending with ], even if there's only one suggestion.\n"+
				"Correct format: [{\"chapter_number_int\": 1, \"chapter_name\": \"Title\", \"suggestions_array_string\": [\"Fix this\"]}]\n"+
				"Your previous incorrect response: %s", query)
		} else if strings.Contains(query, "```") {
			problemInjsonTxt = fmt.Sprintf("\n\n**ERROR: You wrapped the JSON in markdown code blocks!**\n"+
				"Do NOT use ```json or ``` markers. Return ONLY the raw JSON array.\n"+
				"Your previous incorrect response: %s", query)
		} else {
			problemInjsonTxt = fmt.Sprintf("\n\n**ERROR: Invalid JSON format: %v**\n"+
				"Return a valid JSON array: [{\"chapter_number_int\": 1, \"chapter_name\": \"Title\", \"suggestions_array_string\": [\"Fix\"]}]\n"+
				"Your previous response: %s", err, query)
		}
	}

	log.Fatalf("Failed to suggest story fixes for problem chapter %d after 10 attempts", problem.Chapter)
	return story.Suggestions{}
}

func (a *AI) trySuggestStoryFixes(storyEl story.Story, problem story.Problem, addressedSuggestions story.Suggestions, problemInjsonTxt string) (story.Suggestions, string, error) {
	if problem.Chapter < len(storyEl.Chapters) {
		storyEl.Chapters = storyEl.Chapters[:problem.Chapter]
	}

	systemPrompt := "You are a story editor suggesting fixes for story chapters to resolve issues. Your suggestions will be used to re-write chapters later. Return ONLY raw JSON without any markdown formatting or code blocks."

	userPrompt := fmt.Sprintf("Analyze chapter %d (%s) of this %s story and suggest fixes for the following issues:\n\n"+
		"**Issues to fix:**\n%s\n\n"+
		"**Story context (chapters 1-%d):**\n```json\n%s\n```\n\n"+
		"**Already addressed suggestions (ignore these):**\n```json\n%s\n```\n\n"+
		"**Instructions:**\n"+
		"1. Suggest specific, actionable changes to fix the issues\n"+
		"2. Identify which chapter(s) need changes (current or earlier chapters)\n"+
		"3. Keep suggestions practical - minimal text changes preferred\n"+
		"4. Focus on major plot holes and inconsistencies, not minor details\n"+
		"5. Maximum 5 suggestions total (or return empty array if no fixes needed)\n"+
		"6. Do not suggest creating new chapters\n"+
		"7. Maintain the existing %s story writing style\n\n"+
		"**CRITICAL: Response format requirements:**\n"+
		"- Return ONLY a JSON array, nothing else\n"+
		"- Do NOT wrap the JSON in markdown code blocks (no ```json or ```)\n"+
		"- Do NOT add any explanatory text before or after the JSON\n"+
		"- Start your response with [ and end with ]\n"+
		"- Each object in the array must have: chapter_number_int (integer), chapter_name (string), suggestions_array_string (array of strings)\n"+
		"- Return empty array [] if no important fixes are needed\n\n"+
		"Example valid response: [{\"chapter_number_int\": 1, \"chapter_name\": \"Title\", \"suggestions_array_string\": [\"Fix X\", \"Change Y\"]}]%s",
		problem.Chapter, problem.ChapterName, a.audience,
		problem.ToJson(),
		problem.Chapter,
		storyEl.ToJson(),
		addressedSuggestions.ToJson(),
		a.audience,
		problemInjsonTxt)

	// Create JSON schema for structured output
	schema := request.JSONSchema{
		Name: "story_suggestions",
		Schema: map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chapter_number_int": map[string]interface{}{
						"type":        "integer",
						"description": "The chapter number that needs adjustment",
					},
					"chapter_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the chapter",
					},
					"suggestions_array_string": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Array of suggestions for fixing the chapter",
					},
				},
				"required":             []string{"chapter_number_int", "chapter_name", "suggestions_array_string"},
				"additionalProperties": false,
			},
		},
		Strict: true,
	}

	model, err := a.client.Model(a.ctx, models.ModelID(a.model))
	if err != nil {
		return story.Suggestions{}, "", fmt.Errorf("failed to get model: %w", err)
	}

	messages := request.Messages{
		request.SystemMessageSimple(systemPrompt),
		request.UserMessageSimple(userPrompt),
	}

	req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
	req.SetJSONSchema(schema)

	resp, err := a.client.Completion(a.ctx, req)
	if err != nil {
		return story.Suggestions{}, "", fmt.Errorf("completion failed: %w", err)
	}

	respStr := resp.String()

	// Check if response is empty
	if len(respStr) == 0 {
		return story.Suggestions{}, "", fmt.Errorf("received empty response from AI")
	}

	// Clean the response - remove markdown code blocks if present
	respStr = cleanResponse(respStr)

	// Extract just the JSON array - find the first [ and last ]
	startIdx := strings.Index(respStr, "[")
	if startIdx == -1 {
		return story.Suggestions{}, respStr, fmt.Errorf("no JSON array found in response (missing '[')")
	}

	// Find the matching closing bracket
	endIdx := strings.LastIndex(respStr, "]")
	if endIdx == -1 || endIdx < startIdx {
		return story.Suggestions{}, respStr, fmt.Errorf("no valid JSON array found in response (missing ']')")
	}

	// Extract just the JSON array portion
	jsonStr := respStr[startIdx : endIdx+1]

	// Log the extracted JSON for debugging
	if len(jsonStr) > 500 {
		log.Printf("Extracted JSON (truncated): %s...", jsonStr[:500])
	} else {
		log.Printf("Extracted JSON: %s", jsonStr)
	}

	var picked story.Suggestions
	err = json.Unmarshal([]byte(jsonStr), &picked)
	if err != nil {
		return story.Suggestions{}, respStr, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	log.Printf("Successfully parsed %d suggestions", len(picked))

	return picked, "", nil
}

func (a *AI) AdjustStoryChapter(storyEl story.Story, problem story.Problem, suggestions story.Suggestions, addressedSuggestions story.Suggestions, wordCount int) string {
	if problem.Chapter < len(storyEl.Chapters) {
		storyEl.Chapters = storyEl.Chapters[:problem.Chapter]
	}

	systemPrompt := "You are story writer that is fixing story issues before it goes to publishing."

	userPrompt := fmt.Sprintf("Re-write the %s Story chapter %d %s. "+
		"Analyze full %s Story and adjust the problematic chapter %d %s.\n"+
		"Here are all already addressed suggestions: \n<already_addressed_suggestions>\n%s\n</already_addressed_suggestions>\n"+
		"**IMPORTANT**: Suggestions how to fix the issues at hand: \n<fix_suggestions>\n%s\n</fix_suggestions>\n"+
		"Use and rely only on these suggestions provided!\n"+
		"For reference, here is full story until this chapter ```json\n%s\n```. "+
		"# Orders:"+
		"- There are maybe more chapters but lets focus on story until this moment.\n"+
		"- Fix only this chapter so story is coherent, entertaining and makes sense (use given suggestions). "+
		"- Use suggestions from fix_suggestions tag to re-write the story chapter %d %s as suggested. "+
		"- Make sure you don't break out of suggestions that were fixed before (see json in: already_addressed_suggestions tags). "+
		"- Answer with only one chapter text. We are fixing it one chapter at the time. "+
		"- Be creative to fix the issue at hand. Be swift and decisive. No need for long texts, we just need to fix these issues and move on. "+
		"- Small text extensions are OK, but we should try to keep this chapter withing a limit of %d words. "+
		"%s %s",
		a.audience, problem.Chapter, problem.ChapterName,
		a.audience, problem.Chapter, problem.ChapterName,
		addressedSuggestions.ToJson(),
		suggestions.ToJson(),
		storyEl.ToJson(),
		problem.Chapter, problem.ChapterName,
		wordCount,
		GeneralInstruction, ChapterPromptInstructions)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return removeThinking(templateResponse)
}

func (a *AI) FigureStoryLogicalProblems(storyText string, loop, maxLoops int) story.Problems {
	problemInjsonTxt := ""
	for i := 0; i < 10; i++ {
		problems, query, err := a.findStoryLogicalProblems(storyText, loop, maxLoops, problemInjsonTxt)
		if err == nil {
			return problems
		}
		log.Println("Failed to figure story problems. Trying again.")
		problemInjsonTxt = fmt.Sprintf("Your last answer contained invalid JSON: ----\n\n%s\n\n----. Try again and this time make sure your JSON is valid!", query)
	}

	log.Fatalln("Failed to figure story problems")
	return story.Problems{}
}

func (a *AI) findStoryLogicalProblems(storyText string, loop, maxLoops int, promptExend string) (story.Problems, string, error) {
	problems := story.Problems{
		{
			Chapter:     1,
			ChapterName: "The Beginning",
			Issues: []string{
				"Doctor could not know about the name of a cat because no one told him yet",
				"Girls leg was broken, she could not hop her way trough the forest, its close to impossible feat",
			},
		},
		{
			Chapter:     3,
			ChapterName: "Home sweet home",
			Issues: []string{
				"Story ending do not make sense, they didnt came back home so it is not end of the journey",
				"On first chapter book had brown color and now its black",
				"This chapter is just too boring to read. Need more action and twists.",
			},
		},
	}

	systemPrompt := "You are helping to pre-read a story and your output will help us to fix the story flaws."

	userPrompt := fmt.Sprintf("Create a JSON problem list for %s story we need to check (pre-read):\n"+
		"<story_text>\n%s\n</story_text>\n\n"+
		"Find problems and flaws in the plot and answer with formatted output as mentioned in examples.\n"+
		"Carefully read the story text chapter by chapter and analyze it for logical flaws in the story in each chapter.\n"+
		"This is cycle %d of pre-reading. Reduce strictness and issue count proportionally to the number of cycles completed. Max cycles: %d.\n\n"+
		"%s %s\n"+
		"If no flaws are found, do not include the chapter in your output. "+
		"Example format: %s\n"+
		"%s",
		a.audience,
		storyText,
		loop, maxLoops,
		GeneralInstruction, ForceJson,
		problems.ToJson(),
		promptExend)

	// Create JSON schema for structured output
	schema := request.JSONSchema{
		Name: "story_problems",
		Schema: map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"chapter_number_int": map[string]interface{}{
						"type":        "integer",
						"description": "The chapter number with issues",
					},
					"chapter_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the chapter",
					},
					"issues_array_string": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
						"description": "Array of issues found in the chapter",
					},
				},
				"required":             []string{"chapter_number_int", "chapter_name", "issues_array_string"},
				"additionalProperties": false,
			},
		},
		Strict: true,
	}

	model, err := a.client.Model(a.ctx, models.ModelID(a.model))
	if err != nil {
		return story.Problems{}, "", fmt.Errorf("failed to get model: %w", err)
	}

	messages := request.Messages{
		request.SystemMessageSimple(systemPrompt),
		request.UserMessageSimple(userPrompt),
	}

	req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
	req.SetJSONSchema(schema)

	resp, err := a.client.Completion(a.ctx, req)
	if err != nil {
		return story.Problems{}, "", fmt.Errorf("completion failed: %w", err)
	}

	respStr := resp.String()

	// Clean the response
	respStr = cleanResponse(respStr)

	// Extract just the JSON array - find the first [ and last ]
	startIdx := strings.Index(respStr, "[")
	if startIdx == -1 {
		return story.Problems{}, respStr, fmt.Errorf("no JSON array found in response (missing '[')")
	}

	endIdx := strings.LastIndex(respStr, "]")
	if endIdx == -1 || endIdx < startIdx {
		return story.Problems{}, respStr, fmt.Errorf("no valid JSON array found in response (missing ']')")
	}

	jsonStr := respStr[startIdx : endIdx+1]

	var picked story.Problems
	err = json.Unmarshal([]byte(jsonStr), &picked)
	if err != nil {
		return story.Problems{}, respStr, fmt.Errorf("failed to parse JSON: %w", err)
	}

	ret := make(story.Problems, 0)
	for _, pr := range picked {
		if len(pr.Issues) > 0 {
			ret = append(ret, pr)
		}
	}

	return ret, "", nil
}

func (a *AI) FigureStoryProtagonists(storyEl story.Story) story.Protagonists {
	examples := func(count int) string {
		p := story.GetRandomProtagonists(count)
		return p.ToJson()
	}

	systemPrompt := "You are helping to prepare a story ideas that will be used later on."

	userPrompt := fmt.Sprintf("Create a JSON protagonists list that will fit the %s story we will write. Story:\n```json\n%s\n```\n\n"+
		"Be mindful about how many you are picking. "+
		"It is totally OK to pick single or multiple same types of protagonists as they're personas will be extended later on with more details."+
		"Your task now is to pick from the list.\n"+
		"Pick good simple but memorable protagonist names.\n"+
		"Be creative with your picks. We want a vibrant, exciting story and protagonists are/is important and needs to be suitable and interesting."+
		"Don't specify protagonists sexual orientations, that type of info is mostly irrelevant in %s stories.\n"+
		"%s %s\n"+
		"Example format: %s",
		a.audience,
		storyEl.ToJson(),
		a.audience,
		GeneralInstruction, ForceJson,
		examples(5))

	// Create JSON schema for structured output
	schema := request.JSONSchema{
		Name: "protagonists_list",
		Schema: map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The protagonist's name",
					},
					"voice": map[string]interface{}{
						"type":        "string",
						"description": "Description of the protagonist's voice",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Type of protagonist (human, animal, mythical being, etc.)",
					},
					"gender": map[string]interface{}{
						"type":        "string",
						"description": "Gender of the protagonist",
					},
					"size": map[string]interface{}{
						"type":        "string",
						"description": "Size of the protagonist (small, normal, large)",
					},
					"age": map[string]interface{}{
						"type":        "string",
						"description": "Age category of the protagonist",
					},
				},
				"required":             []string{"name", "voice", "type", "gender", "size", "age"},
				"additionalProperties": false,
			},
		},
		Strict: true,
	}

	model, err := a.client.Model(a.ctx, models.ModelID(a.model))
	if err != nil {
		log.Fatalf("Failed to get model: %v", err)
	}

	messages := request.Messages{
		request.SystemMessageSimple(systemPrompt),
		request.UserMessageSimple(userPrompt),
	}

	req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
	req.SetJSONSchema(schema)

	resp, err := a.client.Completion(a.ctx, req)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	respStr := resp.String()
	respStr = cleanResponse(respStr)

	// Extract JSON array
	startIdx := strings.Index(respStr, "[")
	if startIdx != -1 {
		endIdx := strings.LastIndex(respStr, "]")
		if endIdx != -1 && endIdx > startIdx {
			respStr = respStr[startIdx : endIdx+1]
		}
	}

	var picked story.Protagonists
	err = json.Unmarshal([]byte(respStr), &picked)
	if err != nil {
		log.Fatalf("Failed to parse protagonists as JSON %s: %v", respStr, err)
	}

	return picked
}

func (a *AI) FigureStoryMorales(storyEl story.Story) story.Morales {
	morales := story.GetAvailableStoryMorales()
	moraleExample := func(count int) string {
		moraleExamples := story.GetRandomMorales(count, morales)
		moraleNames := make([]string, 0)
		for _, m := range moraleExamples {
			moraleNames = append(moraleNames, m.Name)
		}
		jsonResp, err := json.Marshal(moraleNames)
		if err != nil {
			log.Fatalln(err)
		}
		return string(jsonResp)
	}

	systemPrompt := "You are helping to prepare a story ideas that will be used later on."

	userPrompt := fmt.Sprintf("Create a list of morale names that will fit the %s story we will write. Story:\n```json\n%s\n```\n\n"+
		"Pick morales (`name`) from list of available morales:\n```\njson%s\n```"+
		"Be flexible with your picks. We want creative choices for exciting story.\n"+
		"Do not be afraid to pick something (I noticed you always pick Courage) that is not fitting perfectly. The more the better.\n"+
		"%s %s\n"+
		"No yapping. Answer with a list of morale names as strings (as simple array list with no key(s)) in JSON format.\n"+
		"Example: %s",
		a.audience,
		storyEl.ToJson(),
		morales.ToJson(),
		GeneralInstruction, ForceJson,
		moraleExample(3))

	// Create JSON schema for structured output
	schema := request.JSONSchema{
		Name: "morale_names",
		Schema: map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type":        "string",
				"description": "Name of a morale from the available list",
			},
			"description": "Array of morale names",
		},
		Strict: true,
	}

	model, err := a.client.Model(a.ctx, models.ModelID(a.model))
	if err != nil {
		log.Fatalf("Failed to get model: %v", err)
	}

	messages := request.Messages{
		request.SystemMessageSimple(systemPrompt),
		request.UserMessageSimple(userPrompt),
	}

	req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
	req.SetJSONSchema(schema)

	resp, err := a.client.Completion(a.ctx, req)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	respStr := resp.String()
	respStr = cleanResponse(respStr)

	// Extract JSON array
	startIdx := strings.Index(respStr, "[")
	if startIdx != -1 {
		endIdx := strings.LastIndex(respStr, "]")
		if endIdx != -1 && endIdx > startIdx {
			respStr = respStr[startIdx : endIdx+1]
		}
	}

	var picked []string
	err = json.Unmarshal([]byte(respStr), &picked)
	if err != nil {
		log.Fatalf("Failed to parse JSON for morales response: %v", err)
	}

	return story.FindMoralesByName(picked)
}

func (a *AI) FigureStoryIdeas(count int) []string {
	systemPrompt := "You are helping to prepare a story ideas that will be used later on."

	userPrompt := fmt.Sprintf("Create a list of %d story ideas that will fit the %s\n"+
		"Be creative and funny.\n"+
		"%s %s\n"+
		"No yapping. Answer with a list of story ideas as strings (as simple array list with no key(s)) in JSON format.",
		count,
		a.audience,
		GeneralInstruction, ForceJson)

	// Create JSON schema for structured output
	schema := request.JSONSchema{
		Name: "story_ideas",
		Schema: map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type":        "string",
				"description": "A creative and funny story idea",
			},
			"description": "Array of story ideas",
		},
		Strict: true,
	}

	model, err := a.client.Model(a.ctx, models.ModelID(a.model))
	if err != nil {
		log.Fatalf("Failed to get model: %v", err)
	}

	messages := request.Messages{
		request.SystemMessageSimple(systemPrompt),
		request.UserMessageSimple(userPrompt),
	}

	req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
	req.SetJSONSchema(schema)

	resp, err := a.client.Completion(a.ctx, req)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	respStr := resp.String()
	respStr = cleanResponse(respStr)

	// Extract JSON array
	startIdx := strings.Index(respStr, "[")
	if startIdx != -1 {
		endIdx := strings.LastIndex(respStr, "]")
		if endIdx != -1 && endIdx > startIdx {
			respStr = respStr[startIdx : endIdx+1]
		}
	}

	var picked []string
	err = json.Unmarshal([]byte(respStr), &picked)
	if err != nil {
		log.Fatalf("Failed to parse JSON for story ideas response: %v", err)
	}

	return picked
}

func (a *AI) FigureStoryVillainVoice(storyEl story.Story) string {
	systemPrompt := "You are helping to prepare a story book. Now working on picking story villain voice."

	userPrompt := fmt.Sprintf("Create Villain voice. How it sounds, what are the intricate details of how he/she/them talk."+
		"This is Villain description: %s in a story:\n```json\n%s\n```\n\n"+
		"%s\n"+
		"Short clear description of how the villain(s) talk. No yapping. Don't explain your choice or add any other notes and explenations. Answer only with the villain(s) voice description. Answer with raw text (not json).",
		storyEl.Villain,
		storyEl.ToJson(),
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return removeThinking(templateResponse)
}

func (a *AI) FigureStoryVillain(storyEl story.Story) string {
	systemPrompt := "You are helping to prepare a story book. Villain that you are building (writing) will be used later on when story itself will be written."

	userPrompt := fmt.Sprintf("Create Villain for this %s story:\n```json\n%s\n```\n\n"+
		"Keep it simple and do not build backstory or villain characteristics or motives. "+
		"Take into consideration Story Suggestion. "+
		"That kind of details are irrelevant right now and will actually harm the story building part that will come next, "+
		"so be mindful about it. Just short description about who the villain(s) is/are. "+
		"It is OK to not have a villain if it dont belong to the story we're writing."+
		"I noticed that you often pick wizards that can do magic. "+
		"Try to be more creative (if story suggestion allows it) "+
		"and find a villain that is more down to earth (but still evil, bad, annoying, etc.) "+
		"with his/her own backstory, skills and agenda "+
		"that we can work with in the story.\n"+
		"%s\n"+
		"By the way, villain can also be elements of nature or unmovable objects and that kind of stuff. "+
		"Depends on the story we're building. Be creative if possible. Answer with plain text.\n"+
		"Sort description and name of the villain(s) or nothing. No yapping. Don't explain your choice or add any other notes and explenations. Answer only with the villain(s) description in plain text.",
		a.audience,
		storyEl.ToJson(),
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return removeThinking(templateResponse)
}

func (a *AI) FigureStoryPlan(storyEl story.Story) string {
	systemPrompt := "You are helping to prepare a story book."

	userPrompt := fmt.Sprintf("Create and %s story plan about the story. **This is the Story you need to work with**:\n```json\n%s\n```\n\n"+
		"Follow main ideas that are already prepared for the story. "+
		"Be careful building story plan in a way that existing story you are working with (from json above) fits good. "+
		"Make sure you work with Story structure that was picked. We want our plan to align with picked story structure. "+
		"Keep in mind story length. "+
		"Take into consideration Story Suggestion. "+
		"Same goes for picked story morales. Summary and plan should match picked story morales. "+
		"Story plan should be quite brief and short list of things that will happen in the story with no specifics. Details will be written later on. "+
		"Write the plan in a way that the writer later on will not be much constrained with. We want to keep story plan loose and flexible (no details). "+
		"Be creative and make sure that this %s story is moving forward fast so it is engaging and fun to read. "+
		"Plan a story in a way where there are no boring parts and plot is moving forward fast. "+
		"Don't forget to include ending to the story you're planning so there is satisfying conclusions is built into the story properly. "+
		"Consider adding some plot twists and funny interactions between characters.\n"+
		"%s\n"+
		"Story summary and story plan to help the writer later on when they will write the story. No yapping. Don't explain your choice or add any other notes and explenations.",
		a.audience,
		storyEl.ToJson(),
		a.audience,
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return removeThinking(templateResponse)
}

func (a *AI) CompareStories(storyA, storyB story.Story) story.Story {
	systemPrompt := "You are helping to compare 2 story books."

	userPrompt := fmt.Sprintf("Analyze these 2 %s stories and answer with number which story is better.\n."+
		"**Story Nr. 1**:\n```json\n%s\n```\n\n"+
		"**Story Nr. 2**:\n```json\n%s\n```\n\n"+
		"Compare these 2 stories and answer with number which story is better. This is really important task, be careful. Your answer matters a lot! Best story author will get $ 1000000 cash prize.\n"+
		"Consider story plot, engagement and how fun it would be to read. "+
		"Analyze also story plot logical issues. If one story plot is logically broken (do not make sense), then that is really bad. "+
		"Answer with single word that is a number in INTEGER format. Do not explain why you picked one over the other. If story 1 is better then answer with 1, if story 2 is better then answer with 2. "+
		"%s",
		a.audience,
		storyA.ToJson(),
		storyB.ToJson(),
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	templateResponse = removeThinking(templateResponse)
	log.Println(templateResponse)
	templateResponse = strings.TrimSpace(templateResponse)
	picked, err := strconv.Atoi(templateResponse)
	if err != nil {
		log.Fatalf("Failed to parse story comparison response as number: %v", err)
	}
	if picked != 1 && picked != 2 {
		log.Fatalf("Failed to parse story comparison response as number: %v", picked)
	}

	if picked == 1 {
		return storyA
	}
	return storyB
}

func (a *AI) FigureStoryTimePeriod(storyEl story.Story) story.TimePeriod {
	timePeriodExample := func(count int) string {
		moraleExamples := story.GetRandomTimePeriods(count)
		names := make([]string, 0)
		for _, m := range moraleExamples {
			names = append(names, m.Name)
		}
		jsonResp, err := json.Marshal(names)
		if err != nil {
			log.Fatalln(err)
		}
		return string(jsonResp)
	}

	systemPrompt := "You are helping to prepare a story ideas that will be used later on."

	allTimePeriods := story.GetAvailableTimePeriods()
	userPrompt := fmt.Sprintf("Create a list of time periods that will fit the %s story we will write. Story:\n```json\n%s\n```\n\n"+
		"Pick time periods (`name`) from list of available time periods:\n```\njson%s\n```"+
		"Be flexible with your picks. We want a vibrant, exciting story and time period is important and needs to be suitable and interesting. "+
		"%s %s\n"+
		"Example: %s",
		a.audience,
		storyEl.ToJson(),
		allTimePeriods.ToJson(),
		GeneralInstruction, ForceJson,
		timePeriodExample(3))

	// Create JSON schema for structured output
	schema := request.JSONSchema{
		Name: "time_period_names",
		Schema: map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type":        "string",
				"description": "Name of a time period from the available list",
			},
			"description": "Array of time period names",
		},
		Strict: true,
	}

	model, err := a.client.Model(a.ctx, models.ModelID(a.model))
	if err != nil {
		log.Fatalf("Failed to get model: %v", err)
	}

	messages := request.Messages{
		request.SystemMessageSimple(systemPrompt),
		request.UserMessageSimple(userPrompt),
	}

	req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
	req.SetJSONSchema(schema)

	resp, err := a.client.Completion(a.ctx, req)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	respStr := resp.String()
	respStr = cleanResponse(respStr)

	// Extract JSON array
	startIdx := strings.Index(respStr, "[")
	if startIdx != -1 {
		endIdx := strings.LastIndex(respStr, "]")
		if endIdx != -1 && endIdx > startIdx {
			respStr = respStr[startIdx : endIdx+1]
		}
	}

	var picked []string
	err = json.Unmarshal([]byte(respStr), &picked)
	if err != nil {
		log.Fatalf("Failed to parse time period response as JSON: %v", err)
	}

	return story.FindTimePeriodsByName(picked)[0]
}

func (a *AI) FigureStoryChapterTitles(storyEl story.Story, chapterCount int) ([]string, error) {
	systemPrompt := "You are helping to prepare a story content chapter titles."

	userPrompt := fmt.Sprintf("Create a list of story chapter titles that will be used for this %s story:\n```json\n%s\n```\n\n"+
		"Make sure that chapter titles align with existing story details. "+
		"Take into consideration Story Suggestion. Make sure that story have a clear ending. "+
		"Be mindful about the chapter count so it aligns good with story length. Usually there is no need for more than %d chapters. "+
		"Write chapter titles in a way that the plot is naturally moving forward and is aligned with defined %s story structure requirements.\n"+
		"%s %s\n"+
		"Example: ['The Mysterious Map', 'The Magic Paintbrush', 'The Rainbow Bridge', 'The final battle', 'The Return to Home Sweet Home']",
		a.audience,
		storyEl.ToJson(),
		chapterCount,
		a.audience,
		GeneralInstruction, ForceJson+" Make sure your answer starts with [ and list of json array values.")

	// Create JSON schema for structured output
	schema := request.JSONSchema{
		Name: "chapter_titles",
		Schema: map[string]interface{}{
			"type": "array",
			"items": map[string]interface{}{
				"type":        "string",
				"description": "A chapter title",
			},
			"description": "Array of chapter titles",
		},
		Strict: true,
	}

	model, err := a.client.Model(a.ctx, models.ModelID(a.model))
	if err != nil {
		return []string{}, fmt.Errorf("failed to get model: %w", err)
	}

	messages := request.Messages{
		request.SystemMessageSimple(systemPrompt),
		request.UserMessageSimple(userPrompt),
	}

	req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
	req.SetJSONSchema(schema)

	resp, err := a.client.Completion(a.ctx, req)
	if err != nil {
		return []string{}, fmt.Errorf("completion failed: %w", err)
	}

	respStr := resp.String()
	respStr = cleanResponse(respStr)

	// Extract JSON array
	startIdx := strings.Index(respStr, "[")
	if startIdx != -1 {
		endIdx := strings.LastIndex(respStr, "]")
		if endIdx != -1 && endIdx > startIdx {
			respStr = respStr[startIdx : endIdx+1]
		}
	}

	var picked []string
	err = json.Unmarshal([]byte(respStr), &picked)
	if err != nil {
		return []string{}, fmt.Errorf("failed to parse chapter titles as JSON: %w", err)
	}

	return picked, nil
}

func (a *AI) FigureStorySummary(storyEl story.Story) string {
	systemPrompt := "You are summarizing a story book."

	userPrompt := fmt.Sprintf("Create 1 sentence story summary for this story. "+
		"**This is the %s story you need to work with**:\n```json\n%s\n```\n\n"+
		"If exists, take into consideration Story Suggestion.\n"+
		"%s\n"+
		"Answer only with the summary. No yapping. No other explanations, comments, notes or anything else. Answer only with the story summary text (content).",
		a.audience,
		storyEl.ToJson(),
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return removeThinking(templateResponse)
}

func (a *AI) FigureStoryTitle(storyEl story.Story) string {
	systemPrompt := "You are writing a story book title."

	userPrompt := fmt.Sprintf("Write a book name (title) for this %s story. **This is the %s Story you need to work with**:\n```json\n%s\n```\n\n"+
		"Title must be 3-5 words long. Do not explain your choice, no explenation, notes or anything else is necessary. Answer only with 3-5 words!\n"+
		"%s\n"+
		"Examples: 'The Secret Library of Wishes', 'The Brave Little Firefly', 'The girl and the Talking Tree'\n"+
		"Answer only with the short title (3-5 words). Answer only with short story title text.",
		a.audience,
		a.audience,
		storyEl.ToJson(),
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return removeThinking(templateResponse)
}

func (a *AI) FigureStoryChapter(storyEl story.Story, chapterNumber int, chapterTitle string, words int) string {
	isLast := len(storyEl.Chapters) == chapterNumber
	chapterIntent := "to proceed the storyline."
	if isLast {
		chapterIntent = "to finish the story with satisfying ending. " +
			"This is the last chapter of the story so make sure that you end open story topics and end them with good conclusions."
	}

	systemPrompt := "You are writing a story book chapter by chapter. Expand the story with one chapter. You are creative and decisive story writer."

	userPrompt := fmt.Sprintf("Write the single full chapter text, ensuring it flows naturally and keeps the reader engaged. "+
		"**This is the %s story you need to work with**:\n```json\n%s\n```\n\n"+
		"You need to write a chapter: \"%d) - %s\" content (text) %s "+
		"Chapter should be written (should fit within) with approximately %d words.\n"+
		"Take your time to think about well-crafted chapter that fits the plot, enhances the narrative, and makes logical sense.\n"+
		"%s\n"+
		"%s\n"+
		"Answer only with the story content. No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Write only story content and nothing else. Answer only with the story chapter text.",
		a.audience,
		storyEl.ToJson(),
		chapterNumber, chapterTitle, chapterIntent,
		words,
		GeneralInstruction,
		ChapterPromptInstructions)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return removeThinking(templateResponse)
}

func (a *AI) FigureStoryLocation(storyEl story.Story) string {
	systemPrompt := "You are helping to prepare a story book. Story location that you are building (writing) will be used later on when story itself will be written."

	userPrompt := fmt.Sprintf("Create and describe a location where the story will take place. "+
		"**This is the %s Story you need to work with**:\n```json\n%s\n```\n\n"+
		"Be creative while creating this story world. "+
		"Do not mention protagonist or villain. "+
		"Take into consideration Story Suggestion. "+
		"Keep the world within time period that the story is taking place in. "+
		"Keep the world size in line with story length. We will not be able to cram huge world into 2 minute story. "+
		"Same applies other way around, we should have big enough world for longer stories. "+
		"Specific details are good. "+
		"Where who lives and other places around the protagonist(s) and villain are important as there most often the action (story) will happen. "+
		"Dont be afraid to expand the world with more locations if you see that will benefit the upcoming story. "+
		"Make the world so it is easy to imagine for %s. "+
		"If writing for children then make interesting but not excessively complicated, so that little readers have no problem understanding it.\n"+
		"%s\n"+
		"Answer only with the location text (content). No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the story location text.",
		a.audience,
		storyEl.ToJson(),
		a.audience,
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return removeThinking(templateResponse)
}

func (a *AI) TranslateSimpleText(englishText, toLanguage string) string {
	systemPrompt := "You are a translator."

	userPrompt := fmt.Sprintf("Provide good translation. **This is the text you need to translate**:\n```\n%s\n```\n\n"+
		"Translate from English to %s.\n"+
		"%s\n"+
		"Answer only with the translated text. No yapping. No other explanations or unrelated notes or remarks are necessary. Dont explain yourself. Answer only with the translation.",
		englishText,
		toLanguage,
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return cleanResponse(templateResponse)
}

func (a *AI) TranslateText(englishText, toLanguage string) string {
	systemPrompt := "You are translating single chapter for a story book."

	userPrompt := fmt.Sprintf("Inspect given English text carefully and provide good translation. **This is the text you need to translate**:\n```\n%s\n```\n\n"+
		"Translate from English to %s.\n"+
		"Maintain the feeling and vibe of the original text.\n"+
		"Target audience is \"%s\", so translate accordingly to match it in a way that target audience are able to easily understand the translation.\n"+
		"Keep original text newlines as is.\n"+
		"%s\n"+
		"Answer only with the translated text. No yapping. No other explanations or unrelated notes or remarks are necessary. Dont explain yourself. Answer only with the translation.",
		englishText,
		toLanguage,
		a.audience,
		GeneralInstruction)

	templateResponse, err := a.generate(systemPrompt, userPrompt, false)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	templateResponse = cleanResponse(templateResponse)

	return templateResponse
}

func cleanResponse(response string) string {
	response = removeThinking(response)
	response = strings.Replace(response, "\u201c", "\"", -1)
	response = strings.Replace(response, "\u201d", "\"", -1)

	// Clean JSON response
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	return response
}

func removeThinking(response string) string {
	lines := strings.Split(response, "\n")
	clean := make([]string, 0)
	inside := false
	for _, line := range lines {
		if strings.Contains(line, "<think>") {
			inside = true
			continue
		}
		if !inside {
			clean = append(clean, line)
		}
		if strings.Contains(line, "</think>") {
			inside = false
		}
	}

	return story.RemoveEmojis(strings.Join(clean, "\n"))
}
