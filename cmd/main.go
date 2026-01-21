package main

import (
	"fmt"
	"os"

	gist "github.com/jonstjohn/crdb-plan-gist-decoder"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <base64-gist-string>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nDecode CockroachDB plan gists into human-readable EXPLAIN format.\n\n")
		fmt.Fprintf(os.Stderr, "Example:\n")
		fmt.Fprintf(os.Stderr, "  %s 'AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM'\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Get gists from CockroachDB:\n")
		fmt.Fprintf(os.Stderr, "  cockroach sql -e \"SELECT metadata->'plan_gist' FROM crdb_internal.statement_statistics LIMIT 1\"\n")
		os.Exit(1)
	}

	gistString := os.Args[1]

	// Default lookup functions return empty string (displays "?")
	// You can customize these to provide actual table/index names
	tableLookup := func(id int64) string {
		return ""
	}

	indexLookup := func(tableID int64, indexID int64) string {
		return ""
	}

	node, err := gist.DecodePlanGist(gistString, tableLookup, indexLookup)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding gist: %v\n", err)
		os.Exit(1)
	}

	output := gist.FormatPlan(node)
	fmt.Print(output)
}
