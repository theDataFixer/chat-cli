package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"
)

// Constants for emojis used in the chat interface.
const (
	// Person emoji used for user input.
	personEmoji = "󰙊 "
	// Robot emoji used for assistant output.
	robotEmoji = "󰚩 "
)

// Global variables used throughout the application.
var (
	// Flag to enable verbose output.
	verbose bool
	// Map of valid OpenAI models and their corresponding environment variable names.
	validModels = map[string]string{
		"llama-instant": "llama-3.1-8b-instant",
		"llama-70b":     "llama3-70b-8192",
		"mixtral":       "mixtral-8x7b-32768",
	}
)

// Function to clear the console screen based on the operating system.
func clearScreen() {
	// Use the 'clear' or 'cls' command depending on the OS.
	switch runtime.GOOS {
	case "windows":
		// Create a new process to run the 'cls' command.
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		// Create a new process to run the 'clear' command.
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}

// Root command for the chat-cli application.
var rootCmd = &cobra.Command{
	// Command usage string.
	Use: "chat-cli",
	// Short description of the command.
	Short: "A CLI chat application using Groq",
	// Run function to handle the command execution.
	Run: func(cmd *cobra.Command, args []string) {
		// Call the chat function to start the chat interface.
		chat()
	},
}

// Initialize the root command with a flag for verbose output.
func init() {
	// Add a flag to enable verbose output.
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// Function to execute the root command.
func Execute() {
	// Check if there's an error executing the command.
	if err := rootCmd.Execute(); err != nil {
		// Print the error message.
		fmt.Println(err)
		// Exit the application with a non-zero status code.
		os.Exit(1)
	}
}

// Function to handle the chat interface.
func chat() {
	// Get the API key from the environment variable.
	apiKey := os.Getenv("GROQ_API_KEY")
	// Check if the API key is set.
	if apiKey == "" {
		// Print an error message using color.
		color.Red("Error: GROQ_API_KEY environment variable not set")
		return
	}

	// Get the selected model from the environment variable.
	modelEnv := os.Getenv("GROQ_MODEL")
	// Default to the 'llama-instant' model if the environment variable is not set.
	selectedModel := validModels[modelEnv]
	if selectedModel == "" {
		selectedModel = validModels["llama-instant"]
	}

	// Create an OpenAI client with the API key and selected model.
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://api.groq.com/openai/v1"
	client := openai.NewClientWithConfig(config)

	// Create a new scanner to read input from the console.
	scanner := bufio.NewScanner(os.Stdin)
	// Set the buffer size to 64KB.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Create colors for user input, assistant output, and metrics.
	userColor := color.New(color.FgHiGreen)
	assistantColor := color.New(color.FgHiMagenta)
	metricsColor := color.New(color.FgHiYellow)

	// Create printf functions for user input and assistant output.
	userPrompt := userColor.PrintfFunc()
	assistantPrompt := assistantColor.PrintfFunc()

	// Clear the console screen.
	clearScreen()
	// Print a welcome message with the selected model.
	fmt.Printf("Chat started using model: %s (type 'exit' to quit)\n", selectedModel)
	// Print instructions for the user.
	fmt.Println("Type 'clear' to clear the screen")
	fmt.Println("For multiline input, type 'paste' and press Enter")

	// Initialize the chat messages.
	messages := []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "Provide helpful and concise responses",
		},
	}

	// Loop indefinitely to handle user input and assistant output.
	for {
		// Print a prompt for user input.
		userPrompt("\n%s You: ", personEmoji)
		// Read the user input from the scanner.
		scanner.Scan()
		text := scanner.Text()

		// Check if the user wants to exit the chat interface.
		if text == "exit" {
			break
		}

		// Check if the user wants to clear the console screen.
		if text == "clear" {
			// Clear the console screen.
			clearScreen()
			continue
		}

		// Check if the user wants to paste a multiline input.
		if text == "paste" {
			// Print a prompt for the user to enter the text.
			userPrompt("Enter your text (type 'done' on a new line when finished):\n")
			// Create a new builder to store the pasted text.
			var builder strings.Builder
			// Loop until the user types 'done'.
			for scanner.Scan() {
				line := scanner.Text()
				if line == "done" {
					break
				}
				builder.WriteString(line + "\n")
			}
			// Get the pasted text and strip any trailing whitespace.
			text = strings.TrimSpace(builder.String())
		}

		// Check if the user input is empty.
		if text == "" {
			continue
		}

		// Add the user input to the chat messages.
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: text,
		})

		// Get the start time to measure the response time.
		start := time.Now()

		// Create a new chat completion request with the selected model and chat messages.
		req := openai.ChatCompletionRequest{
			Model:       selectedModel,
			Messages:    messages,
			Temperature: 0.7,
			Stream:      true,
		}

		// Create a new chat completion stream with the client and request.
		stream, err := client.CreateChatCompletionStream(context.Background(), req)
		// Check if there's an error creating the stream.
		if err != nil {
			// Print an error message using color.
			color.Red("Error creating stream: %v\n", err)
			continue
		}

		// Print a prompt for the assistant output.
		assistantPrompt("%s Assistant: ", robotEmoji)
		// Create a new builder to store the assistant output.
		var fullResponse strings.Builder

		// Loop indefinitely to receive the assistant output.
		for {
			// Get the response from the stream.
			response, err := stream.Recv()
			// Check if there's an error receiving the response.
			if err != nil {
				// Check if the error is due to an EOF (end-of-file).
				if strings.Contains(err.Error(), "EOF") {
					break
				}
				// Print an error message using color.
				color.Red("Error receiving response: %v\n", err)
				break
			}

			// Get the assistant output from the response.
			content := response.Choices[0].Delta.Content
			// Print the assistant output using the printf function.
			assistantPrompt("%s", content)
			// Add the assistant output to the builder.
			fullResponse.WriteString(content)
		}

		// Close the stream.
		stream.Close()
		// Print a newline character.
		fmt.Println()

		// Get the elapsed time since the start of the request.
		elapsed := time.Since(start)

		// Add the assistant output to the chat messages.
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "assistant",
			Content: fullResponse.String(),
		})

		// Check if verbose output is enabled.
		if verbose {
			// Get the input and output token counts.
			inputTokens := len(strings.Split(text, " "))
			outputTokens := len(strings.Split(fullResponse.String(), " "))
			totalTokens := inputTokens + outputTokens

			// Calculate the tokens per second.
			tokensPerSecond := float64(totalTokens) / elapsed.Seconds()

			// Print the metrics using the printf function.
			metricsColor.Println("\nMetrics:")
			metricsColor.Printf("Time taken: %.2f seconds\n", elapsed.Seconds())
			metricsColor.Printf("Speed: %.2f tokens/second\n", tokensPerSecond)
			metricsColor.Printf("Input tokens: %d\n", inputTokens)
			metricsColor.Printf("Output tokens: %d\n", outputTokens)
			metricsColor.Printf("Total tokens: %d\n", totalTokens)
		}
	}
}

// Main function to execute the root command.
func main() {
	// Call the Execute function to run the root command.
	Execute()
}
