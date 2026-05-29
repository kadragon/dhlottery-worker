// Command worker is the GitHub Actions entry point: it validates the
// environment and runs the lottery workflow once.
package main

import (
	"os"
	"time"

	"github.com/kadragon/dhlottery-worker/internal/dhlottery"
	"github.com/kadragon/dhlottery-worker/internal/env"
	"github.com/kadragon/dhlottery-worker/internal/logger"
	"github.com/kadragon/dhlottery-worker/internal/workflow"
)

// Indirected for tests.
var (
	validateEnv = env.Validate
	runWorkflow = func(now time.Time) bool {
		return workflow.RunWorkflow(now, dhlottery.NewClient())
	}
	nowFn = time.Now
)

// run executes the workflow and returns the process exit code:
//
//	0 success, 2 workflow ran but notification failed, 1 fatal error.
func run() int {
	if err := validateEnv(); err != nil {
		logger.Error("Lottery workflow failed", logger.Fields{"error": err.Error()})
		return 1
	}

	logger.Info("Starting lottery workflow", nil)

	notified := runWorkflow(nowFn())

	if notified {
		logger.Info("Lottery workflow completed successfully", nil)
		return 0
	}
	logger.Error("Lottery workflow completed but Telegram notification failed", nil)
	return 2
}

func main() {
	os.Exit(run())
}
