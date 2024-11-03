package worker

import (
	"fmt"
	"net/http"
	"sniper/internal/config"
	"sniper/internal/csrf"
	"sniper/internal/parser"
	"sniper/internal/purchase"
	"sniper/internal/scraper"
	"sniper/internal/webhook"
	"sync"
	"time"

	"github.com/charmbracelet/log"
)

type LastCheck struct {
	TimeTaken time.Duration
}

var (
	Client          = &http.Client{} // Initialize the HTTP client
	PurchasedItems  = sync.Map{}
	FailedItems     = sync.Map{}
	IterationChecks = sync.Map{}
	InQueue         = sync.Map{}
	Mutex           sync.Mutex
	wg              sync.WaitGroup // WaitGroup for managing webhook completion
)

func worker(
	quit <-chan struct{},
	config *config.ConfigStruct,
	limited parser.LimitedInfo,
) {
	for {
		select {
		case <-quit:
			return
		default:
			log.Debug("Fetching Limited Information For Record")
			first_info, limited_error := scraper.ScrapeItemDetails(config.Cookie, limited.Id)
			if limited_error != nil {
				log.Error(limited_error)
				return
			}

			if first_info.ProductID == 0 || first_info.Price < 0 {
				log.Warn("Could not fetch data for worker to start, skipping.", "Limited ID", limited.Id)
				return
			}

			log.Info("Worker Activated", "Id", limited.Id, "Price", first_info.Price)

			iteration_count := 1
			for {
				go func() {
					if config.Verbose {
						if check, ok := IterationChecks.Load(limited.Id); ok {
							log.Info("[Interval Pass]:", "Get Info Latency:", check.(LastCheck).TimeTaken, "Iteration Count", iteration_count)
						}
						if _, ok := FailedItems.Load(limited.Id); ok {
							log.Info("Item Has Failed Before. Waiting One Second", "Limited ID", limited.Id)
							time.Sleep(time.Second * 1)
						}
					}

					start := time.Now()

					if in_queue, _ := InQueue.LoadOrStore(limited.Id, false); in_queue.(bool) {
						if config.Verbose {
							log.Warn("Limited is in the process of being sniped, Continuing Loop.")
						}
						time.Sleep(time.Millisecond * time.Duration(config.Rate))
						return
					}

					info, err := scraper.ScrapeItemDetails(config.Cookie, limited.Id)
					if err != nil {
						log.Error(err)
						time.Sleep(time.Millisecond * time.Duration(config.Rate))
						return
					}

					if info.Price <= limited.Price || (info.Price != 0 || info.Price != -1) {
						InQueue.Store(limited.Id, true)
						if config.Verbose {
							log.Warn("Lower Than Expected Price Detected.", "Limited ID", limited.Id)
						}

						purchase_response, purchase_error := purchase.MakePurchase(csrf.Token, config.Cookie, info.ProductID, info.Price, first_info.SellerID, first_info.UserAssetID)

						var thumbnail_url string
						thumbnail, err := scraper.GetThumbnail(limited.Id)
						if err != nil {
							fmt.Println("Error:", err)
						} else {
							fmt.Println(thumbnail)
							thumbnail_url = thumbnail.ImageUrl
						}

						if purchase_error == nil {
							InQueue.Store(limited.Id, false)
							if purchase_response.Purchased {
								PurchasedItems.Store(limited.Id, struct{}{})

								embed := webhook.Embed{
									Title: "Limited Snipe Success",
									Description: fmt.Sprintf("Item Purchase: `%s`\nSeller ID: `%d`\nLatency: `%v`",
										limited.Id,
										info.SellerID,
										purchase_response.Latency,
									),
									Color: 0xF58A42,
									Thumbnail: webhook.EmbedThumbnail{
										URL: thumbnail_url,
									},
								}

								wg.Add(1) // Increment the wait group counter
								go func() {
									defer wg.Done() // Decrement the wait group counter
									webhook.SendWebhook(config.WebhookURL, embed)
								}()

								// Wait until the webhook sending is complete
								wg.Wait()

								log.Warn("Sniped Successfully Executed", "Message", purchase_response.ErrorMsg)
								return
							} else {
								FailedItems.Store(limited.Id, struct{}{})
								embed := webhook.Embed{
									Title: "Purchase Failure",
									Description: fmt.Sprintf("Limited ID: `%s`\nLatency: `%v`\nMessage: `%s`",
										limited.Id,
										purchase_response.Latency,
										purchase_response.ErrorMsg,
									),
									Color: 0x8115ed,
									Thumbnail: webhook.EmbedThumbnail{
										URL: thumbnail_url,
									},
								}

								wg.Add(1) // Increment the wait group counter
								go func() {
									defer wg.Done() // Decrement the wait group counter
									webhook.SendWebhook(config.WebhookURL, embed)
								}()

								// Wait until the webhook sending is complete
								wg.Wait()

								log.Warn("Purchase Failure", "Message", purchase_response.ErrorMsg)
							}
						} else {
							FailedItems.Store(limited.Id, struct{}{})
							embed := webhook.Embed{
								Title: "Error",
								Description: fmt.Sprintf("Limited Item ID: `%s`\nLatency: `%v`\nMessage: `%v`",
									limited.Id,
									purchase_response.Latency,
									purchase_error,
								),
								Color: 0xd11197,
								Thumbnail: webhook.EmbedThumbnail{
									URL: thumbnail_url,
								},
							}

							wg.Add(1) // Increment the wait group counter
							go func() {
								defer wg.Done() // Decrement the wait group counter
								webhook.SendWebhook(config.WebhookURL, embed)
							}()

							// Wait until the webhook sending is complete
							wg.Wait()
						}

						if config.Verbose {
							log.Info(fmt.Sprintf("Sniping Limited: Price: %d. Actual: %d", limited.Price, info.Price))
						}
					}

					iteration_check := time.Since(start)
					IterationChecks.Store(limited.Id, LastCheck{TimeTaken: iteration_check})
					iteration_count++

				}()
				time.Sleep(time.Millisecond * time.Duration(config.Rate))
			}
		}
	}
}

func Run(config *config.ConfigStruct, limiteds []parser.LimitedInfo) {
	quit := make(chan struct{})
	for i := 0; i < len(limiteds); i++ {
		go func() {
			limited := limiteds[i]
			worker(quit, config, limited)
		}()
	}

	for {
		time.Sleep(time.Hour)
	}
}
