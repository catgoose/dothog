package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

func main() {
	fmt.Println("🚀 Starting Go Test Watcher...")
	fmt.Println("Tests will run automatically on .go file changes")
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("Failed to create watcher:", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Printf("Error closing watcher: %v", err)
		}
	}()

	// Channel to handle shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	// Channel to debounce test runs
	testChan := make(chan bool, 1)

	// Start test runner goroutine
	go func() {
		for range testChan {
			runTests()
		}
	}()

	// Watch current directory and subdirectories
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip certain directories
			if shouldSkipDir(path) {
				return filepath.SkipDir
			}
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		log.Fatal("Failed to walk directory:", err)
	}

	// Run initial tests
	testChan <- true

	// Watch for events
	for {
		select {
		case event := <-watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				if strings.HasSuffix(event.Name, ".go") {
					// Debounce test runs
					select {
					case testChan <- true:
					default:
						// Channel is full, skip this run
					}
				}
			}
		case err := <-watcher.Errors:
			log.Println("Watcher error:", err)
		case <-done:
			fmt.Println("\n🛑 Shutting down test watcher...")
			return
		}
	}
}

func runTests() {
	fmt.Println("🔄 Running tests...")
	fmt.Println("==================================")

	cmd := exec.Command("go", "test", "-v", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	fmt.Println("==================================")
	if err != nil {
		fmt.Printf("❌ Tests failed: %v\n", err)
	} else {
		fmt.Printf("✅ Tests passed at %s\n", time.Now().Format("15:04:05"))
	}
	fmt.Println()
}

func shouldSkipDir(path string) bool {
	skipDirs := []string{
		".git",
		"vendor",
		"node_modules",
		"tmp",
		"build",
		"bin",
		"web/assets/public",
		"web/views",
	}

	for _, skip := range skipDirs {
		if strings.Contains(path, skip) {
			return true
		}
	}
	return false
}
