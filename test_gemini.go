package main

import (
	"context"
	"fmt"
	"github.com/shiva/ai-match/pkg/logger"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

func main() {
	godotenv.Load(".env")
	apiKey := os.Getenv("GEMINI_API_KEY")
	
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		logger.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	models := []string{"gemini-1.5-flash", "gemini-1.0-pro", "gemini-pro", "gemini-1.5-pro"}
	
	for _, m := range models {
		fmt.Printf("--- Testing model: %s ---\n", m)
		model := client.GenerativeModel(m)
		resp, err := model.GenerateContent(ctx, genai.Text("Say hi"))
		if err != nil {
			fmt.Println("❌ Error:", err)
		} else {
			fmt.Println("✅ Success!")
			_ = resp
		}
	}
}
