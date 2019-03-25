package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
)

func replConfirmation(text string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(text)
	inputText, _ := reader.ReadString('\n')
	if inputText != "y\n" {
		fmt.Println("Stopping...")
		os.Exit(1)
	}
}

func clamp(value int64, min int64, max int64) int64 {
	return int64(math.Min(math.Max(float64(value), float64(min)), float64(max)))
}
