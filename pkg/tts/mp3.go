package tts

import (
	"io"
	"os"

	"github.com/hyacinthus/mp3join"
)

func Remove(files []string) error {
	for _, file := range files {
		err := os.Remove(file)
		if err != nil {
			return err
		}
	}
    return nil
}

func JoinMp3Files(files []string, output string) error {
	joiner := mp3join.New()

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		err = joiner.Append(f)
		if err != nil {
			return err
		}
	}

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

    defer Remove(files)

	return nil
}
