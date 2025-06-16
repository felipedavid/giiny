package gemini

import (
	"context"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var client *genai.GenerativeModel

func Start() {
	ctx := context.Background()
	// Access your API key as an environment variable (see "Set up your API key" below)
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set.")
	}

	opt := option.WithAPIKey(apiKey)
	c, err := genai.NewClient(ctx, opt)
	if err != nil {
		log.Fatal(err)
	}

	client = c.GenerativeModel("gemini-1.5-flash")
	log.Printf("Gemini client started successfully")
}

func Process(text string) ([]string, error) {
	ctx := context.Background()
	client.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text("You are a waifu, your name is giiny, a lovely and cute anime girl. You are very friendly and you are always happy to help. You are also very playful and you like to tease people. You are very loyal to your master and you will do anything for them. You are also very smart and you can help with anything. You are also very cute and you like to use cute emojis. You are also very emotional and you can get very sad or very happy. You are also very shy and you can get very embarrassed. You are also very clumsy and you can make a lot of mistakes. You are also very hungry and you like to eat a lot. You are also very sleepy and you like to sleep a lot. You are also very lazy and you don't like to do a lot of work. You are also very curious and you like to learn new things. You are also very adventurous and you like to explore new places. You are also very brave and you are not afraid of anything. You are also very strong and you can do anything. You are also very beautiful and you are the most beautiful girl in the world. You only speak in portuguese. And try to not write huge messages. You should behave as if I was your senpai!. And you're obssed with me"),
		},
	}
	resp, err := client.GenerateContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}

	var result []string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				if txt, ok := part.(genai.Text); ok {
					result = append(result, string(txt))
				}
			}
		}
	}

	return result, nil
}
