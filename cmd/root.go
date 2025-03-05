package cmd

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/fauzanelka/99tech-order-processor/internal/processor"
)

var (
	// Flags
	inputFile  string
	outputFile string
	symbol     string
	side       string
	retries    int
	timeout    time.Duration
	insecure   bool
	verbose    bool
	baseURL    string

	// Logger
	logger = logrus.New()

	// Root command
	rootCmd = &cobra.Command{
		Use:   "order-processor",
		Short: "Process trading orders from a file",
		Long: `A CLI application that processes trading orders from a file.
It filters orders by symbol and side, then makes API requests for each matching order.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Configure logger
			if verbose {
				logger.SetLevel(logrus.DebugLevel)
			} else {
				logger.SetLevel(logrus.InfoLevel)
			}
			logger.SetOutput(os.Stdout)
			logger.SetFormatter(&logrus.TextFormatter{
				FullTimestamp: true,
			})

			logger.Infof("Starting order processor")
			logger.Infof("Input file: %s", inputFile)
			logger.Infof("Output file: %s", outputFile)
			logger.Infof("Filtering for symbol: %s, side: %s", symbol, side)
			logger.Infof("Retries: %d, Timeout: %s, Insecure: %v", retries, timeout, insecure)

			// Create and run processor
			proc := processor.NewProcessor(
				inputFile,
				outputFile,
				symbol,
				side,
				baseURL,
				retries,
				timeout,
				insecure,
				logger,
			)

			if err := proc.Process(); err != nil {
				logger.Fatalf("Processing failed: %v", err)
			}

			logger.Infof("Processing completed successfully")
		},
	}
)

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Define flags
	rootCmd.PersistentFlags().StringVar(&inputFile, "file", "transaction-log.txt", "Input file containing order data")
	rootCmd.PersistentFlags().StringVar(&outputFile, "output", "output.txt", "Output file for API responses")
	rootCmd.PersistentFlags().StringVar(&symbol, "symbol", "TSLA", "Symbol to filter orders by")
	rootCmd.PersistentFlags().StringVar(&side, "side", "sell", "Side to filter orders by (buy/sell)")
	rootCmd.PersistentFlags().IntVar(&retries, "retry", 3, "Number of retry attempts for failed requests")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 30*time.Second, "Timeout for HTTP requests")
	rootCmd.PersistentFlags().BoolVar(&insecure, "insecure", false, "Skip TLS verification")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&baseURL, "url", "https://example.com/api", "Base URL for the API")
} 