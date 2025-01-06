package pkg

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
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

	llm, err := ai.NewAI()
	if err != nil {
		return nil, err
	}

	cmd.AddCommand(
		newWorkCommand(llm),
		newTranslateCommand(llm),
	)

	return cmd, nil
}

func newTranslateCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "translate",
		Short: "Load a Story from JSON (first arg) and translate to language (second param)",
		RunE: func(_ *cobra.Command, args []string) error {
			file := args[0]
			log.Printf("Loading story from file: %s", file)

			s := &story.Story{}
			json.Unmarshal(utils.LoadTextFromFile(file), s)

			translated := *s
			chapter := "Chapter"
			toLang := "english"
			if len(args) == 2 {
				toLang = args[1]
				log.Printf("Translating to: %s", toLang)
				translated, chapter = translate(llm, *s, toLang)
			}

			text := translated.BuildContent(chapter)
			soundFile := file[:len(file)-4] + "mp3"

			log.Println("Text to Speech...")
			file = toLang + "_" + file
			ToVoice(translated, file, text)
			err := tts.TextToSpeech(openai.VoiceShimmer, soundFile, text, inbetweenChaptersFile)

			return err
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

			chapter := "Chapter"
			if toLang != "english" {
				title := s.Title
				s, chapter = translate(llm, s, toLang)
				_, err = utils.SaveTextToFile(toLang+"_"+title, "json", s.ToJson())
				if err != nil {
					return err
				}
				log.Println(toLang, " JSON saved")
				file = toLang + "_" + file
			}

			ToVoice(s, file, s.BuildContent(chapter))

			return err
		},
	}
}

func ToVoice(s story.Story, file, content string) {
	soundFile := file[:len(file)-4] + "mp3"

	fmt.Println(content)

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

func translate(llm *ai.AI, s story.Story, toLang string) (story.Story, string) {
	translated := story.Story{}

	log.Printf("Translating Title %s ...\n", s.Title)
	translated.Title = llm.TranslateText(s.Title, toLang)
	log.Printf("Translated Title %s ...\n", translated.Title)

	for _, c := range s.Chapters {
		log.Printf("Translating Chapter %d - %s ...\n", c.Number, c.Title)
		translated.Chapters = append(translated.Chapters, story.Chapter{
			Number: c.Number,
			Title:  llm.TranslateText(c.Title, toLang),
			Text:   llm.TranslateText(c.Text, toLang),
		})
	}

	chapter := llm.TranslateText("Chapter", toLang)
	log.Printf("Chapter is: %s\n", chapter)

	log.Println("Translation Done")

	return translated, chapter
}

func buildStory(llm *ai.AI, suggestion string) story.Story {
	s := story.NewStory()
	s.StorySuggestion = suggestion
	s.Protagonists = story.GetRandomProtagonists(1)
	s.Structure = story.GetRandomStoryStructure()
	s.TimePeriod = story.GetRandomTimePeriods(1)[0]

	readSpeedWordsInMinute := viper.GetInt("STORYGEN_READSPEED")
	if readSpeedWordsInMinute == 0 {
		log.Fatalln("Please set the STORYGEN_READSPEED environment variable")
	}

	lengthInMin := viper.GetInt64("STORYGEN_LENGTH_IN_MIN")
	if lengthInMin == 0 {
		lengthInMin = 8
	}
	minutes := time.Minute * time.Duration(lengthInMin)
	chapterCount := int(minutes.Minutes() / 2)
	if chapterCount < 3 {
		chapterCount = 3
	}

	s.Length = fmt.Sprintf("%d minutes to read", int(minutes.Minutes()))

	log.Printf("Length: %s", s.Length)
	log.Printf("Structure: %s", s.Structure.ToJson())
	log.Printf("Protagonists: %s", s.Protagonists[0].ToJson())
	log.Printf("TimePeriod: %s", s.TimePeriod.ToJson())

	log.Println("Morales...")
	//validMorales := llm.FigureStoryMorales(s, story.GetAvailableStoryMorales()) // results in predictable stories all about courage
	validMorales := story.GetRandomMorales(3, story.GetAvailableStoryMorales())

	randomMoraleCount := rand.Intn(3) + 1
	s.Morales = story.GetRandomMorales(randomMoraleCount, validMorales)

	picked := make([]string, len(s.Morales))
	for i, m := range s.Morales {
		picked[i] = m.Name
	}
	log.Printf("Picked Morales: %s", strings.Join(picked, ", "))

	log.Println("Villain...")
	s.Villain = llm.FigureStoryVillain(s)
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

	maxChapterWords := (readSpeedWordsInMinute * int(minutes.Minutes())) / len(chapterTitles)
	log.Printf("Default chapter words: %d\n", maxChapterWords)

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
