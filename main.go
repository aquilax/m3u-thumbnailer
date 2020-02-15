package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/aquilax/m3u"
	"github.com/kennygrant/sanitize"
)

const DEBUG = false
const TIMEOUT_SEC = 5 * time.Second

func getCommand(streamURL, fileName string) string {
	return fmt.Sprintf("ffmpeg ", streamURL, fileName)
}

func runCommand(command string, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), TIMEOUT_SEC)
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

func getJob(track m3u.Track, i, tracks int, success chan<- m3u.Track) func() {
	return func() {
		var b bytes.Buffer
		fmt.Fprintf(&b, "[%d/%d] URI: %s Name: %s", i+1, tracks, track.URI, track.Name)
		fileName := sanitize.BaseName(fmt.Sprintf("%s-%s", track.Name, time.Now().Format(time.RFC3339))) + ".jpg"
		err := takeScreenshot(track.URI, fileName)
		if err != nil {
			fmt.Fprint(&b, " ERROR")
			fmt.Println(b.String())
			return
		}
		fmt.Fprint(&b, " Screenshot saved to: "+fileName)
		fmt.Println(b.String())
		track.Tags = append(track.Tags, m3u.Tag{"screenshot", fileName})
		success <- track
	}
}

func main() {
	var err error
	if len(os.Args) < 2 {
		fmt.Printf(`Usage:
	%s PLAYLIST`+"\n", os.Args[0])
		return
	}
	playlist, err := m3u.Parse(os.Args[1])
	success := m3u.Playlist{}
	if err != nil {
		panic(err)
	}

	results := make(chan m3u.Track)
	var workers = runtime.NumCPU()
	if workers > 1 {
		workers--
	}
	wp := workerpool.New(workers)
	tracks := len(playlist.Tracks)
	for i := range playlist.Tracks {
		track := playlist.Tracks[i]
		wp.Submit(getJob(track, i, tracks, results))
	}

	go func() {
		for {
			track := <-results
			success.Tracks = append(success.Tracks, track)
		}
	}()

	wp.StopWait()
	close(results)

	reader, err := m3u.Marshall(success)
	if err != nil {
		panic(err)
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile("succeess.m3u", data, 0644)
	if err != nil {
		panic(err)
	}
}
