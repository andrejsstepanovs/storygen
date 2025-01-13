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
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const inbetweenChaptersFile = ""

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
		newStoryCompareCommand(llm, audience),
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

            storyIdeas := llm.FigureStoryIdeas(l, audience)
            log.Printf("Ideas: %d\n", len(storyIdeas))
            for _, idea := range storyIdeas {
                log.Printf("%s\n", idea)
            }
			return nil
		},
	}
}

func newStoryCompareCommand(llm *ai.AI, audience string) *cobra.Command {
	return &cobra.Command{
		Use:   "compare",
		Short: "Compare two stories. First param is path to one json file, second is path to another json file",
		RunE: func(_ *cobra.Command, args []string) error {
		    storyAFile := args[0]
		    storyBFile := args[1]
            log.Printf("%q, %q\n", storyAFile, storyBFile)

			storyA := &story.Story{}
			storyB := &story.Story{}
			json.Unmarshal(utils.LoadTextFromFile(storyAFile), storyA)
			json.Unmarshal(utils.LoadTextFromFile(storyBFile), storyB)

            log.Printf("StoryA: %q\n", storyA.Title)
            log.Printf("StoryB: %q\n", storyB.Title)

            betterStory := llm.CompareStories(*storyA, *storyB)
            log.Printf("Story: %q is better\n", betterStory.Title)

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
	if preReadLoops == 0 {
		preReadLoops = viper.GetInt("STORYGEN_PREREAD_LOOPS")
		if preReadLoops == 0 {
			preReadLoops = 3
		}
	}

	chapterCount, maxChapterWords, _ := utils.GetChapterCountAndLength()
	chapterWords := utils.ChapterWordCount(chapterCount, maxChapterWords)
    tmpDir := viper.GetString("STORYGEN_TMP_DIR")

	allAddressedSuggestions := make(story.Suggestions, 0)
	for i := 1; i <= preReadLoops; i++ {
		log.Printf("Pre-reading / story fixing loop: %d...\n", i)
		text := s.BuildContent(story.TextChapter, story.TextTheEnd)

    	//utils.SaveTextToFile(tmpDir, strconv.Itoa(i)+"_groomed_text_"+s.Title, "txt", text)

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

		allSuggestions := make(story.Suggestions, 0)
		for _, problem := range problems {
			log.Printf("Finding suggestions how to fix chapter %d...", problem.Chapter)
			for _, c := range s.Chapters {
				if c.Number == problem.Chapter {
					log.Printf("Suggesting fix suggestions for: %d. %s...", problem.Chapter, problem.ChapterName)
					suggestions := llm.SuggestStoryFixes(s, problem, allAddressedSuggestions)
					log.Printf("Suggestions: %d", len(suggestions))
					for _, sug := range suggestions {
						allSuggestions = append(allSuggestions, sug)
					}
				}
			}
		}

		log.Printf("Found problems: %d\n", len(problems))
		chapterSuggestions := make(map[int]story.Suggestions)
		for _, sug := range allSuggestions {
			chapterSuggestions[sug.Chapter] = append(chapterSuggestions[sug.Chapter], sug)
		}
		totalSuggestions := make([]string, 0)
		for _, suggestions := range chapterSuggestions {
			w := make([]string, 0)
			for _, sug := range suggestions {
				for _, k := range sug.Suggestions {
					w = append(w, k)
					totalSuggestions = append(totalSuggestions, k)
				}
			}
			//log.Printf("Chapter %d suggestions (%d):", chapter, len(w))
			//for _, txt := range w {
			//	log.Printf(" - %s", txt)
			//}
		}
		log.Printf("# Total Suggestions Points: %d", len(totalSuggestions))

		// sort chapterSuggestions by key
		keys := make([]int, 0, len(chapterSuggestions))
		for k := range chapterSuggestions {
			keys = append(keys, k)
		}
		sort.Ints(keys)

		log.Println("Fixing...")
		for _, chapter := range keys {
			suggestions := chapterSuggestions[chapter]
			for _, problem := range problems {
				if problem.Chapter == chapter {
					for j, c := range s.Chapters {
						if c.Number == problem.Chapter {
							log.Printf("Adjusting chapter %d. %q with %d suggestions (%d)...", problem.Chapter, problem.ChapterName, len(suggestions), suggestions.Count())
							wordCount := chapterWords[problem.Chapter]
							fixedChapter := llm.AdjustStoryChapter(s, problem, suggestions, allAddressedSuggestions, wordCount)
							s.Chapters[j].Text = fixedChapter
							break
						}
					}
					break
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
			toLang := viper.GetString("STORYGEN_LANGUAGE")
			if toLang != "english" {
				log.Printf("Translating to: %s", toLang)
				translated, chapter, theEnd = translate(llm, *s, toLang)
			}

			soundFile := file[:len(file)-4] + "mp3"
			ToVoice(translated, toLang + "_" + soundFile, translated.BuildContent(chapter, theEnd))

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

			toLang := viper.GetString("STORYGEN_LANGUAGE")
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
	soundFile := file[:len(file)-4] + "mp3"
	toLang := viper.GetString("STORYGEN_TARGET_DIR")
	log.Println("Text to Speech...")
	err := tts.TextToSpeech(openai.VoiceShimmer, toLang, soundFile, content, inbetweenChaptersFile)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Success!")
	log.Println("")
	log.Printf("Story: %s\n", s.Title)
	log.Printf("Summary: %s\n\n", s.Summary)
	log.Printf("json: %s\n", file)
	log.Printf("mp3: %s\n", soundFile)
}

func translate(llm *ai.AI, s story.Story, toLang string) (story.Story, string, string) {
	translated := story.Story{}

	log.Printf("Translating Title %s ...\n", s.Title)
	translated.Title = llm.TranslateText(s.Title, toLang)
	log.Printf("Translated Title %s ...\n", translated.Title)

	for _, c := range s.Chapters {
		log.Printf("Translating Chapter %d - %s ...\n", c.Number, c.Title)
		translated.Chapters = append(translated.Chapters, story.Chapter{
			Number: c.Number,
			Title:  llm.TranslateSimpleText(c.Title, toLang),
			Text:   llm.TranslateText(c.Text, toLang),
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
	protagonists := make([]string, 0)
	for _, p := range s.Protagonists {
		protagonists = append(protagonists, fmt.Sprintf("%s %s %s %s", p.Size, p.Age, p.Gender, p.Type))
	}
	log.Printf("Protagonists (%d):\n - %s", len(s.Protagonists), strings.Join(protagonists, "\n - "))

	log.Println("Villain...")
	s.Villain = llm.FigureStoryVillain(s)
	log.Printf("%s\n", s.Villain)

	log.Println("Location...")
	s.Location = llm.FigureStoryLocation(s)
	log.Println("Plan...")
	s.Plan = llm.FigureStoryPlan(s)
	log.Println("Summary...")
	s.Summary = llm.FigureStorySummary(s)
	log.Println("Chapter Titles...")
	chapterTitles := llm.FigureStoryChapterTitles(s, chapterCount)
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
