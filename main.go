package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type OnboardRequest struct {
	CRNumber       string      `json:"cr_number"`
	VATNumber      string      `json:"vat_number"`
	Industry       string      `json:"industry"`
	MonthlyRevenue interface{} `json:"monthly_revenue"`
}

type EligibilityRequest struct {
	CRNumber string  `json:"cr_number"`
	Revenue  float64 `json:"revenue"`
}

type EligibilityResponse struct {
	Eligible      bool    `json:"eligible"`
	MaxLoanAmount float64 `json:"max_loan_amount"`
}

type OnboardResponse struct {
	Status        string  `json:"status"`
	Eligible      bool    `json:"eligible"`
	MaxLoanAmount float64 `json:"max_loan_amount"`
}

func main() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"POST", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))

	r.POST("/api/v1/onboard", func(c *gin.Context) {
		var req OnboardRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			logError(fmt.Sprintf("Failed to parse onboard request: %v", err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		logInfo(fmt.Sprintf("Received onboard request: cr_number=%s, vat_number=%s, industry=%s, monthly_revenue=%v",
			req.CRNumber, req.VATNumber, req.Industry, req.MonthlyRevenue))

		// Parse monthly_revenue to float64
		var monthlyRevenue float64
		switch v := req.MonthlyRevenue.(type) {
		case float64:
			monthlyRevenue = v
		case string:
			parsed, err := strconv.ParseFloat(v, 64)
			if err != nil {
				logError(fmt.Sprintf("Unable to convert monthly_revenue '%s' to number, defaulting to 0", v))
			} else {
				monthlyRevenue = parsed
			}
		default:
			logError(fmt.Sprintf("Unexpected type for monthly_revenue: %T, defaulting to 0", v))
		}

		// Call loan_service for eligibility check
		eligReq := EligibilityRequest{
			CRNumber: req.CRNumber,
			Revenue:  monthlyRevenue,
		}

		body, _ := json.Marshal(eligReq)
		logInfo(fmt.Sprintf("Calling loan_service eligibility with payload: %s", string(body)))

		resp, err := http.Post(
			"http://localhost:3002/api/v1/eligibility",
			"application/json",
			bytes.NewBuffer(body),
		)
		if err != nil {
			logError(fmt.Sprintf("Failed to call loan_service: %v", err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check eligibility"})
			return
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(resp.Body)
		logInfo(fmt.Sprintf("loan_service response: %s", string(respBody)))

		var eligResp EligibilityResponse
		json.Unmarshal(respBody, &eligResp)

		status := "approved"
		if !eligResp.Eligible {
			status = "rejected"
		}

		logInfo(fmt.Sprintf("Onboard result: cr_number=%s, status=%s, eligible=%v, max_loan_amount=%.2f",
			req.CRNumber, status, eligResp.Eligible, eligResp.MaxLoanAmount))

		c.JSON(http.StatusOK, OnboardResponse{
			Status:        status,
			Eligible:      eligResp.Eligible,
			MaxLoanAmount: eligResp.MaxLoanAmount,
		})
	})

	logInfo("onboarding_be starting on :3001")
	r.Run(":3001")
}
