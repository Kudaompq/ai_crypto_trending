package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProductionRouterOpportunityEndpointIsRemoved(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodGet, "/api/opportunities?symbol=ETH%2FUSDT", nil)
	response := httptest.NewRecorder()

	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("GET /api/opportunities returned %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestProductionRouterRegistersMarketAnalysisChatEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodPost, "/api/analysis/chat", strings.NewReader("{"))
	response := httptest.NewRecorder()

	newRouter().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /api/analysis/chat returned %d, want malformed request status %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}
