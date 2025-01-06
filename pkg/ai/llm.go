package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/andrejsstepanovs/storygen/pkg/story"
	"github.com/spf13/viper"
	"github.com/teilomillet/gollm"
)

type AI struct {
	client gollm.LLM
}

func NewAI() (*AI, error) {
	provider := viper.GetString("STORYGEN_PROVIDER")
	keys := []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
	}
	apiKey := ""
	for _, key := range keys {
		if !strings.Contains(strings.ToLower(key), strings.ToLower(provider)) {
			continue
		}
		apiKey = viper.GetString(key)
		if apiKey != "" {
			log.Printf("Using key %s\n", key)
			break
		}
	}

	conn, err := gollm.NewLLM(
		gollm.SetProvider(provider),
		gollm.SetModel(viper.GetString("STORYGEN_MODEL")),
		gollm.SetAPIKey(apiKey),
		gollm.SetMaxRetries(30),
		gollm.SetRetryDelay(time.Second*5),
		gollm.SetLogLevel(gollm.LogLevelInfo),
		gollm.SetMaxTokens(4096),
	)
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
		return nil, err
	}

	return &AI{
		client: conn,
	}, nil
}

func (a *AI) FigureStoryStructures() []string {

	return []string{}
}

func (a *AI) FigureStoryProtagonists() []string {

	return []string{}
}

func (a *AI) FigureStoryLocations() []string {

	return []string{}
}

func (a *AI) FigureStoryMorales(storyEl story.Story, morales story.Morales) story.Morales {
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
		"Create a list of morale names that will fit the children story we will write:\n```json\n{{.Story}}\n```\n\n"+
			"Pick morales (`name`) from list of available morales:\n```\njson{{.Morales}}\n```"+
			"Be flexible with your picks. We want a vibrant, exciting story and morale is really important and needs to be interesting. ",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a children story ideas that will be used later on."),
			gollm.WithOutput("List of morale names strings (as array) in JSON format"),
			gollm.WithExamples([]string{moraleExample(2), moraleExample(3)}...),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Morales": morales.ToJson(),
		"Story":   storyEl.ToJson(),
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
		"Create Villain for this story:\n```json\n{{.Story}}\n```\n\n"+
			"Keep it simple and do not build backstory or villain characteristics or motives. "+
			"Take into consideration Story Suggestion. "+
			"That kind of details are irelevant right now and will actually harm the story building part that will come next, "+
			"so be mindful about it. Just short description about who the villain(s) is/are. "+
			"It is OK to not have a villain if it dont belong to the story we're writing.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a children story book. Villain that you are building (writing) will be used later on when story itself will be written."),
			gollm.WithOutput("Sort description and name of the villain(s) or nothing."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story": storyEl.ToJson(),
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
		"Create and story plan about the story. **This is the Story you need to work with**:\n```json\n{{.Story}}\n```\n\n"+
			"Follow main ideas that are already prepared for the story. "+
			"Be careful building story plan in a way that existing story you are working with (from json above) fits good. "+
			"Make sure you work with Story structure that was picked. We want our plan to align with picked story structure. "+
			"Keep in mind story length. "+
			"Take into consideration Story Suggestion. "+
			"Same goes for picked story morales. Summary and plan should match picked story morales. "+
			"Story plan should be quite brief and short list of things that will happen in the story with no specifics. Details will be written later on. "+
			"Write the plan in a way that the writer later on will not be much constrained with. We want to keep story plan loose and flexible (no details). "+
			"It will be part of bed time story for children.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a children story book."),
			gollm.WithOutput("Story summary and story plan to help the writer later on when they will write the story."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story": storyEl.ToJson(),
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

func (a *AI) FigureStoryChapterTitles(storyEl story.Story, chapterCount int) []string {
	templatePrompt := gollm.NewPromptTemplate(
		"StoryChapterTitleCreator",
		"Create a story chapter titles.",
		"Create a list of story chapter titles that will be used for this story:\n```json\n{{.Story}}\n```\n\n"+
			"Make sure that chapter titles align with existing story details. "+
			"Take into consideration Story Suggestion. "+
			"Be mindful about the chapter count so it aligns good with story length. Usually there is no need for more than {{.Count}} chapters. "+
			"Write chapters so the plot can move forward and is aligned with defined story structure requirements. ",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a children story content chapter titles."),
			gollm.WithOutput("List of chapter titles strings (as array) in JSON format. No other text should be present. Only JSON."),
			gollm.WithExamples([]string{"The Mysterious Map", "The Magic Paintbrush", "The Rainbow Bridge", "The final battle", "The Return to Home Sweet Home"}...),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story": storyEl.ToJson(),
		"Count": chapterCount,
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
			"**This is the Story you need to work with**:\n```json\n{{.Story}}\n```\n\n"+
			"Take into consideration Story Suggestion.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are summarizing a children story book."),
			gollm.WithOutput("Answer only with the summary. No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the Summary text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story": storyEl.ToJson(),
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
		"Write a book name (title) for this story. **This is the Story you need to work with**:\n```json\n{{.Story}}\n```\n\n",
		gollm.WithPromptOptions(
			gollm.WithContext("You are writing a children story book title."),
			gollm.WithExamples([]string{"The Secret Library of Wishes", "The Brave Little Firefly", "The girl and the Talking Tree"}...),
			gollm.WithOutput("Answer only with the title.  No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the Title text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story": storyEl.ToJson(),
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
		"Analyze a story and write a single chapter for it.",
		"Write the single full chapter text, ensuring it flows naturally and keeps the reader engaged. "+
			"**This is the Story you need to work with**:\n```json\n{{.Story}}\n```\n\n"+
			"You need to write a chapter: \"{{.Number}}) - {{.Title}}\" content (text) to proceed the storyline. "+
			"# Instructions:\n"+
			"- Chapter should be written (should fit within) with approximately {{.Words}} words. \n"+
			"- Analyze previous chapters (if exists) before writing the next one. \n"+
			"- Use simple language without complex words. Story is targeting small children that dont know english very well. \n"+
			"- Proceed the storyline in a way that fits the chapter's place in the story.\n"+
			"- Use all provided story details (characters, setting, plot, morals, etc.) to create a rich, imaginative, and engaging chapter.\n"+
			"- Ensure the chapter aligns with the story's structure, timeline, themes, protagonist, villain, and overall plan.\n"+
			"- Take into consideration Story Suggestion. \n"+
			"- Write it using funny interactions between characters. \n"+
			"- Move plot forward without diving into surrounding details. Tell what happened and what happened next moving plot forward. \n\n"+
			"# Writing style Adjustments:\n"+
			"You often use descriptive phrases or clauses to extend sentences. "+
			"While they add great imagery, they can feel repetitive if overused. "+
			"Try mixing it up with shorter, punchier sentences or different ways of describing actions and settings! "+
			"It’ll help keep the pacing fresh and engaging!",
		gollm.WithPromptOptions(
			gollm.WithContext("You are writing a children story book chapter by chapter. Expand the story with one chapter."),
			gollm.WithDirectives("You are creative and decisive story writer."),
			gollm.WithOutput("Answer only with the story content. No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the story chapter text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story":  storyEl.ToJson(),
		"Title":  chapterTitle,
		"Number": chapterNumber,
		"Words":  words,
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
		"Create and describe a location where the story will take place. **This is the Story you need to work with**:\n```json\n{{.Story}}\n```\n\n"+
			"Be creative while creating this story world. "+
			"Do not mention protagonist or villain. "+
			"Take into consideration Story Suggestion. "+
			"Keep the world within time period that the story is taking place in. "+
			"Keep the world size in line with story length. We will not be able to cram huge world into 2 minute story. "+
			"Same applies other way around, we should have big enough world for longer stories. "+
			"Specific details are good. "+
			"Where who lives and other places around the protagonist and villan are important as there most often the action (story) will happen. "+
			"Dont be afraid to expand the world with more locations if you see that will benefit the upcoming story. "+
			"Make the world so it is easy to imagine for a child. "+
			"It will be part of bed time story for children so make it also interesting but not excessive complicated, so that children have no problem understanding it.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are helping to prepare a children story book. Story location that you are building (writing) will be used later on when story itself will be written."),
			gollm.WithOutput("Answer only with the title. No yapping. No other explanations or unrelated to title text is necessary. Dont explain yourself. Answer only with the story location text."),
		),
	)

	prompt, err := templatePrompt.Execute(map[string]interface{}{
		"Story": storyEl.ToJson(),
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

func (a *AI) TranslateText(englishText, toLanguage string) string {
	templatePrompt := gollm.NewPromptTemplate(
		"Translator",
		fmt.Sprintf("Analyze given English language text and provide good translation in **%s** language).", toLanguage),
		"Inspect given English text carefully and provide good translation. **This is the text you need to translate**:\n```\n{{.Text}}\n```\n\n"+
			"Translate from English to {{.Language}}.\n"+
			"Maintain the feeling and vibe of the original text. "+
			"It will be part of bed time story for children so translate accordingly. "+
			"Children should be able to easily understand the translation. "+
			"Keep original text newlines as is.",
		gollm.WithPromptOptions(
			gollm.WithContext("You are translating a children story book."),
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

	templateResponse = gollm.CleanResponse(templateResponse)

	return templateResponse
}
