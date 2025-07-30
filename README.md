# pub

A CLI tool that reads JSON from stdin, transforms it using expressions, and sends HTTP requests.

## Installation

```bash
go install github.com/octoberswimmer/pub@latest
```

Or build from source:

```bash
go build -o pub
```

## Usage

```bash
pub [flags] <URL expression>
```

### Flags

- `--transform <expression>` - Transform the input JSON before sending
- `--header <expression>` - Add HTTP headers (can be used multiple times)
- `--request <method>` - HTTP method (default: POST)
- `--dry-run` - Print requests without sending them

### Expression Language

All expressions have access to:
- `input` - The current JSON line being processed
- `env` - Environment variables (including those from `.env` file)

## Examples

### Basic Usage

Send JSON as-is to an endpoint:
```bash
echo '{"message": "hello"}' | pub "http://localhost:8080/webhook"
```

### Transform Input

Wrap input in a data field:
```bash
echo '{"id": 123}' | pub --transform '{data: input}' "http://localhost:8080/api"
```

### Dynamic URLs

Use input fields in the URL:
```bash
echo '{"queue": "urgent", "id": 123}' | pub '"http://localhost:8080/publish?queue=" + input.queue'
```

### Headers with Environment Variables

Add authorization header from environment:
```bash
echo '{"data": "test"}' | pub \
  --header '"Authorization: Bearer " + env.API_TOKEN' \
  "http://api.example.com/endpoint"
```

### Multiple Headers

```bash
echo '{"data": "test"}' | pub \
  --header '"Authorization: Bearer " + env.TOKEN' \
  --header '"X-Request-ID: " + input.id' \
  --request PUT \
  "http://api.example.com/resource"
```

### Dry Run Mode

See what would be sent without making requests:
```bash
echo '{"test": "data"}' | pub --dry-run \
  --transform '{wrapped: input}' \
  --header '"Content-Type: application/json"' \
  "http://localhost:8080/test"
```

### Real-world Example

Process Salesforce platform events:
```bash
force pubsub subscribe /event/My_Event__e | pub \
  --transform '{data: input}' \
  --header '"Authorization: Bearer " + env.EVENTS_PUBLISH_TOKEN' \
  --request POST \
  '"http://localhost:8080/publish?queue=" + input.Queue_Name__c'
```

## Environment Variables

Create a `.env` file in your working directory:
```env
API_TOKEN=your-secret-token
API_ENDPOINT=https://api.example.com
```

These will be automatically loaded and available as `env.API_TOKEN` and `env.API_ENDPOINT` in expressions.

## Processing Multiple Lines

The tool processes JSON line by line, making a separate HTTP request for each valid JSON line:

```bash
# Process multiple events
cat events.jsonl | pub --transform '{event: input}' "http://localhost:8080/ingest"
```

Empty lines are skipped. Invalid JSON lines will log an error and continue processing.

## Error Handling

- HTTP errors (status >= 400) are logged but processing continues
- JSON parsing errors are logged per line
- Expression evaluation errors are logged with details
- The tool exits with status 1 if stdin reading fails