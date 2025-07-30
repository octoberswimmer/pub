package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	headers       []string
	transform     string
	requestMethod string
)

var rootCmd = &cobra.Command{
	Use:   "pub <URL expression>",
	Short: "Read JSON from stdin, transform it, and send HTTP requests",
	Long: `pub reads JSON lines from stdin, transforms them using expressions,
and sends HTTP requests to the specified URL.

Example:
  force pubsub subscribe /event/Fax_Classification_Job_Update__e | pub --transform '{data: input}' --header '"Authorization: Bearer " + env.EVENTS_PUBLISH_TOKEN' --request POST '"http://localhost:8080/publish?queue=" + input.eFax_Test_Queue'`,
	Args: cobra.ExactArgs(1),
	Run:  run,
}

func init() {
	rootCmd.Flags().StringArrayVar(&headers, "header", []string{}, "Add header (can be used multiple times)")
	rootCmd.Flags().StringVar(&transform, "transform", "", "Transform expression to apply to input")
	rootCmd.Flags().StringVar(&requestMethod, "request", "POST", "HTTP request method")
}

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) {
	urlExpr := args[0]

	scanner := bufio.NewScanner(os.Stdin)
	client := &http.Client{}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		if err := processLine(line, urlExpr, client); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing line: %v\n", err)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}
}

func processLine(line string, urlExpr string, client *http.Client) error {
	var input interface{}
	if err := json.Unmarshal([]byte(line), &input); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	env := map[string]interface{}{
		"input": input,
		"env":   getEnvMap(),
	}

	// Evaluate URL expression
	url, err := evaluateExpression(urlExpr, env)
	if err != nil {
		return fmt.Errorf("evaluating URL expression: %w", err)
	}

	// Transform input if specified
	var body interface{}
	if transform != "" {
		body, err = evaluateExpression(transform, env)
		if err != nil {
			return fmt.Errorf("evaluating transform expression: %w", err)
		}
	} else {
		body = input
	}

	// Marshal body to JSON
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest(requestMethod, fmt.Sprintf("%v", url), bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add headers
	for _, header := range headers {
		headerValue, err := evaluateExpression(header, env)
		if err != nil {
			return fmt.Errorf("evaluating header expression: %w", err)
		}

		// Parse header string (format: "Header-Name: Value")
		headerStr := fmt.Sprintf("%v", headerValue)
		parts := strings.SplitN(headerStr, ":", 2)
		if len(parts) == 2 {
			req.Header.Set(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		} else {
			return fmt.Errorf("invalid header format: %s", headerStr)
		}
	}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody := new(bytes.Buffer)
	respBody.ReadFrom(resp.Body)

	// Output response
	fmt.Printf("Status: %s, Response: %s\n", resp.Status, respBody.String())

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	return nil
}

func evaluateExpression(expression string, env map[string]interface{}) (interface{}, error) {
	program, err := expr.Compile(expression, expr.Env(env))
	if err != nil {
		return nil, err
	}

	return expr.Run(program, env)
}

func getEnvMap() map[string]string {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			envMap[pair[0]] = pair[1]
		}
	}
	return envMap
}
