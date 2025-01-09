package utils

import (
	"fmt"
	"log"
	"time"

	"github.com/spf13/viper"
)

func ChapterWordCount(chapterCount, maxChapterWords int) map[int]int {
	chapterWords := make(map[int]int)
	for i := 0; i < chapterCount; i++ {
		number := i + 1
		wordCount := maxChapterWords

		// first chapter 80% shorter
		if number == 1 {
			wordCount = int(float64(maxChapterWords) * 0.8)
		}

		// last chapter 60% shorter
		if number == chapterCount {
			wordCount = int(float64(maxChapterWords) * 0.6)
		}
		chapterWords[number] = wordCount
	}
	return chapterWords
}

func GetChapterCountAndLength() (int, int, string) {
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
	lengthText := fmt.Sprintf(format, int(minutes.Minutes()), chapterCount, maxChapterWords)

	return chapterCount, maxChapterWords, lengthText
}
