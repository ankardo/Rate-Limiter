package e2e

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

func TestRateLimiterE2E(t *testing.T) {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})
	defer redisClient.Close()

	client := &http.Client{}

	baseURL := getEnv("BASE_URL", "http://rate-limiter:8080")
	maxRequests, _ := strconv.Atoi(getEnv("MAX_REQUESTS_PER_SECOND", "5"))
	tokenMaxRequests, _ := strconv.Atoi(getEnv("TOKEN_MAX_REQUESTS", "10"))
	blockDuration, _ := strconv.Atoi(getEnv("BLOCK_DURATION_SECONDS", "5"))

	tests := []struct {
		name         string
		url          string
		queryParam   string
		paramValue   string
		iterations   int
		expectStatus []int
	}{
		{
			name:         "Limit by IP",
			url:          "/ip",
			queryParam:   "ip",
			paramValue:   "192.168.1.1:12345",
			iterations:   maxRequests + 1,
			expectStatus: generateExpectedStatus(maxRequests),
		},
		{
			name:         "Limit by Token",
			url:          "/token",
			queryParam:   "token",
			paramValue:   "valid-token",
			iterations:   tokenMaxRequests + 1,
			expectStatus: generateExpectedStatus(tokenMaxRequests),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < tt.iterations; i++ {
				req, _ := http.NewRequest("GET", baseURL+tt.url+"?"+tt.queryParam+"="+tt.paramValue, nil)

				res, err := client.Do(req)
				if err != nil {
					t.Fatalf("Request %d failed: %v", i+1, err)
				}
				defer res.Body.Close()

				validateResponse(t, i+1, res.StatusCode, tt.expectStatus[i])

				if res.StatusCode == http.StatusOK {
					var response map[string]string
					err = json.NewDecoder(res.Body).Decode(&response)
					if err != nil {
						t.Fatalf("Failed to decode JSON response: %v", err)
					}
					if response[tt.queryParam] != tt.paramValue {
						t.Fatalf("Iteration %d failed: expected %s to be %s, got %s", i+1, tt.queryParam, tt.paramValue, response[tt.queryParam])
					}
				}
			}

			time.Sleep(time.Duration(blockDuration+1) * time.Second)

			req, _ := http.NewRequest("GET", baseURL+tt.url+"?"+tt.queryParam+"="+tt.paramValue, nil)
			res, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request after block duration failed: %v", err)
			}
			defer res.Body.Close()

			validateResponse(t, "after block duration", res.StatusCode, http.StatusOK)
		})
	}
}

func validateResponse(t *testing.T, iteration interface{}, actual, expected int) {
	if actual != expected {
		t.Fatalf("Iteration %v failed: expected %d, got %d", iteration, expected, actual)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func generateExpectedStatus(limit int) []int {
	statuses := make([]int, limit+1)
	for i := 0; i < limit; i++ {
		statuses[i] = http.StatusOK
	}
	statuses[limit] = http.StatusTooManyRequests
	return statuses
}
