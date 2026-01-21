package gistdecoder

import (
	"fmt"
	"strings"
)

// formatNode formats a single node with proper tree characters.
// This function is called recursively to build the complete plan output.
func formatNode(n *Node, prefix string, isLast bool) string {
	if n == nil {
		return ""
	}

	// Skip trivial projections (like CockroachDB does in non-verbose mode)
	if n.op == simpleProjectOp || n.op == serializingProjectOp {
		if len(n.children) > 0 {
			return formatNode(n.children[0], prefix, isLast)
		}
		return ""
	}

	var sb strings.Builder

	// Node name with tree character
	opName := opNames[n.op]
	if opName == "" {
		opName = fmt.Sprintf("op_%d", n.op)
	}
	sb.WriteString(fmt.Sprintf("• %s\n", opName))

	// Determine attribute prefix
	// The │ should align with the • above it
	var attrPrefix string
	if len(n.children) > 0 {
		attrPrefix = "│ "
	} else {
		attrPrefix = "  "
	}

	// Special handling for different operators
	if n.op == scanOp {
		table := n.args["table"]
		index := n.args["index"]
		sb.WriteString(fmt.Sprintf("%stable: %s@%s\n", attrPrefix, table, index))
		if spans, ok := n.args["spans"]; ok {
			// Format as "1+ spans" if multiple
			if strings.Contains(fmt.Sprint(spans), " ") {
				parts := strings.Fields(fmt.Sprint(spans))
				sb.WriteString(fmt.Sprintf("%sspans: %s+ spans\n", attrPrefix, parts[0]))
			} else {
				sb.WriteString(fmt.Sprintf("%sspans: %v\n", attrPrefix, spans))
			}
		} else {
			sb.WriteString(fmt.Sprintf("%sspans: FULL SCAN\n", attrPrefix))
		}
		if limit, ok := n.args["limit"]; ok {
			sb.WriteString(fmt.Sprintf("%slimit: %v\n", attrPrefix, limit))
		}
	} else if n.op == hashJoinOp || n.op == mergeJoinOp || n.op == lookupJoinOp {
		if jt, ok := n.args["type"]; ok {
			sb.WriteString(fmt.Sprintf("%stype: %v\n", attrPrefix, jt))
		}
		if table, ok := n.args["table"]; ok {
			index := n.args["index"]
			sb.WriteString(fmt.Sprintf("%stable: %s@%s\n", attrPrefix, table, index))
		}
		if leftCols, ok := n.args["left_eq_cols"]; ok {
			sb.WriteString(fmt.Sprintf("%sequality cols: %v\n", attrPrefix, leftCols))
		}
	} else if n.op == indexJoinOp {
		if table, ok := n.args["table"]; ok {
			sb.WriteString(fmt.Sprintf("%stable: %s\n", attrPrefix, table))
		}
	} else if n.op == valuesOp {
		if rows, ok := n.args["rows"]; ok {
			sb.WriteString(fmt.Sprintf("%ssize: %v columns, %v rows\n", attrPrefix, n.args["columns"], rows))
		}
	} else if n.op == topKOp {
		if k, ok := n.args["k"]; ok {
			sb.WriteString(fmt.Sprintf("%sk: %v\n", attrPrefix, k))
		}
	} else if n.op == insertOp || n.op == updateOp || n.op == deleteOp || n.op == upsertOp {
		if table, ok := n.args["table"]; ok {
			sb.WriteString(fmt.Sprintf("%stable: %s\n", attrPrefix, table))
		}
		// For updates, add "set" like CockroachDB does
		if n.op == updateOp {
			sb.WriteString(fmt.Sprintf("%sset\n", attrPrefix))
		}
		if len(n.children) > 0 {
			// Empty line with just the vertical bar before children
			sb.WriteString(fmt.Sprintf("%s\n", strings.TrimRight(attrPrefix, " ")))
		}
	} else if n.op == renderOp {
		// Render typically doesn't show attributes in simplified mode
		if len(n.children) > 0 {
			sb.WriteString(fmt.Sprintf("%s\n", strings.TrimRight(attrPrefix, " ")))
		}
	}

	// Format children
	for i, child := range n.children {
		childIsLast := i == len(n.children)-1
		var childPrefix string
		var connector string

		if childIsLast {
			connector = "└── "
			childPrefix = "    "
		} else {
			connector = "├── "
			childPrefix = "│   "
		}

		sb.WriteString(connector)
		childStr := formatNode(child, childPrefix, childIsLast)
		// Insert child output, handling multiline output
		lines := strings.Split(strings.TrimSuffix(childStr, "\n"), "\n")
		for j, line := range lines {
			if j == 0 {
				sb.WriteString(line + "\n")
			} else {
				sb.WriteString(childPrefix + line + "\n")
			}
		}
	}

	return sb.String()
}

// FormatPlan formats a decoded plan tree as EXPLAIN-style output.
// The output matches CockroachDB's EXPLAIN format with proper tree characters
// and indentation.
//
// Example:
//
//	node, _ := DecodePlanGist(gist, nil, nil)
//	output := FormatPlan(node)
//	fmt.Print(output)
//
// Output:
//
//	  • update
//	  │ table: 112
//	  │ set
//	  │
//	  └── • render
//	      │
//	      └── • scan
//	            table: 112@1
//	            spans: 1+ spans
func FormatPlan(n *Node) string {
	if n == nil {
		return ""
	}
	// Add the leading indentation
	output := formatNode(n, "", true)
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	var sb strings.Builder
	for _, line := range lines {
		sb.WriteString("  " + line + "\n")
	}
	return sb.String()
}
