package rajaongkir

import (
	"encoding/json"
	"go-digital-library-fiber/pkg/clients"
	"go-digital-library-fiber/pkg/constant"
	"go-digital-library-fiber/pkg/utils"
	"net/http"
)

const BASE_URL = "https://rajaongkir.komerce.id/api/v1"

type Service struct {
	client clients.HTTPClient
}

func InitService() Service {
	return Service{
		client: clients.InitHTTPClient(BASE_URL, 10, utils.GetConfig(constant.RAJAONGKIR_API_KEY)),
	}
}

func (r *Service) GetDeliveryFee(req GetFeeRequest) (float64, error) {
	var response FeeResponse

	payload := map[string]string{
		"origin":      req.Origin,
		"destination": req.Destination,
		"weight":      "1",
		"courier":     req.Courier,
	}

	res, err := r.client.SendFormEncoded(
		"/calculate/district/domestic-cost",
		http.MethodPost,
		payload,
	)

	if err != nil {
		return 0, err
	}

	if err := json.Unmarshal([]byte(res), &response); err != nil {
		return 0, err
	}

	fee := float64(response.Data[0].Cost)

	return fee, nil
}
