package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/andrejsstepanovs/storygen/pkg/story"
	"github.com/teilomillet/gollm"
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
	"- Move plot forward without diving into surrounding details. " +
	"Tell what happened and what happened next moving plot forward.\n\n" +
	"# Writing style Adjustments:\n" +
	"You often use descriptive phrases or clauses to extend sentences. " +
	"While they add great imagery, they can feel repetitive if overused. " +
	"Try mixing it up with shorter, punchier sentences or different ways of describing actions and settings! " +
	"It’ll help keep the pacing fresh and engaging!" +
	"Another thing - laughing and dancing is nice but too much is cringe."

func (a *AI) SuggestStoryFixes(storyEl story.Story, problem story.Problem, addressedSuggestions story.Suggestions) story.Suggestions {
	problemInjsonTxt := ""
	for i := 0; i < 3; i++ {
		suggestions, query, err := a.trySuggestStoryFixes(storyEl, problem, addressedSuggestions, problemInjsonTxt)
		if err == nil {
			return suggestions
		}
		log.Println("Failed to suggest story fixes for problem chapter. Trying again.")
		problemInjsonTxt = fmt.Sprintf("Your last answer contained invalid JSON: ----\n\n%s\n\n----. Try again and this time make sure your JSON is valid!", query)
	}

	log.Fatalf("Failed to suggest story fixes for problem chapter: %v", problem.Chapter)
	return story.Suggestions{}
}

func (a *AI) trySuggestStoryFixes(storyEl story.Story, problem story.Problem, addressedSuggestions story.Suggestions, problemInjsonTxt string) (story.Suggestions, string, error) {
	if problem.Chapter < len(storyEl.Chapters) {
		storyEl.Chapters = storyEl.Chapters[:problem.Chapter]
	}

	suggestions := story.Suggestions{
		{
			Chapter:     1,
			ChapterName: "The Beginning",
			Suggestions: []string{
				"Introduce a dolphin that was following the boat in chapter 3",
				"Make protagonist angry at the doctor for not knowing the cat name, because this will be useful in ending chapter.",
			},
		},
		{
			Chapter:     problem.Chapter,
			ChapterName: problem.ChapterName,
			Suggestions: []string{
				"Add a scene where the doctor is told about the cat name",
				"Show where the doctor is told about the cat name",
			},
		},
	}

	templatePrompt := gollm.NewPromptTemplate(
		"StoryChapterFixer",
		"Our story auditor (pre-reader) found issues in story chapter. Pick what story chapters (chapter number) need to be re-written and suggest how to do it.",
		"Analyze the {{.Audience}} story chapter {{.ChapterNumber}} {{.ChapterName}} issues:\n"+
			"<issues>\n{{.Issues}}\n</issues>\n\n"+
			"Analyze full {{.Audience}} story and adjust pinpoint chapter numbers that need adjustments and suggestions how to do it.\n"+
			"For reference, here is full Story until chapter \n```json\n{{.StoryChapters}}\n```\n. "+
			"Already addressed suggestions that you should ignore \n```json\n{{.AddressedSuggestions}}\n```\n. "+
			"There are maybe more chapters but lets focus on story until this moment. "+
			"Think about what needs to be changed in what chapter before re-writing."+
			"# Instructions:"+
			"- Fix this or past chapters so story is coherent, entertaining and makes sense (use given suggestions).\n"+
			"- Don't challenge (and keep) the {{.Audience}} story writing style.\n"+
			"- Story writing style was already predefined and we are sticking with it.\n"+
			"- You do not need to re-write the chapter text, just suggestions how to do it and where and in what chapter.\n"+
			"- Dont be overly pedantic. It's just a {{.Audience}} story after all.\n"+
			"- Be creative with suggestions to fix issues at hand.\n"+
			"- Be swift and decisive. Suggest changes that can be done with reasonable amount of new text. "+
			"- It is OK to extend the story if that is necessary to fix the plot.\n"+
			"- Don't suggest creating new chapters. We are sticking with existing chapter count.\n"+
			"# Answer:"+
			"- Return empty JSON array (`[]`) if there is nothing important to fix. "+problemInjsonTxt,
		gollm.WithPromptOptions(
			gollm.WithContext("You are story writer that is suggesting a fixes for story chapters to resolve found issues. Your suggestions will be used to re-write the story chapters later on."),
			gollm.WithOutput("Answer only JSON array with columns 'chapter_number_int', 'chapter_name', 'suggestions_array_string'. No yapping. No other explanations or unrelated text is necessary. Dont explain yourself. Answer only with JSON content. Be careful generating JSON, it needs to be valid. "+problemInjsonTxt),
			gollm.WithExamples(suggestions.ToJson()),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Issues":               problem.ToJson(),
		"StoryChapters":        storyEl.ToJson(),
		"AddressedSuggestions": addressedSuggestions.ToJson(),
		"ChapterNumber":        problem.Chapter,
		"ChapterName":          problem.ChapterName,
		"Audience":             a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt, gollm.WithJSONSchemaValidation())
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	var picked []story.Suggestion
	err = json.Unmarshal([]byte(templateResponse), &picked)
	if err != nil {
		responseJson := gollm.CleanResponse(templateResponse)
		if responseJson != "[]" {
			log.Println("Failed to parse JSON. Trying again")
			responseJson = fmt.Sprintf("[%s]", responseJson)
			err = json.Unmarshal([]byte(responseJson), &picked)
			if err != nil {
				log.Println(templateResponse)
				log.Println("cleaned:", responseJson)
				return story.Suggestions{}, templateResponse, err
			}
		}
	}

	var p story.Suggestions
	p = picked
	return p, "", nil
}

func (a *AI) AdjustStoryChapter(storyEl story.Story, problem story.Problem, suggestions story.Suggestions, addressedSuggestions story.Suggestions, wordCount int) string {
	if problem.Chapter < len(storyEl.Chapters) {
		storyEl.Chapters = storyEl.Chapters[:problem.Chapter]
	}

	templatePrompt := gollm.NewPromptTemplate(
		"StoryChapterFixer",
		"There are issues in this story chapter. Re-write the chapter and fix all mentioned problems",
		"Re-write the {{.Audience}} Story chapter {{.ChapterNumber}} {{.ChapterName}}. "+
			"Issues found: \n<issues>\n{{.Issues}}\n</issues>\n\n"+
			"Analyze full {{.Audience}} Story and adjust the problematic chapter {{.ChapterNumber}} {{.ChapterName}}.\n"+
			"Here are all already addressed suggestions: \n<already_addressed_suggestions>\n{{.AddressedSuggestions}}\n</already_addressed_suggestions>\n"+
			"**IMPORTANT**: Suggestions how to fix the issues at hand: \n<fix_suggestions>\n{{.Suggestions}}\n</fix_suggestions>\n"+
			"Use and rely only on these suggestions provided!\n"+
			"For reference, here is full story until this chapter ```json\n{{.StoryChapters}}\n```. "+
			"# Orders:"+
			"- There are maybe more chapters but lets focus on story until this moment.\n"+
			"- Fix only this chapter so story is coherent, entertaining and makes sense (use given suggestions). "+
			"- Use suggestions from fix_suggestions tag to re-write the story chapter {{.ChapterNumber}} {{.ChapterName}} as suggested. "+
			"- Make sure you don't break out of suggestions that were fixed before (see json in: already_addressed_suggestions tags). "+
			"- Answer with only one chapter text. We are fixing it one chapter at the time. "+
			"- Be creative to fix the issue at hand. Be swift and decisive. No need for long texts, we just need to fix these issues and move on. "+
			//"- It is OK to extend the story if that is necessary to fix the plot. "+
			"- Small text extensions are OK, but we should try to keep this chapter withing a limit of {{.Words}} words."+
			ChapterPromptInstructions,
		gollm.WithPromptOptions(
			gollm.WithContext("You are story writer that is fixing story issues before it goes to publishing."),
			gollm.WithOutput("Story chapter text. Answer with story chapter text only. We need nothing else than just this one chapter with fixed content. No yapping. No other explanations or unrelated text is necessary. Dont explain yourself. Answer only with this one fixed chapter text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Issues":               problem.ToJson(),
		"StoryChapters":        storyEl.ToJson(),
		"Suggestions":          suggestions.ToJson(),
		"AddressedSuggestions": addressedSuggestions.ToJson(),
		"ChapterNumber":        problem.Chapter,
		"ChapterName":          problem.ChapterName,
		"Audience":             a.audience,
		"Words":                wordCount,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return templateResponse
}

func (a *AI) FigureStoryLogicalProblems(storyText string, loop, maxLoops int) story.Problems {
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

	templatePrompt := gollm.NewPromptTemplate(
		"StoryIssueSpotter",
		"Pre read the story and figure out the logical issues.",
		"Create a JSON problem list for {{.Audience}} story we need to check (pre-read):\n"+
			"<story_text>\n{{.StoryText}}\n</story_text>\n\n"+
			"Find problems and flaws in the plot and answer with formatted output as mentioned in examples. "+
			"Carefully read the story text chapter by chapter and analyze it for logical flaws in the story in each chapter."+
			"This is cycle {{.Loop}} of pre-reading. Reduce strictness and issue count proportionally to the number of cycles completed. Max cycles: {{.MaxLoops}}.\n"+
			"If no flaws are found, do not include the chapter in your output.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to pre-read a story and your output will help us to fix the story flaws."),
			gollm.WithOutput("JSON of story issues (problems) (as array) in JSON format. Use only protagonists from the list that was provided."),
			gollm.WithOutput("Answer only JSON array with columns 'chapter_number_int', 'chapter_name', 'issues_array_string'. No yapping. No other explanations or unrelated text is necessary. Dont explain yourself. Answer only with JSON content."),
			gollm.WithExamples(problems.ToJson()),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"StoryText": storyText,
		"Audience":  a.audience,
		"Loop":      loop,
		"MaxLoops":  maxLoops,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt, gollm.WithJSONSchemaValidation())
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	var picked []story.Problem
	err = json.Unmarshal([]byte(templateResponse), &picked)
	if err != nil {
		responseJson := gollm.CleanResponse(templateResponse)
		responseJson = fmt.Sprintf("[%s]", responseJson)
		err = json.Unmarshal([]byte(responseJson), &picked)
		if responseJson != "[]" {
			if err != nil {
				log.Println(templateResponse)
				log.Println("cleaned:", responseJson)
				log.Fatalf("Failed to parse problems as JSON: %v", err)
			}
		}
	}

	ret := make(story.Problems, 0)
	for _, pr := range picked {
		if len(pr.Issues) > 0 {
			ret = append(ret, pr)
		}
	}

	return ret
}

func (a *AI) FigureStoryProtagonists(storyEl story.Story) story.Protagonists {
	examples := func(count int) string {
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

	templatePrompt := gollm.NewPromptTemplate(
		"ProtagonistsPicker",
		"Pick protagonists that will be a good fit for the given story.",
		"Create a JSON protagonists list that will fit the {{.Audience}} story we will write:\n```json\n{{.Story}}\n```\n\n"+
			"Pick protagonist elements from of available protagonists elements:\n```\njson{{.Protagonists}}\n```\n"+
			"Be mindful about how many you are picking. "+
			"It is totally OK to pick single or multiple same types of protagonists as they're personas will be extended later on with more details."+
			"Your task now is to pick from the list.\n"+
			"Be creative with your picks. We want a vibrant, exciting story and protagonists are/is important and needs to be suitable and interesting. ",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a story ideas that will be used later on."),
			gollm.WithOutput("JSON of protagonist elements (as array) in JSON format. Use only protagonists from the list that was provided."),
			gollm.WithOutput("Answer only JSON array with columns 'type', 'gender', 'size', 'age'. All parameters must be string (also age is string). No yapping. No other explanations or unrelated text is necessary. Dont explain yourself. Answer only with JSON content."),
			gollm.WithExamples(examples(2)),
		),
	)

	allTimePeriods := story.GetAvailableTimePeriods()
	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"TimePeriods": allTimePeriods.ToJson(),
		"Story":       storyEl.ToJson(),
		"Audience":    a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt, gollm.WithJSONSchemaValidation())
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	var picked []story.Protagonist
	err = json.Unmarshal([]byte(templateResponse), &picked)
	if err != nil {
		log.Println("Failed to parse JSON. Trying again")
		responseJson := gollm.CleanResponse(templateResponse)
		responseJson = fmt.Sprintf("[%s]", responseJson)
		err = json.Unmarshal([]byte(responseJson), &picked)
		if err != nil {
			log.Println(templateResponse)
			log.Println("cleaned:", responseJson)
			log.Fatalf("Failed to parse time protagonists as JSON: %v", err)
		}
	}

	var p story.Protagonists
	p = picked
	return p
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

	templatePrompt := gollm.NewPromptTemplate(
		"MoralesPicker",
		"Pick all morales that will be a good fit for the given story.",
		"Create a list of morale names that will fit the {{.Audience}} story we will write:\n```json\n{{.Story}}\n```\n\n"+
			"Pick morales (`name`) from list of available morales:\n```\njson{{.Morales}}\n```"+
			"Be flexible with your picks. We want creative choices for exciting story.\n"+
			"Do not be afraid to pick something (I noticed you always pick Courage) that is not fitting perfectly. The more the better.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a story ideas that will be used later on."),
			gollm.WithOutput("List of morale names strings (as array) in JSON format. Don't explain your choice. No yapping. Answer only with the morale names in JSON array."),
			gollm.WithExamples(moraleExample(3)),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Morales":  morales.ToJson(),
		"Story":    storyEl.ToJson(),
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt, gollm.WithJSONSchemaValidation())
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	responseJson := gollm.CleanResponse(templateResponse)
	responseJson = gollm.CleanResponse(responseJson)

	var picked []string
	err = json.Unmarshal([]byte(responseJson), &picked)
	if err != nil {
		log.Println(templateResponse)
		log.Println(responseJson)
		log.Fatalf("Failed to parse morales response as JSON: %v", err)
	}

	//fmt.Printf("%s", strings.Join(picked, ","))

	return story.FindMoralesByName(picked)
}

func (a *AI) FigureStoryVillain(storyEl story.Story) string {
	templatePrompt := gollm.NewPromptTemplate(
		"VillainGenerator",
		"Analyze a story and come up with a villain for this story that will fit good.",
		"Create Villain for this {{.Audience}} story:\n```json\n{{.Story}}\n```\n\n"+
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
			"By the way, villain can also be elements of nature or unmovable objects and that kind of stuff. "+
			"Depends on the story we're building. Be creative if possible. ",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a story book. Villain that you are building (writing) will be used later on when story itself will be written."),
			gollm.WithOutput("Sort description and name of the villain(s) or nothing. No yapping. Don't explain your choice. Answer only with the villain(s) description."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story":    storyEl.ToJson(),
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return templateResponse
}

func (a *AI) FigureStoryPlan(storyEl story.Story) string {
	templatePrompt := gollm.NewPromptTemplate(
		"StoryPlanGenerator",
		"Analyze a story and come up with a story plan that will be useful to follow for a writer later on.",
		"Create and {{.Audience}} story plan about the story. **This is the Story you need to work with**:\n```json\n{{.Story}}\n```\n\n"+
			"Follow main ideas that are already prepared for the story. "+
			"Be careful building story plan in a way that existing story you are working with (from json above) fits good. "+
			"Make sure you work with Story structure that was picked. We want our plan to align with picked story structure. "+
			"Keep in mind story length. "+
			"Take into consideration Story Suggestion. "+
			"Same goes for picked story morales. Summary and plan should match picked story morales. "+
			"Story plan should be quite brief and short list of things that will happen in the story with no specifics. Details will be written later on. "+
			"Write the plan in a way that the writer later on will not be much constrained with. We want to keep story plan loose and flexible (no details). "+
			"Be creative and make sure that this {{.Audience}} story is moving forward fast so it is engaging and fun to read. "+
			"Plan a story in a way where there are no boring parts and plot is moving forward fast. "+
			"Consider adding some plot twists and funny interactions between characters.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a story book."),
			gollm.WithOutput("Story summary and story plan to help the writer later on when they will write the story."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story":    storyEl.ToJson(),
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return templateResponse
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

	templatePrompt := gollm.NewPromptTemplate(
		"TimePeriodPicker",
		"Pick time periods that will be a good fit for the given story.",
		"Create a list of time periods that will fit the {{.Audience}} story we will write:\n```json\n{{.Story}}\n```\n\n"+
			"Pick time periods (`name`) from list of available time periods:\n```\njson{{.TimePeriods}}\n```"+
			"Be flexible with your picks. We want a vibrant, exciting story and time period is important and needs to be suitable and interesting. ",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a story ideas that will be used later on."),
			gollm.WithOutput("List of time period names strings (as array) in JSON format"),
			gollm.WithOutput("Answer only JSON. No yapping. No other explanations or unrelated text is necessary. Dont explain yourself. Answer only with JSON."),
			gollm.WithExamples(timePeriodExample(3)),
		),
	)

	allTimePeriods := story.GetAvailableTimePeriods()
	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"TimePeriods": allTimePeriods.ToJson(),
		"Story":       storyEl.ToJson(),
		"Audience":    a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt, gollm.WithJSONSchemaValidation())
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	responseJson := gollm.CleanResponse(templateResponse)
	responseJson = gollm.CleanResponse(responseJson)

	var picked []string
	err = json.Unmarshal([]byte(responseJson), &picked)
	if err != nil {
		log.Println(templateResponse)
		log.Println(responseJson)
		log.Fatalf("Failed to parse time period response as JSON: %v", err)
	}

	return story.FindTimePeriodsByName(picked)[0]
}

func (a *AI) FigureStoryChapterTitles(storyEl story.Story, chapterCount int) []string {
	templatePrompt := gollm.NewPromptTemplate(
		"StoryChapterTitleCreator",
		"Create a story chapter titles.",
		"Create a list of story chapter titles that will be used for this {{.Audience}} story:\n```json\n{{.Story}}\n```\n\n"+
			"Make sure that chapter titles align with existing story details. "+
			"Take into consideration Story Suggestion. "+
			"Be mindful about the chapter count so it aligns good with story length. Usually there is no need for more than {{.Count}} chapters. "+
			"Write chapters so the plot can move forward and is aligned with defined {{.Audience}} story structure requirements. ",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a story content chapter titles."),
			gollm.WithOutput("List of chapter titles strings (as array) in JSON format. No other text should be present. Only JSON."),
			gollm.WithExamples([]string{"The Mysterious Map", "The Magic Paintbrush", "The Rainbow Bridge", "The final battle", "The Return to Home Sweet Home"}...),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story":    storyEl.ToJson(),
		"Count":    chapterCount,
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt, gollm.WithJSONSchemaValidation())
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	responseJson := gollm.CleanResponse(templateResponse)
	responseJson = gollm.CleanResponse(responseJson)

	var picked []string
	err = json.Unmarshal([]byte(responseJson), &picked)
	if err != nil {
		fmt.Println(templateResponse)
		log.Fatalf("Failed to parse chapters response as JSON: %v", err)
	}

	return picked
}

func (a *AI) FigureStorySummary(storyEl story.Story) string {
	templatePrompt := gollm.NewPromptTemplate(
		"StorySummaryGenerator",
		"Analyze a story and summarize it in 1 sentence.",
		"Create 1 sentence story summary for this story. "+
			"**This is the {{.Audience}} story you need to work with**:\n```json\n{{.Story}}\n```\n\n"+
			"If exists, take into consideration Story Suggestion.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are summarizing a story book."),
			gollm.WithOutput("Answer only with the summary. No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the Summary text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story":    storyEl.ToJson(),
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return templateResponse
}

func (a *AI) FigureStoryTitle(storyEl story.Story) string {
	templatePrompt := gollm.NewPromptTemplate(
		"StoryTitleGenerator",
		"Analyze a story and come up with creative book name for the story.",
		"Write a book name (title) for this {{.Audience}} story. **This is the {{.Audience}} Story you need to work with**:\n```json\n{{.Story}}\n```\n\n",
		gollm.WithPromptOptions(
			gollm.WithContext("You are writing a story book title."),
			gollm.WithExamples([]string{"The Secret Library of Wishes", "The Brave Little Firefly", "The girl and the Talking Tree"}...),
			gollm.WithOutput("Answer only with the title.  No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the Title text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story":    storyEl.ToJson(),
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return templateResponse
}

func (a *AI) FigureStoryChapter(storyEl story.Story, chapterNumber int, chapterTitle string, words int) string {
	templatePrompt := gollm.NewPromptTemplate(
		"CreativeStoryChapterWriter",
		"Analyze story data and write a single chapter for it.",
		"Write the single full chapter text, ensuring it flows naturally and keeps the reader engaged. "+
			"**This is the {{.Audience}} story you need to work with**:\n```json\n{{.Story}}\n```\n\n"+
			"You need to write a chapter: \"{{.Number}}) - {{.Title}}\" content (text) to proceed the storyline. "+
			"Chapter should be written (should fit within) with approximately {{.Words}} words.\n"+
			ChapterPromptInstructions,
		gollm.WithPromptOptions(
			gollm.WithContext("You are writing a story book chapter by chapter. Expand the story with one chapter."),
			gollm.WithDirectives("You are creative and decisive story writer."),
			gollm.WithOutput("Answer only with the story content. No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the story chapter text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story":    storyEl.ToJson(),
		"Title":    chapterTitle,
		"Number":   chapterNumber,
		"Words":    words,
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return templateResponse
}

func (a *AI) FigureStoryLocation(storyEl story.Story) string {
	templatePrompt := gollm.NewPromptTemplate(
		"LocationGenerator",
		"Analyze a story and come up with a story location that will fit good.",
		"Create and describe a location where the story will take place. "+
			"**This is the {{.Audience}} Story you need to work with**:\n```json\n{{.Story}}\n```\n\n"+
			"Be creative while creating this story world. "+
			"Do not mention protagonist or villain. "+
			"Take into consideration Story Suggestion. "+
			"Keep the world within time period that the story is taking place in. "+
			"Keep the world size in line with story length. We will not be able to cram huge world into 2 minute story. "+
			"Same applies other way around, we should have big enough world for longer stories. "+
			"Specific details are good. "+
			"Where who lives and other places around the protagonist(s) and villain are important as there most often the action (story) will happen. "+
			"Dont be afraid to expand the world with more locations if you see that will benefit the upcoming story. "+
			"Make the world so it is easy to imagine for {{.Audience}}. "+
			"If writing for children then make interesting but not excessively complicated, so that little readers have no problem understanding it.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a story book. Story location that you are building (writing) will be used later on when story itself will be written."),
			gollm.WithOutput("Answer only with the title. No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the story location text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story":    storyEl.ToJson(),
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return templateResponse
}

func (a *AI) TranslateSimpleText(englishText, toLanguage string) string {
	templatePrompt := gollm.NewPromptTemplate(
		"Translator",
		fmt.Sprintf("Analyze given English language text and provide good translation in **%s** language).", toLanguage),
		"Provide good translation. **This is the text you need to translate**:\n```\n{{.Text}}\n```\n\n"+
			"Translate from English to {{.Language}}.",
		gollm.WithPromptOptions(
			gollm.WithOutput("Answer only with the translated text. No yapping. No other explanations or unrelated notes or remarks are necessary. Dont explain yourself. Answer only with the translation."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Text":     englishText,
		"Language": toLanguage,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	return gollm.CleanResponse(templateResponse)
}

func (a *AI) TranslateText(englishText, toLanguage string) string {
	templatePrompt := gollm.NewPromptTemplate(
		"StoryChapterTranslator",
		fmt.Sprintf("Analyze given English language text and provide good translation in **%s** language).", toLanguage),
		"Inspect given English text carefully and provide good translation. **This is the text you need to translate**:\n```\n{{.Text}}\n```\n\n"+
			"Translate from English to {{.Language}}.\n"+
			"Maintain the feeling and vibe of the original text. "+
			"Target audience is {{.Audience}} so translate accordingly. "+
			"{{.Audience}} should be able to easily understand the translation. "+
			"Keep original text newlines as is.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are translating single chapter for a story book."),
			gollm.WithOutput("Answer only with the translated text. No yapping. No other explanations or unrelated notes or remarks are necessary. Dont explain yourself. Answer only with the translation."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Text":     englishText,
		"Language": toLanguage,
		"Audience": a.audience,
	})
	if err != nil {
		log.Fatalf("Failed to execute prompt template: %v", err)
	}

	ctx := context.Background()
	templateResponse, err := a.client.Generate(ctx, prompt)
	if err != nil {
		log.Fatalf("Failed to generate template response: %v", err)
	}

	templateResponse = gollm.CleanResponse(templateResponse)

	return templateResponse
}
