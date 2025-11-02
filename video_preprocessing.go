package main

import (
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	// Create a new string for the output filePath
	outputFilePath := fmt.Sprintf("%s.processing", filePath)

	// Create the ffmpeg command
	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputFilePath)

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Ffmpeg command failed: %v", err)
	}

	// Return the output filePath
	return outputFilePath, nil
}
