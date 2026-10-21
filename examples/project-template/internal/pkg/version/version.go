package version

import (
	"fmt"
	"runtime"
)

var (
	version   = "v1.0.0"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func Print() {
	fmt.Printf("Server Version: %s\n", version)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Git Commit: %s\n", gitCommit)
	fmt.Printf("Build Time: %s\n", buildTime)
	fmt.Println()
}
