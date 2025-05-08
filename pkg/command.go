package pkg

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"github.com/andrejsstepanovs/storygen/pkg/ai"
	"github.com/andrejsstepanovs/storygen/pkg/story"
	"github.com/andrejsstepanovs/storygen/pkg/tts"
	"github.com/andrejsstepanovs/storygen/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewCommand() (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "story",
		Short: "Generate Story",
	}

	audience := viper.GetString("STORYGEN_AUDIENCE")
	if audience == "" {
		audience = "Children"
	}
	llm, err := ai.NewAI(audience)
	if err != nil {
		return nil, err
	}

	cmd.AddCommand(
		newWorkCommand(llm),
		newTranslateCommand(llm),
		newReadCommand(llm),
		newWriteCommand(llm),
		newGroomCommand(llm),
		newStoryIdeasCommand(llm, audience),
		newStoryCompareCommand(llm),
		newStoryCompetitionCommand(llm),
	)

	return cmd, nil
}

func newStoryIdeasCommand(llm *ai.AI, audience string) *cobra.Command {
	return &cobra.Command{
		Use:   "ideas",
		Short: "Provide list of idewas for stories",
		RunE: func(_ *cobra.Command, args []string) error {
			const defaultLen = 6
			l := defaultLen
			if len(args) == 1 {
				count := args[0]
				l, _ = strconv.Atoi(count)
				if l == 0 {
					l = defaultLen
				}
			}

			storyIdeas := llm.FigureStoryIdeas(l)
			log.Printf("Ideas: %d\n", len(storyIdeas))
			for _, idea := range storyIdeas {
				log.Printf("%s\n", idea)
			}
			return nil
		},
	}
}

func newStoryCompareCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "compare",
		Short: "Compare two stories. First param is path to one json file, second is path to another json file",
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 2 {
				storyAFile := args[0]
				storyBFile := args[1]
				log.Printf("%q, %q\n", storyAFile, storyBFile)
				betterStory := compareStories(llm, storyAFile, storyBFile)
				log.Printf("Story: %q is better\n", betterStory.Title)
				return nil
			}

			return nil
		},
	}
}

func newStoryCompetitionCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "competition",
		Short: "Generates x stories and compares them to find the best one.",
		RunE: func(_ *cobra.Command, args []string) error {
			count := 10
			if len(args) == 1 {
				var err error
				count, err = strconv.Atoi(args[0])
				if err != nil {
					log.Fatalln(err)
				}
			}
			log.Printf("Generating %d stories...\n", count)
			ideas := llm.FigureStoryIdeas(count)
			for i, idea := range ideas {
				log.Printf("Idea: %d - %s\n", i+1, idea)
			}

			stories := make([]story.Story, 0)
			for _, idea := range ideas {
				s := buildStory(llm, idea)
				stories = append(stories, s)
			}

			score := make(map[string]int)
			for i := 0; i < len(stories); i++ {
				for j := i + 1; j < len(stories); j++ {
					storyA := stories[i]
					storyB := stories[j]
					betterStory := llm.CompareStories(storyA, storyB)
					if betterStory.Title == storyA.Title {
						log.Printf("Story: %q is better\n", storyA.Title)
						score[storyA.Title]++
					} else {
						log.Printf("Story: %q is better\n", storyB.Title)
						score[storyB.Title]++
					}
				}
			}
			// find the best one
			sort.Slice(stories, func(i, j int) bool {
				return score[stories[i].Title] > score[stories[j].Title]
			})

			log.Printf("Best Story: %q\n", stories[0].Title)
			file, _ := refineStory(llm, stories[0], 0)
			log.Println("JSON saved")
			log.Println(file)

			ToVoice(stories[0], file, stories[0].BuildContent(story.TextChapter, story.TextTheEnd))

			return nil
		},
	}
}

func newGroomCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "groom",
		Short: "Groom the Story from JSON (first arg) and fix found issues",
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			log.Printf("Loading story from file: %s", file)

			x := &story.Story{}
			json.Unmarshal(utils.LoadTextFromFile(file), x)

			s := *x
			file, s = refineStory(llm, s, 0)

			log.Println("Done")
			log.Println(file)

			return nil
		},
	}
}

func refineStory(llm *ai.AI, s story.Story, preReadLoops int) (string, story.Story) {
	tmpDir := viper.GetString("STORYGEN_TMP_DIR")

	preReadLoops = viper.GetInt("STORYGEN_PREREAD_LOOPS")
	if preReadLoops == 0 {
		file, err := utils.SaveTextToFile(tmpDir, "final_"+s.Title, "json", s.ToJson())
		if err != nil {
			log.Fatalln(err)
		}
		return file, s
	}

	chapterCount, maxChapterWords, _ := utils.GetChapterCountAndLength()
	chapterWords := utils.ChapterWordCount(chapterCount, maxChapterWords)

	allAddressedSuggestions := make(story.Suggestions, 0)
	for i := 1; i <= preReadLoops; i++ {
		log.Printf("## Pre-reading / story fixing loop: %d...\n", i)
		text := s.BuildContent(story.TextChapter, story.TextTheEnd)

		problems := llm.FigureStoryLogicalProblems(text, i, preReadLoops)
		if len(problems) == 0 {
			log.Println("Story is OK")
			break
		}

		c := fmt.Sprintf("%d", len(problems))
		if len(problems) == len(s.Chapters) {
			c = "all"
		}
		log.Printf("Found problems in %s chapters\n", c)

		// sort problems so first problem is for chapter 1 and last one is for last chapter
		sort.Slice(problems, func(i, j int) bool {
			return problems[i].Chapter < problems[j].Chapter
		})

		chapterSuggestions := make(map[int]story.Suggestions)
		allSuggestions := make(story.Suggestions, 0)
		totalSuggestions := 0
		for _, problem := range problems {
			log.Printf("Finding suggestions how to fix chapter %d...", problem.Chapter)
			for _, c := range s.Chapters {
				if c.Number != problem.Chapter {
					continue
				}
				log.Printf("Suggesting fix suggestions for: %d. %s...", problem.Chapter, problem.ChapterName)
				suggestions := llm.SuggestStoryFixes(s, problem, allAddressedSuggestions)
				if len(suggestions) == 0 {
					continue
				}
				allSuggestions = append(allSuggestions, suggestions...)
				for _, sug := range suggestions {
					_, ok := chapterSuggestions[sug.Chapter]
					if !ok {
						chapterSuggestions[sug.Chapter] = make(story.Suggestions, 0)
					}
					chapterSuggestions[sug.Chapter] = append(chapterSuggestions[sug.Chapter], sug)
					totalSuggestions++
				}
			}
		}

		log.Printf("Found problems: %d with %d suggestions\n", len(problems), totalSuggestions)

		// sort chapterSuggestions by key
		keys := make([]int, 0, len(chapterSuggestions))
		for k := range chapterSuggestions {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		log.Println("Fixing...") // todo: fix - this is too complex and probably buggy
		for _, chapter := range keys {
			for suggestionChapter, suggestions := range chapterSuggestions {
				if suggestionChapter != chapter {
					continue
				}
				for _, problem := range problems {
					if problem.Chapter != chapter {
						continue
					}
					for j, c := range s.Chapters {
						if c.Number != chapter {
							continue
						}
						log.Printf("Adjusting chapter %d with suggestions (%d)...", chapter, suggestions.Count())
						wordCount := chapterWords[chapter]
						fixedChapter := llm.AdjustStoryChapter(s, problem, suggestions, allAddressedSuggestions, wordCount)
						if fixedChapter != "" {
							s.Chapters[j].Text = fixedChapter
						}
					}
				}
			}
		}

		utils.SaveTextToFile(tmpDir, strconv.Itoa(i)+"_groomed_"+s.Title, "json", s.ToJson())
		allAddressedSuggestions = append(allAddressedSuggestions, allSuggestions...)
	}

	file, err := utils.SaveTextToFile(tmpDir, "final_groomed_"+s.Title, "json", s.ToJson())
	if err != nil {
		log.Fatalln(err)
	}

	return file, s
}

func newTranslateCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "voice",
		Short: "Load a Story from JSON file",
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			log.Printf("Loading story from file: %s", file)

			s := &story.Story{}
			json.Unmarshal(utils.LoadTextFromFile(file), s)

			translated := *s
			chapter := story.TextChapter
			theEnd := story.TextTheEnd
			toLang := strings.ToLower(viper.GetString("STORYGEN_LANGUAGE"))
			if toLang != "english" {
				log.Printf("Translating to: %s", toLang)
				translated, chapter, theEnd = translate(llm, *s, toLang)
			}

			soundFile := file[:len(file)-4] + "mp3"
			ToVoice(translated, toLang+"_"+soundFile, translated.BuildContent(chapter, theEnd))

			return nil
		},
	}
}

func newReadCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "read",
		Short: "Load a Story from JSON (first arg) and shows story text",
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			log.Printf("Loading story from file: %s", file)

			s := &story.Story{}
			json.Unmarshal(utils.LoadTextFromFile(file), s)

			translated := *s

			text := translated.BuildContent(story.TextChapter, story.TextTheEnd)
			fmt.Println(text)
			return nil
		},
	}
}

func newWriteCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "write",
		Short: "Writes a Story with no text to voice",
		RunE: func(_ *cobra.Command, args []string) error {
			log.Println("Starting to work on a new story...")

			suggestion := strings.Join(args, " ")
			s := buildStory(llm, suggestion)

			tmpDir := viper.GetString("STORYGEN_TMP_DIR")
			file, err := utils.SaveTextToFile(tmpDir, s.Title, "json", s.ToJson())
			if err != nil {
				return err
			}
			log.Println("JSON saved")
			log.Println(file)
			return nil
		},
	}
}

func newWorkCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Creates a Story",
		RunE: func(_ *cobra.Command, args []string) error {
			log.Println("Starting to work on a new story...")

			suggestion := strings.Join(args, " ")
			s := buildStory(llm, suggestion)

			tmpDir := viper.GetString("STORYGEN_TMP_DIR")
			file, err := utils.SaveTextToFile(tmpDir, s.Title, "json", s.ToJson())
			if err != nil {
				return err
			}
			log.Println("JSON saved")

			file, s = refineStory(llm, s, 0)

			toLang := strings.ToLower(viper.GetString("STORYGEN_LANGUAGE"))
			if toLang == "" {
				toLang = "english"
			}

			//_ = file
			chapter := story.TextChapter
			theEnd := story.TextTheEnd
			if toLang != "english" {
				title := s.Title
				s, chapter, theEnd = translate(llm, s, toLang)
				_, err = utils.SaveTextToFile(tmpDir, toLang+"_"+title, "json", s.ToJson())
				if err != nil {
					return err
				}
				log.Println(toLang, " JSON saved")
				file = toLang + "_" + file
			}
			ToVoice(s, file, s.BuildContent(chapter, theEnd))

			return err
		},
	}
}

func ToVoice(s story.Story, file, content string) {
	soundFile := file + ".mp3"
	lastDot := strings.LastIndex(file, ".")
	if lastDot >= 0 {
		soundFile = file[:lastDot] + ".mp3"
	}
	targetDir := strings.ToLower(viper.GetString("STORYGEN_TARGET_DIR"))
	log.Println("Text to Speech...")

	speed := viper.GetFloat64("STORYGEN_SPEECH_SPEED")
	if speed == 0 {
		speed = 0.9
	}

	voice := story.Voice{
		Provider: story.VoiceProvider{
			Provider: "openai",
			APIKey:   viper.GetString("OPENAI_API_KEY"),
			Model:    viper.GetString("STORYGEN_OPENAI_TTS_MODEL"),
			Voice:    viper.GetString("STORYGEN_VOICE"),
			Speed:    speed,
		},
		Instruction: story.VoiceInstruction{
			Affect:  viper.GetString("STORYGEN_VOICE_AFFECT"),
			Tone:    viper.GetString("STORYGEN_VOICE_TONE"),
			Pacing:  viper.GetString("STORYGEN_VOICE_PACING"),
			Emotion: viper.GetString("STORYGEN_VOICE_EMOTION"),
			Pauses:  viper.GetString("STORYGEN_VOICE_PAUSES"),
			Story:   s,
		},
	}

	postProcess := viper.GetBool("STORYGEN_TTS_POSTPROCESS")
	splitLen := viper.GetInt("STORYGEN_TTS_SPLITLEN")
	finalSoundFile, err := tts.TextToSpeech(targetDir, soundFile, content, voice, splitLen, postProcess)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Success!")
	log.Println("")
	log.Printf("Story: %s\n", s.Title)
	log.Printf("Summary: %s\n\n", s.Summary)
	log.Printf("json: %s\n", file)
	log.Printf("mp3: %s\n", finalSoundFile)
}

func translate(llm *ai.AI, s story.Story, toLang string) (story.Story, string, string) {
	translated := story.Story{}

	log.Printf("Translating Title %s ...\n", s.Title)
	translated.Title = llm.TranslateText(s.Title, toLang)
	log.Printf("Translated Title %q\n", translated.Title)

	for _, c := range s.Chapters {
		log.Printf("Translating Chapter %d - %s ...\n", c.Number, c.Title)
		translatedTitle := llm.TranslateSimpleText(c.Title, toLang)
		log.Printf("Translated Chapter Title %q\n", translatedTitle)
		translatedText := llm.TranslateText(c.Text, toLang)
		log.Printf("Translated Chapter Text %q\n", translatedText)

		translated.Chapters = append(translated.Chapters, story.Chapter{
			Number: c.Number,
			Title:  translatedTitle,
			Text:   translatedText,
		})
	}

	chapter := llm.TranslateSimpleText(story.TextChapter, toLang)
	log.Printf("Chapter is: %s\n", chapter)

	theEnd := llm.TranslateSimpleText(story.TextTheEnd, toLang)
	log.Printf("The End. is: %s\n", theEnd)

	log.Println("Translation Done")

	return translated, chapter, theEnd
}

func buildStory(llm *ai.AI, suggestion string) story.Story {
	s := story.NewStory()
	s.StorySuggestion = strings.Trim(suggestion, " ")
	s.Structure = story.GetRandomStoryStructure()

	chapterCount, maxChapterWords, lengthTxt := utils.GetChapterCountAndLength()
	s.Length = lengthTxt
	log.Printf("Length: %s", s.Length)
	log.Printf("Structure: %s", s.Structure.ToJson())

	if s.StorySuggestion != "" {
		log.Println("Time period...")
		s.TimePeriod = llm.FigureStoryTimePeriod(s)
	} else {
		s.TimePeriod = story.GetRandomTimePeriods(1)[0]
	}
	log.Printf("TimePeriod: %s\n", s.TimePeriod.ToJson())

	log.Println("Morales...")
	randomMoraleCount := viper.GetInt("STORYGEN_MORALE_COUNT")
	if randomMoraleCount == 0 {
		randomMoraleCount = rand.Intn(3) + 1
	}

	validMorales := llm.FigureStoryMorales(s)
	//validMorales := story.GetRandomMorales(3, story.GetAvailableStoryMorales())
	s.Morales = story.GetRandomMorales(randomMoraleCount, validMorales)

	picked := make([]string, len(s.Morales))
	for i, m := range s.Morales {
		picked[i] = m.Name
	}
	log.Printf("Picked Morales: %s", strings.Join(picked, ", "))

	log.Println("Protagonists...")
	s.Protagonists = llm.FigureStoryProtagonists(s)
	log.Printf("Protagonists (%d):\n - %s", len(s.Protagonists), s.Protagonists.String())

	log.Println("Villain...")
	s.Villain = llm.FigureStoryVillain(s)
	log.Printf("Villain: %s\n", s.Villain)

	s.VillainVoice = llm.FigureStoryVillainVoice(s)
	log.Printf("Voice: %s\n", s.VillainVoice)

	log.Println("Location...")
	s.Location = llm.FigureStoryLocation(s)
	log.Println("Plan...")
	s.Plan = llm.FigureStoryPlan(s)
	log.Println("Summary...")
	s.Summary = llm.FigureStorySummary(s)
	log.Println("Chapter Titles...")
	chapterTitles, err := llm.FigureStoryChapterTitles(s, chapterCount)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Built (%d) Chapters", len(chapterTitles))

	for i, title := range chapterTitles {
		number := i + 1
		s.Chapters = append(s.Chapters, story.Chapter{
			Number: number,
			Title:  title,
		})
	}

	chapterWords := utils.ChapterWordCount(chapterCount, maxChapterWords)
	for i, title := range chapterTitles {
		number := i + 1
		wordCount := chapterWords[number]
		log.Printf("Chapter %d - %s (words %d) ...\n", number, title, wordCount)
		chapterText := llm.FigureStoryChapter(s, number, title, wordCount)
		s.Chapters[i].Text = chapterText
	}

	log.Println("Story Title...")
	s.Title = llm.FigureStoryTitle(s)
	log.Printf("Picked title: %s\n", s.Title)

	return s
}

func compareStories(llm *ai.AI, storyAFile, storyBFile string) story.Story {
	storyA := &story.Story{}
	storyB := &story.Story{}
	json.Unmarshal(utils.LoadTextFromFile(storyAFile), storyA)
	json.Unmarshal(utils.LoadTextFromFile(storyBFile), storyB)

	log.Printf("StoryA: %q\n", storyA.Title)
	log.Printf("StoryB: %q\n", storyB.Title)

	return llm.CompareStories(*storyA, *storyB)
}
