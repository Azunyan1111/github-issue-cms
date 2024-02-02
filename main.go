package main

import (
	"fmt"
	"github.com/Azunyan1111/github-issue-cms/internal/config"
	"time"

	"github.com/Azunyan1111/github-issue-cms/cmd"
)

func main() {
	// Measure the time it takes to run the program
	startTime := time.Now()
	defer func() {
		config.Logger.Info(fmt.Sprintf("Finished in %f seconds\n", time.Since(startTime).Seconds()))
	}()

	// Execute the root command
	cmd.Execute()
}
