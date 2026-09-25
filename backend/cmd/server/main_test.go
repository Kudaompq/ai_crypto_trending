package main

import (
	"net/http"
	"net/http/httptest"
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
