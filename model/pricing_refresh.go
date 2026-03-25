package model

import "github.com/roseforljh/opencrab/constant"

func RefreshPricing() {}

func GetAllPricingFromCache() []struct {
	ModelName               string
	SupportedEndpointTypes  []constant.EndpointType
	EnableGroup             []string
	QuotaType               int
} {
	return []struct {
		ModelName              string
		SupportedEndpointTypes []constant.EndpointType
		EnableGroup            []string
		QuotaType              int
	}{}
}

func GetModelSupportEndpointTypes(modelName string) []constant.EndpointType {
	return []constant.EndpointType{}
}
