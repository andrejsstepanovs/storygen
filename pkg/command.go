package pkg

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"time"

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
	)

	return cmd, nil
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
			log.Println("Pre reading...")

			preReadLoops := viper.GetInt("STORYGEN_PREREAD_LOOPS")
			if preReadLoops == 0 {
				preReadLoops = 3
			}
			for i := 1; i <= preReadLoops; i++ {
				log.Printf("Pre-reading / story fixing loop %d...\n", i)
				text := s.BuildContent(story.TextChapter, story.TextTheEnd)
				problems := llm.FigureStoryLogicalProblems(text)
				if len(problems) == 0 {
					log.Println("Story is OK")
					break
				}
				log.Printf("Found problems: %d\n", len(problems))

				allSuggestions := make(story.Suggestions, 0)
				for _, problem := range problems {
					log.Printf("Finding suggestions how to fix chapter %d...", problem.Chapter)
					for _, c := range s.Chapters {
						if c.Number == problem.Chapter {
							log.Printf("Suggesting fix suggestions for: %d. %s...", problem.Chapter, problem.ChapterName)
							suggestions := llm.SuggestStoryFixes(s, problem)
							log.Printf("Suggestions: %d", len(suggestions))
							for _, sug := range suggestions {
								allSuggestions = append(allSuggestions, sug)
							}
						}
					}
				}

				log.Printf("Found problems: %d\n", len(problems))
				log.Printf("Total Suggestions %d...", len(allSuggestions))
				chapterSuggestions := make(map[int]story.Suggestions)
				for _, sug := range allSuggestions {
					chapterSuggestions[sug.Chapter] = append(chapterSuggestions[sug.Chapter], sug)
				}
				for chapter, suggestions := range chapterSuggestions {
					log.Printf("Chapter %d has %d suggestions", chapter, len(suggestions))
					for _, sug := range suggestions {
						log.Printf("- %s\n", sug.Suggestions)
					}
				}

				sort.Slice(allSuggestions, func(i, j int) bool {
					return allSuggestions[i].Chapter < allSuggestions[j].Chapter
				})

				log.Println("Fixing...")
				for chapter, suggestions := range chapterSuggestions {
					for _, problem := range problems {
						if problem.Chapter == chapter {
							for j, c := range s.Chapters {
								if c.Number == problem.Chapter {
									log.Printf("Adjusting chapter %d. %q with %d suggestions...", problem.Chapter, problem.ChapterName, len(suggestions))
									fixedChapter := llm.AdjustStoryChapter(s, problem, suggestions)
									s.Chapters[j].Text = fixedChapter
									break
								}
							}
							break
						}
					}
				}
			}

			log.Println("Done")
			file, err := utils.SaveTextToFile("groomed_"+s.Title, "json", s.ToJson())
			log.Println(file)
			return err
		},
	}
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

			text := translated.BuildContent(chapter, theEnd)
			soundFile := file[:len(file)-4] + "mp3"

			file = toLang + "_" + file
			ToVoice(translated, file, text)
			err := tts.TextToSpeech(openai.VoiceShimmer, soundFile, text, inbetweenChaptersFile)

			return err
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

			file, err := utils.SaveTextToFile(s.Title, "json", s.ToJson())
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

			file, err := utils.SaveTextToFile(s.Title, "json", s.ToJson())
			if err != nil {
				return err
			}
			log.Println("JSON saved")

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
				_, err = utils.SaveTextToFile(toLang+"_"+title, "json", s.ToJson())
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

	log.Println("Text to Speech...")
	err := tts.TextToSpeech(openai.VoiceShimmer, soundFile, content, inbetweenChaptersFile)
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

	readSpeedWordsInMinute := viper.GetInt("STORYGEN_READSPEED")
	if readSpeedWordsInMinute == 0 {
		log.Fatalln("Please set the STORYGEN_READSPEED environment variable")
	}

	lengthInMin := viper.GetInt64("STORYGEN_LENGTH_IN_MIN")
	if lengthInMin == 0 {
		lengthInMin = 8
	}
	minutes := time.Minute * time.Duration(lengthInMin)
	log.Printf("Approximate length: %d min\n", int(minutes.Minutes()))

	chapterCount := viper.GetInt("STORYGEN_CHAPTERS")
	if chapterCount == 0 {
		chapterCount = int(minutes.Minutes() / 1.6)
	}
	if chapterCount < 3 {
		chapterCount = 3
	}
	log.Printf("Chapter count: %d\n", chapterCount)

	maxChapterWords := (readSpeedWordsInMinute * int(minutes.Minutes())) / chapterCount

	format := "Full story reading time: %d minutes. Chapter count: %d. Longest chapter: %d words."
	s.Length = fmt.Sprintf(format, int(minutes.Minutes()), chapterCount, maxChapterWords)
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

	for i, title := range chapterTitles {
		number := i + 1
		wordCount := maxChapterWords

		// first chapter 80% shorter
		if number == 1 {
			wordCount = int(float64(maxChapterWords) * 0.8)
		}

		// last chapter 60% shorter
		if number == len(chapterTitles) {
			wordCount = int(float64(maxChapterWords) * 0.6)
		}

		log.Printf("Chapter %d - %s (words %d) ...\n", number, title, wordCount)
		chapterText := llm.FigureStoryChapter(s, number, title, wordCount)
		s.Chapters[i].Text = chapterText
	}

	log.Println("Story Title...")
	s.Title = llm.FigureStoryTitle(s)
	log.Printf("Picked title: %s\n", s.Title)

	return s
}
