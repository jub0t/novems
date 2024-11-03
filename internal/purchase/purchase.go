package purchase

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

type PurchasePayload struct {
	ExpectedCurrency int `json:"expectedCurrency"`
	ExpectedSellerID int `json:"expectedSellerId"`
	ExpectedPrice    int `json:"expectedPrice"`
	UserAssetID      int `json:"userAssetId"`
}

type PurchaseResponse struct {
	Latency          time.Duration `json:"latency"`
	Purchased        bool          `json:"purchased"`
	Reason           string        `json:"reason"`
	ProductId        int           `json:"productId"`
	StatusCode       int           `json:"statusCode"`
	Title            string        `json:"title"`
	ErrorMsg         string        `json:"errorMsg"`
	ShowDivId        string        `json:"showDivId"`
	ShortfallPrice   int           `json:"shortfallPrice"`
	BalanceAfterSale int           `json:"balanceAfterSale"`
	ExpectedPrice    int           `json:"expectedPrice"`
	Currency         int           `json:"currency"`
	Price            int           `json:"price"`
	AssetId          int           `json:"assetId"`
}

// Preallocate buffer pools to reduce memory allocation overhead
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// Reusable http.Client with connection pool to enhance performance
var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// MakePurchase handles the actual purchase process
func MakePurchase(csrf, cookie string, productID, price, sellerID, userAssetID int) (*PurchaseResponse, error) {
	start := time.Now()

	// Reuse buffer from pool to reduce allocations
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	payload := PurchasePayload{
		ExpectedCurrency: 1,
		ExpectedPrice:    price,
		ExpectedSellerID: sellerID,
		UserAssetID:      userAssetID,
	}

	if err := json.NewEncoder(buf).Encode(payload); err != nil {
		return nil, fmt.Errorf("error encoding purchase payload: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://economy.roblox.com/v1/purchases/products/%d", productID), buf)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers and cookie
	req.AddCookie(&http.Cookie{Name: ".ROBLOSECURITY", Value: cookie})
	req.Header.Set("content-type", "application/json; charset=utf-8")
	req.Header.Set("x-csrf-token", csrf)

	// Execute the request
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing purchase request: %v", err)
	}
	defer resp.Body.Close()

	// Measure latency
	latency := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("purchase failed with status code: %d", resp.StatusCode)
	}

	// Optimize response reading
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// Reuse memory from sync pool if needed for unmarshaling
	var purchaseResponse PurchaseResponse
	purchaseResponse.Latency = latency
	if err := json.Unmarshal(respBody, &purchaseResponse); err != nil {
		return nil, fmt.Errorf("error unmarshaling response body: %w", err)
	}

	return &purchaseResponse, nil
}
