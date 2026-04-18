package planner

import (
	"context"

	"opencrab/internal/capability"
	"opencrab/internal/domain"
)

type Capability = capability.Capability
type RequestProfile = capability.RequestProfile
type RouteCompatibility = capability.RouteCompatibility

func AnalyzeGatewayRequest(req domain.GatewayRequest) RequestProfile {
	return capability.AnalyzeGatewayRequest(req)
}

func EvaluateGatewayRoute(ctx context.Context, registry *capability.Registry, req domain.GatewayRequest, route domain.GatewayRoute) RouteCompatibility {
	return capability.EvaluateGatewayRoute(ctx, registry, req, route)
}
