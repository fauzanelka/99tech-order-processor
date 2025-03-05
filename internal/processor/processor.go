package processor

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/fauzanelka/99tech-order-processor/internal/models"
)

// Processor handles the processing of order data
type Processor struct {
	InputFile    string
	OutputFile   string
	Symbol       string
	Side         string
	Retries      int
	Timeout      time.Duration
	Insecure     bool
	BaseURL      string
	Logger       *logrus.Logger
	client       *http.Client
	outputWriter *os.File
}

// NewProcessor creates a new processor with the given configuration
func NewProcessor(inputFile, outputFile, symbol, side, baseURL string, retries int, timeout time.Duration, insecure bool, logger *logrus.Logger) *Processor {
	return &Processor{
		InputFile:  inputFile,
		OutputFile: outputFile,
		Symbol:     symbol,
		Side:       side,
		Retries:    retries,
		Timeout:    timeout,
		Insecure:   insecure,
		BaseURL:    baseURL,
		Logger:     logger,
	}
}

// Process reads the input file and processes each order
func (p *Processor) Process() error {
	// Setup HTTP client
	p.client = &http.Client{
		Timeout: p.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: p.Insecure,
			},
		},
	}

	// Open input file
	file, err := os.Open(p.InputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	// Open output file
	p.outputWriter, err = os.Create(p.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer p.outputWriter.Close()

	// Process file line by line
	scanner := bufio.NewScanner(file)
	lineNum := 0
	var retryQueue []models.Order

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse JSON
		var order models.Order
		if err := json.Unmarshal([]byte(line), &order); err != nil {
			p.Logger.Warnf("Line %d is not valid JSON: %v", lineNum, err)
			continue
		}

		// Filter by symbol and side
		if order.Symbol == p.Symbol && order.Side == p.Side {
			p.Logger.Infof("Processing order %s: %s %d %s at $%.2f", 
				order.OrderID, order.Side, order.Quantity, order.Symbol, order.Price)
			
			if err := p.processOrder(order, 0); err != nil {
				p.Logger.Warnf("Failed to process order %s, adding to retry queue: %v", order.OrderID, err)
				retryQueue = append(retryQueue, order)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input file: %w", err)
	}

	// Process retry queue
	p.processRetryQueue(retryQueue)

	return nil
}

// processOrder processes a single order with retries
func (p *Processor) processOrder(order models.Order, retryCount int) error {
	url := fmt.Sprintf("%s/%s", strings.TrimRight(p.BaseURL, "/"), order.OrderID)
	
	resp, err := p.client.Get(url)
	if err != nil {
		if retryCount < p.Retries {
			p.Logger.Debugf("Current retry count: %d, Max retries: %d", retryCount, p.Retries)
			p.Logger.Warnf("Request failed for order %s (retry %d/%d): %v", 
				order.OrderID, retryCount+1, p.Retries, err)
			time.Sleep(time.Second * time.Duration(retryCount+1)) // Exponential backoff
			return p.processOrder(order, retryCount+1)
		}
		return err
	}
	defer resp.Body.Close()

	// Check if response is successful (2XX)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("received non-2XX response: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Write response to output file
	if _, err := fmt.Fprintf(p.outputWriter, "%s\n", string(body)); err != nil {
		return fmt.Errorf("failed to write to output file: %w", err)
	}

	p.Logger.Infof("Successfully processed order %s", order.OrderID)
	return nil
}

// processRetryQueue processes the queue of failed orders
func (p *Processor) processRetryQueue(queue []models.Order) {
	if len(queue) == 0 {
		return
	}

	p.Logger.Infof("Processing retry queue with %d orders", len(queue))
	
	for _, order := range queue {
		retryAttempts := 0
		for retryAttempts < p.Retries {
			p.Logger.Infof("Retry attempt %d/%d for order %s", retryAttempts+1, p.Retries, order.OrderID)
			
			if err := p.processOrder(order, retryAttempts); err != nil {
				p.Logger.Warnf("Retry failed for order %s: %v", order.OrderID, err)
				retryAttempts++
				// Continue to next retry attempt
			} else {
				// Success, break out of retry loop
				break
			}
		}
		
		if retryAttempts >= p.Retries {
			p.Logger.Errorf("Exceeded maximum retries for order %s", order.OrderID)
		}
	}
} 