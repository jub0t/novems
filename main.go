package main

import (
	"fmt"
	"os"
	"sniper/internal/config"
	"sniper/internal/csrf"
	"sniper/internal/parser"
	"sniper/internal/scraper"
	"sniper/internal/worker"
	"time"

	"github.com/charmbracelet/log"
	"github.com/urfave/cli/v2"
	"golang.org/x/time/rate"
)

func main() {
	app := &cli.App{
		Name:  "Limited Sniper v1.0.0 By jub0t",
		Usage: "A fast application to snipe Roblox limited items for a desired price.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "file",
				Usage: "Path to the main file with Limited information, format is <price>,<id>",
			},
		},
		Action: func(ctx *cli.Context) error {
			// proxy_path := ctx.String("proxy")
			// if len(proxy_path) < 1 {
			//	log.Error("Provide a valid proxy file using --proxy <path>.")
			//	return nil
			// }

			file_path := ctx.String("file")
			if len(file_path) < 1 {
				log.Error("Provide a valid limited info file using --file <path>.")
				return nil
			}

			// Initialize CSRF
			csrf.Init(5 * time.Second)

			// Read and Setup Configuration File.
			cfg, config_error := config.LoadConfig("config.yaml")
			if config_error != nil {
				log.Error("Please verify that a config.yaml file exists, exiting.")
				return nil
			} else {
				log.Info("🔧 Configuration Has Been Loaded")
			}

			limiteds, lim_error := parser.FromFile(file_path)
			if lim_error != nil {
				log.Error(lim_error)
				// log.Error("Add a 'ids.csv' file to the root directory with appropriate limited id and price.")
				return nil
			} else {
				log.Info("🛒 Limiteds Have Been Loaded", "Limiteds", limiteds)
			}

			// proxies, prox_error := parser.ParseProxies(proxy_path)
			// if prox_error != nil {
			// log.Error(prox_error)
			//	return nil
			// } else {
			//	log.Info("🌐 Proxies Successfuly Obtained", "Proxy Count", len(proxies))
			// }

			// Fetch CSRF
			csrf_error := csrf.UpdateCSRF(cfg.Cookie)
			if csrf_error != nil {
				log.Errorf("Something went wrong while fetching a CSRF token. %s", csrf_error)
			} else {
				log.Info("🪙 Successfuly Retreived CSRF-Token", "Token", csrf.Token)
			}

			bot_info, bot_err := scraper.FetchAuthenticated(cfg.Cookie)
			if bot_err != nil {
				log.Error("Error Occured While Bot Information", bot_err)
			} else {
        
if (bot_info.Id <= 0) {

        log.Error("Authentication Failed, Re-try with a valid Cookie.")
        return nil
      }

				log.Info(fmt.Sprintf("Authentiated As %s(%d).", bot_info.Username, bot_info.Id))
			}

			// Initialize a new rate limiter
			limiter := rate.NewLimiter(rate.Limit(cfg.Rate), cfg.Rate)

			// Start workers
			worker.Run(
				cfg,
				limiter,
				limiteds,
			)

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
