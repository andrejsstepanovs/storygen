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
)

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
		newLoadCommand(llm),
	)

	return cmd, nil
}

func newLoadCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "load",
		Short: "Load a Story",
		RunE: func(_ *cobra.Command, _ []string) error {
			file := "Title_Bruno_and_the_Shadow_Beast_A_Tale_of_Courage_and_Light.json"

			s := &story.Story{}
			json.Unmarshal(utils.LoadTextFromFile(file), s)

			text := s.BuildContent()
			soundFile := file[:len(file)-4] + "mp3"
			//err := tts.DeepgramTextToSpeech(soundFile, text)
			//err := tts.OpenAITextToSpeech(openai.VoiceAlloy, soundFile, text)
			err := tts.OpenAITextToSpeech(openai.VoiceShimmer, soundFile, text)

			return err
		},
	}
}

func newWorkCommand(llm *ai.AI) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Creates a Story",
		RunE: func(_ *cobra.Command, args []string) error {
			log.Println("Starting work with redmine")

			s := story.NewStory()
			s.StorySuggestion = strings.Join(args, " ")
			s.Protagonists = story.GetRandomProtagonists(1)
			s.Structure = story.GetRandomStoryStructure()
			s.TimePeriod = story.GetRandomTimePeriods(1)[0]

			const readSpeedWordsInMinute = 180

			minutes := time.Minute * 6
			chapterCount := int(minutes.Minutes() / 2)

			s.Length = fmt.Sprintf("%d minutes to read", int(minutes.Minutes()))

			s.Structure = story.Structure{
				Name:        "Action adventure with animals",
				Description: "A story with a lot of action and adventure, with animals as the main characters.",
			}

			log.Printf("Length: %s", s.Length)
			log.Printf("Structure: %s", s.Structure.ToJson())
			log.Printf("Protagonists: %s", s.Protagonists[0].ToJson())
			log.Printf("TimePeriod: %s", s.TimePeriod.ToJson())

			log.Println("Morales...")
			//validMorales := llm.FigureStoryMorales(s, story.GetAvailableStoryMorales())
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

			for i, title := range chapterTitles {
				number := i + 1
				log.Printf("Chapter %d - %s ...\n", number, title)
				chapterText := llm.FigureStoryChapter(s, number, title, maxChapterWords)
				s.Chapters[i].Text = chapterText
			}

			log.Println("Story Title...")
			s.Title = llm.FigureStoryTitle(s)
			log.Println("Success")

			file, err := utils.SaveTextToFile(s.Title, "json", s.ToJson())
			if err != nil {
				log.Println("Failed to save story")
				return err
			}
			log.Println(s.ToJson())

			text := s.BuildContent()
			soundFile := file[:len(file)-4] + "mp3"
			//err := tts.DeepgramTextToSpeech(soundFile, text)
			//err := tts.OpenAITextToSpeech(openai.VoiceAlloy, soundFile, text)
			err = tts.OpenAITextToSpeech(openai.VoiceShimmer, soundFile, text)

			return err
		},
	}
}
