package scraper

import (
	"fmt"
  "bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"sniper/internal/csrf"
	"sniper/internal/parser"

	"github.com/goccy/go-json"
	"github.com/gocolly/colly"
)

var (
	Client = &http.Client{} // Initialize the HTTP client
)

type LimitedAssetResponse struct {
	Id        int `json:"id"`
	ProductID int `json:"productId"`
	Price     int `json:"lowestPrice"`
	SellerID  int `json:"sellerId"`
}

type ItemData struct {
	Id       string `json:"id"`
	ItemType string `json:"itemType"`
}

type LimitedResponse struct {
	Id        int    `json:"id"`
	ProductId uint64 `json:"productId"`
	Price     int    `json:"lowestPrice"`
}

type RequestBody struct {
	Items []ItemData `json:"items"`
}

type ResponseBody struct {
	Data []LimitedAssetResponse `json:"data"`
}

// Written By github.com/jub0t
// Reuse the client, ensuring connection reuse and minimizing overhead.
func FasterItemDetails(cookie, limited_id string, proxy *parser.SingleProxy) (LimitedAssetResponse, error) {
	var response LimitedAssetResponse

	// Build the proxy URL
	proxyURL := fmt.Sprintf("http://%s:%s", proxy.IP, proxy.Port)
	proxyParsedURL, err := url.Parse(proxyURL)
	if err != nil {
		return response, fmt.Errorf("invalid proxy URL: %w", err)
	}

	// Create a custom transport with proxy support
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyParsedURL),
	}

	var client http.Client

	if len(proxy.Port) > 0 {
		client = http.Client{
			Transport: transport,
		}
	} else {
		client = http.Client{}
	}

	req, err := http.NewRequest("GET",
		fmt.Sprintf("https://catalog.roblox.com/v1/catalog/items/%s/details?itemType=Asset", limited_id),
		nil, // Use http.NoBody or nil for GET requests
	)
	if err != nil {
		return response, fmt.Errorf("could not create request: %w", err)
	}

	// Set headers and cookies
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", `"Chromium";v="128", "Not;A=Brand";v="24", "Brave";v="128"`)

	// Execute the request and measure latency
	resp, err := client.Do(req)
	if err != nil {
		return response, fmt.Errorf("catalog response failure: %w", err)
	}
	defer resp.Body.Close() // Always ensure the response body is closed

	// Read the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	if err := json.Unmarshal(respBody, &response); err != nil {
		return response, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

type AuthenticatedUser struct {
	Username    string `json:"name"`
	DisplayName string `json:"DisplayName"`
	Id          int    `json:"id"`
}

func FetchAuthenticated(cookie string) (AuthenticatedUser, error) {
	var response AuthenticatedUser
	url := "https://users.roblox.com/v1/users/authenticated"

	req, err := http.NewRequest("GET",
		url,
		http.NoBody,
	)
	if err != nil {
		return response, fmt.Errorf("could not create request: %w", err)
	}

	// Set headers and cookies only once.
	req.Header.Set("X-Csrf-Token", csrf.Token)
	req.AddCookie(&http.Cookie{
		Name:  ".ROBLOSECURITY",
		Value: cookie,
	})

	// Execute the request and measure latency
	resp, err := Client.Do(req)
	if err != nil {
		return response, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close() // Always ensure the response body is closed

	// Read the response body
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return response, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	if err := json.Unmarshal(respBody, &response); err != nil {
		return response, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

type ScrapedDetails struct {
	ProductID   int `json:"productId"`
	Price       int `json:"lowestPrice"`
	SellerID    int `json:"sellerId"`
	UserAssetID int `json:"userAssetId"`
}

func ScrapeItemDetails(cookie, limitedID string) (ScrapedDetails, error) {
	var ret ScrapedDetails
  collector := colly.NewCollector()

	// Set cookie once, reused across requests
	err := collector.SetCookies("https://www.roblox.com/", []*http.Cookie{
		{Name: ".ROBLOSECURITY", Value: cookie},
	})
	if err != nil {
		return ret, fmt.Errorf("Could not set cookies: %w", err)
	}

	// Handle the response directly from byte slice to avoid unnecessary conversions
	collector.OnResponse(func(r *colly.Response) {
		ret.Price, ret.ProductID, ret.SellerID, ret.UserAssetID = parser.ParseItemDetails(r.Body)
	})

	err = collector.Visit(fmt.Sprintf("https://www.roblox.com/catalog/%s/", limitedID))
	if err != nil {
		return ret, fmt.Errorf("Error visiting the catalog URL: %w", err)
	}

	return ret, nil
}

// Struct to capture the response format
type ThumbnailData struct {
	RequestId    string `json:"requestId"`
	ErrorCode    int    `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
	TargetId     int64  `json:"targetId"`
	State        string `json:"state"`
	ImageUrl     string `json:"imageUrl"`
	Version      string `json:"version"`
}

// Struct to capture the full response with a data field
type ThumbnailResponse struct {
	Data []ThumbnailData `json:"data"`
}

func GetThumbnail(targetId string) (ThumbnailData, error) {
	url := "https://thumbnails.roblox.com/v1/batch"

	// Data struct for the POST request, with targetId as a parameter
	data := []map[string]interface{}{
		{
			"requestId": fmt.Sprintf("%s:undefined:Asset:150x150:webp:regular", targetId),
			"type":      "Asset",
			"targetId":  targetId,
			"format":    "webp",
			"size":      "150x150",
		},
	}

	// Convert the struct to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return ThumbnailData{}, fmt.Errorf("error marshalling JSON: %v", err)
	}

	// Create a new POST request with the appropriate headers
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return ThumbnailData{}, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Perform the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ThumbnailData{}, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ThumbnailData{}, fmt.Errorf("error reading response body: %v", err)
	}

	// Unmarshal the response into the ThumbnailResponse struct
	var thumbnailResp ThumbnailResponse
	err = json.Unmarshal(body, &thumbnailResp)
	if err != nil {
		return ThumbnailData{}, fmt.Errorf("error unmarshalling JSON response: %v", err)
	}

	// Return the first element of the Data array, if it exists
	if len(thumbnailResp.Data) > 0 {
		return thumbnailResp.Data[0], nil
	}

	return ThumbnailData{}, fmt.Errorf("no data found in the response")
}


