package services

import (
	"context"
	"narrabyte/internal/llm/client"
	"narrabyte/internal/utils"
	"os"
)

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

	result, err := s.OpenAIClient.ExploreCodebaseDemo(s.context, root)
	if err != nil {
		return "", err
	}

	return result, nil
}
