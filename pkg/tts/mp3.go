package tts

import (
	"io"
	"log"


	"github.com/hyacinthus/mp3join"
)

func Remove(files []string) error {
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			log.Printf("Error deleting the file %s", file)
			return err
		}
	}
	return nil
}

func JoinMp3Files(files []string, output string, inbetweenFile string) error {
	joiner := mp3join.New()

	for i, file := range files {
		log.Printf("Processing %d file %s\n", i, file)
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		err = joiner.Append(f)
		if err != nil {
			return err
		}

		if inbetweenFile != "" && i < len(files)-1 {
			log.Println("Adding inbetween file")
			f, err := os.Open(inbetweenFile)
			if err != nil {
				return err
			}
			defer f.Close()

			err = joiner.Append(f)
			if err != nil {
				return err
			}
		}
	}

	log.Println("Joining")
	dest := joiner.Reader()

	outFile, err := os.Create(output)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, dest)
	if err != nil {
		return err
	}

	return nil
}
