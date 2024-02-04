package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	ld "github.com/launchdarkly/go-server-sdk/v7"
	"github.com/launchdarkly/go-server-sdk/v7/interfaces/flagstate"
)

func currentDir() string {
	pwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	return pwd
}

func readFile(f string, ffs map[string]ldvalue.Value, wg *sync.WaitGroup) {
	defer wg.Done()

	wg.Add(1)

	file, err := os.Open(f)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	line := 0

	for scanner.Scan() {
		line++

		for k := range ffs {
			r, _ := regexp.Compile("('|\")" + k + "('|\")")

			if found := r.MatchString(scanner.Text()); found {
				slog.Info(fmt.Sprintf("File: %s Found: %t Line: %d FF: %s", f, found, line, k))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	root := os.Getenv("PATH_TO_SCAN")
	var files []string

	filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if !info.Mode().IsRegular() {
			return nil
		}

		files = append(files, path)

		return nil
	})

	client, err := ld.MakeClient(os.Getenv("LD_SDK"), 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	defer client.Close()

	context := ldcontext.NewBuilder("context-key-123abc").Name("Sandy").Build()
	state := client.AllFlagsState(context, flagstate.OptionClientSideOnly())

	var wg sync.WaitGroup

	for _, v := range files {
		go readFile(v, state.ToValuesMap(), &wg)
	}

	wg.Wait()
}
