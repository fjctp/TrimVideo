package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type programParams struct {
	pathToVlc  string
	pathToRoot string
	pathToOut  string
	maxWorker  int
}

type vlcParameters struct {
	pathToWorkDir string
	pathToInput   string
	pathToOutput  string
	timeToTrimSec int64
}

func runVlc(pathToVlc string, params vlcParameters) {

	// os.MkdirAll
	if _, errLookPath := exec.LookPath(pathToVlc); errLookPath != nil {
		log.Fatal("vlc is missing. Please install vlc before continue\n")
	}

	pathToOutputDir, _ := filepath.Split(params.pathToOutput)
	if _, errExist := os.Stat(pathToOutputDir); os.IsNotExist(errExist) {
		os.MkdirAll(pathToOutputDir, os.ModeDir)
	}

	cmdStr := []string{
		params.pathToInput,
		"--start-time", strconv.FormatInt(params.timeToTrimSec, 10),
		fmt.Sprintf("--sout=#file{dst=%s}", params.pathToOutput),
		"-Idummy",
		"vlc://quit"}

	log.Printf("Processing: %s -> %s\n", params.pathToInput, params.pathToOutput)

	cmd := exec.Command(pathToVlc, cmdStr...)
	cmd.Dir = params.pathToWorkDir
	// if errRun := cmd.Run(); errRun != nil {
	// 	log.Fatal(errRun)
	// }
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	fmt.Printf("combined out:\n%s\n", string(out))

}

func workers(pathToVlc string, jobs <-chan vlcParameters, wg *sync.WaitGroup) {
	defer wg.Done()

	log.Println("Worker started")
	for jp := range jobs {
		runVlc(pathToVlc, jp)
	}
	log.Println("Worker finished")
}

func addToFolderList(pathToRoot, pathToOut string,
	jobs chan<- vlcParameters) func(path string, info os.FileInfo, err error) error {

	return func(pathToFile string, info os.FileInfo, err error) error {

		relPathToFile, errRelPathToFile := filepath.Rel(pathToRoot, pathToFile)
		relPathToOutDir, errRelPathToOutDir := filepath.Rel(pathToRoot, pathToOut)
		relPathToOutFile := filepath.Join(relPathToOutDir, relPathToFile)

		if errRelPathToFile != nil || errRelPathToOutDir != nil {
			log.Fatal("Fatal: addToFolderList()") // To-doL implement this properly
		}

		switch {
		case err != nil:
			return err
		case info.IsDir():
			// skip folders
			return nil
		case filepath.Ext(info.Name()) != ".mp4":
			// skip NOT mp4 files
			return nil
		case strings.HasPrefix(relPathToFile, relPathToOutDir):
			// skip output folder
			return nil
		}

		if _, errExist := os.Stat(filepath.Join(pathToRoot, relPathToOutFile)); !os.IsNotExist(errExist) {
			// skip, if file already exist
			return nil
		}

		log.Printf("Added %s\n", relPathToFile)

		// remove special char from output
		fixedOutputName1 := strings.Replace(relPathToOutFile, "'", "", -1)
		fixedOutputName := strings.Replace(fixedOutputName1, ",", "", -1)
		vp := vlcParameters{
			pathToWorkDir: pathToRoot,
			pathToInput:   relPathToFile,
			pathToOutput:  fixedOutputName,
			timeToTrimSec: 7}
		jobs <- vp

		return nil
	}
}

func removeSpecialChar(in string) string{
	const spChar := [...]string{"'", ","};

	//fixedOutputName1 := strings.Replace(relPathToOutFile, "'", "", -1)
	//fixedOutputName := strings.Replace(fixedOutputName1, ",", "", -1)
}

func parseArgs() programParams {
	pparams := programParams{}

	flag.StringVar(&pparams.pathToVlc, "vlc", "C:\\Program Files (x86)\\VideoLAN\\VLC\\vlc.exe", "Where vlc is installed")
	flag.StringVar(&pparams.pathToRoot, "in", ".", "Process folder including subfolders")
	flag.StringVar(&pparams.pathToOut, "out", ".\\out", "Output folder")
	flag.IntVar(&pparams.maxWorker, "numWorker", 4, "Number of workers")

	flag.Parse()

	return pparams
}

func main() {

	args := parseArgs()

	jobs := make(chan vlcParameters, args.maxWorker*2)
	var wg sync.WaitGroup

	for i := 0; i < args.maxWorker; i++ {
		wg.Add(1)
		go workers(args.pathToVlc, jobs, &wg)
	}

	if errAddToFolderList := filepath.Walk(args.pathToRoot,
		addToFolderList(args.pathToRoot, args.pathToOut, jobs)); errAddToFolderList != nil {
		log.Fatal("Fatal: filepath.Walk()") // to-do: implement this properly
	}
	close(jobs)

	wg.Wait()

}
