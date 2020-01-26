package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/jamesnetherton/m3u"
	"github.com/kennygrant/sanitize"
)

const DEBUG = false

func getCommand(streamURL, fileName string) string {
	return fmt.Sprintf("ffmpeg ", streamURL, fileName)
}

func runCommand(command string, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if DEBUG {
			fmt.Println(cmd.String() + " : " + fmt.Sprint(err) + " : " + stderr.String())
		}
		return err
	}
	return nil
}

func takeScreenshot(streamURL, fileName string) error {
	return runCommand("ffmpeg", []string{
		"-i",
		streamURL,
		"-ss",
		"00:00:01.500",
		"-f",
		"image2",
		"-vframes",
		"1",
		fileName,
	})
}

func main() {
	var err error
	var fileName string
	playlist, err := m3u.Parse(os.Args[1])
	if err != nil {
		panic(err)
	}
	for i, track := range playlist.Tracks {
		fmt.Printf("[%d/%d] URI: %s Name: %s", i+1, len(playlist.Tracks), track.URI, track.Name)
		fileName = sanitize.BaseName(fmt.Sprintf("%s-%d-%s", time.Now().Format(time.RFC3339), i, track.Name)) + ".png"
		err = takeScreenshot(track.URI, fileName)
		if err != nil {
			fmt.Println(" ERROR")
			continue
		}
		fmt.Print(" Screenshot saved to: " + fileName)
		fmt.Println("")
	}
}
