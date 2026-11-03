package ai

import (
	"encoding/json"
	"fmt"
	"go-digital-library-fiber/pkg/clients"
	"go-digital-library-fiber/pkg/constant"
	"go-digital-library-fiber/pkg/utils"
	"net/http"
)

const BASE_URL = "https://openrouter.ai/api/v1"

type Service interface {
	GetBookRecommendation(req BookRecommendationRequest) (PromptResponse, error)
}

type service struct {
	client clients.HTTPClient
}

var _ Service = (*service)(nil)

func InitService() Service {
	return &service{
		client: clients.InitHTTPClient(BASE_URL, 40, utils.GetConfig(constant.AI_API_KEY)),
	}
}

func (r *service) GetBookRecommendation(req BookRecommendationRequest) (PromptResponse, error) {
	var response PromptResponse

	var model string = utils.GetConfig(constant.AI_MODEL)
	var userPrompt string = fmt.Sprintf("Suggest top %v book recommendations about %v", req.Quantity, req.Topic)

	payload := map[string]any{
		"model": model,
		"messages": []Message{
			{
				Role:    "system",
				Content: SYSTEM_PROMPT,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	res, err := r.client.SendJSON(
		"/chat/completions",
		http.MethodPost,
		payload,
	)

	if err != nil {
		return PromptResponse{}, err
	}

	if err := json.Unmarshal([]byte(res), &response); err != nil {
		return PromptResponse{}, err
	}

	return response, nil
}
