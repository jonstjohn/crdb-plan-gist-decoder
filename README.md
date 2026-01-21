# CockroachDB Plan Gist Decoder

A Go library and CLI tool to decode CockroachDB plan gists into human-readable EXPLAIN format.

Plan gists are compact, base64-encoded representations of query execution plans stored in CockroachDB's `statement_statistics` table. This tool allows you to decode and visualize these gists offline without needing access to the database.

## Installation

### As a CLI Tool

```bash
go install github.com/jonstjohn/crdb-plan-gist-decoder/cmd@latest
```

### As a Library

```bash
go get github.com/jonstjohn/crdb-plan-gist-decoder
```

## Usage

### Standalone CLI

Run the tool with a base64-encoded gist string:

```bash
crdb-plan-gist-decoder 'AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM'
```

Output:
```
  • update
  │ table: ?
  │ set
  │
  └── • render
      │
      └── • scan
            table: ?@?
            spans: 1+ spans
```

#### Getting Plan Gists from CockroachDB

Query the `statement_statistics` table to extract plan gists:

```sql
SELECT metadata->'plan_gist'
FROM crdb_internal.statement_statistics
LIMIT 1;
```

### As a Module in Go Programs

Import the package in your Go code:

```go
import gist "github.com/jonstjohn/crdb-plan-gist-decoder"
```

#### Basic Example (Without Table/Index Lookups)

```go
package main

import (
    "fmt"
    "log"

    gist "github.com/jonstjohn/crdb-plan-gist-decoder"
)

func main() {
    gistString := "AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM"

    // Decode the gist (table/index names will show as "?")
    node, err := gist.DecodePlanGist(gistString, nil, nil)
    if err != nil {
        log.Fatalf("Error decoding gist: %v", err)
    }

    // Format and print the plan
    output := gist.FormatPlan(node)
    fmt.Print(output)
}
```

#### Advanced Example (With Custom Table/Index Lookups)

For more readable output, provide lookup functions that resolve CockroachDB internal IDs to actual table and index names:

```go
package main

import (
    "fmt"
    "log"

    gist "github.com/jonstjohn/crdb-plan-gist-decoder"
)

func main() {
    gistString := "AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM"

    // Define lookup functions
    tableLookup := func(id int64) string {
        // Map table IDs to names
        tables := map[int64]string{
            112: "users",
            113: "orders",
        }
        if name, ok := tables[id]; ok {
            return name
        }
        return "" // Return empty string to display "?"
    }

    indexLookup := func(tableID int64, indexID int64) string {
        // Map (tableID, indexID) pairs to index names
        type indexKey struct {
            tableID int64
            indexID int64
        }
        indexes := map[indexKey]string{
            {112, 1}: "users_pkey",
            {112, 2}: "users_email_idx",
            {113, 1}: "orders_pkey",
        }
        if name, ok := indexes[indexKey{tableID, indexID}]; ok {
            return name
        }
        return "" // Return empty string to display "?"
    }

    // Decode with lookups
    node, err := gist.DecodePlanGist(gistString, tableLookup, indexLookup)
    if err != nil {
        log.Fatalf("Error decoding gist: %v", err)
    }

    // Format and print
    output := gist.FormatPlan(node)
    fmt.Print(output)
}
```

#### API Reference

**DecodePlanGist**

```go
func DecodePlanGist(gist string, tableLookup TableLookupFunc, indexLookup IndexLookupFunc) (*Node, error)
```

Decodes a base64-encoded plan gist into a plan tree.

- `gist`: The base64-encoded gist string
- `tableLookup`: Optional function to resolve table IDs to names (can be `nil`)
- `indexLookup`: Optional function to resolve index IDs to names (can be `nil`)
- Returns: Root node of the plan tree and any error

**FormatPlan**

```go
func FormatPlan(n *Node) string
```

Formats a decoded plan tree as EXPLAIN-style output with tree characters and proper indentation.

- `n`: The root node from `DecodePlanGist`
- Returns: Formatted plan string

**Lookup Functions**

```go
type TableLookupFunc func(id int64) string
type IndexLookupFunc func(tableID int64, indexID int64) string
```

Both functions should return an empty string for unknown IDs (will display as "?").

## Example Output

The decoder produces output similar to CockroachDB's EXPLAIN format:

```
  • hash join
  │ type: inner
  │ equality cols: 1
  │
  ├── • scan
  │     table: users@users_pkey
  │     spans: FULL SCAN
  │
  └── • scan
        table: orders@orders_user_id_idx
        spans: 1+ spans
```

## Requirements

- Go 1.21 or later

## License

Apache License 2.0

## Contributing

Contributions are welcome! Please feel free to submit issues or pull requests.
