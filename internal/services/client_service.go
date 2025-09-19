package services

import (
	"context"
	"narrabyte/internal/llm/client"
	"narrabyte/internal/utils"
	"os"
)

// On pourrait lowkey rendre ca plus generique pour n'importe quel client
// Interface pour clients?
type ClientService struct {
	OpenAIClient client.OpenAIClient
	context      context.Context
}

func (s *ClientService) Startup(ctx context.Context) error {
	s.context = ctx

	err := utils.LoadEnv()
	if err != nil {
		return err
	}
	key := os.Getenv("OPENAI_API_KEY")

	temp, err := client.NewOpenAIClient(ctx, key)

	if err != nil {
		return err
	}

	s.OpenAIClient = *temp

	return nil
}

func NewClientService() *ClientService {
	return &ClientService{}
}

func (s *ClientService) ExploreDemo() (string, error) {
	root, err := utils.FindProjectRoot()
	if err != nil {
		return "", err
	}

	ctx := s.OpenAIClient.StartStream(s.context)
	defer s.OpenAIClient.StopStream()

	result, err := s.OpenAIClient.ExploreCodebaseDemo(ctx, root)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (s *ClientService) StopStream() {
	s.OpenAIClient.StopStream()
}
