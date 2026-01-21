// Package gistdecoder provides functionality to decode CockroachDB plan gists
// into human-readable query execution plans.
//
// Plan gists are compact, base64-encoded representations of query execution plans
// stored in CockroachDB's statement_statistics table. This package allows you to
// decode and format these gists offline without needing access to the database.
package gistdecoder

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

const gistVersion = 1

// TableLookupFunc resolves CockroachDB internal table IDs to table names.
// Return an empty string to display "?" for unknown tables.
type TableLookupFunc func(id int64) string

// IndexLookupFunc resolves CockroachDB internal index IDs to index names.
// Return an empty string to display "?" for unknown indexes.
type IndexLookupFunc func(tableID int64, indexID int64) string

// Node represents a decoded plan node in the query execution tree.
// Each node has an operator type, arguments specific to that operator,
// and zero or more child nodes.
type Node struct {
	op       execOperator
	args     map[string]interface{}
	children []*Node
}

// planGistDecoder handles the binary decoding of plan gist data.
type planGistDecoder struct {
	buf           bytes.Reader
	nodeStack     []*Node
	TableLookupFn TableLookupFunc
	IndexLookupFn IndexLookupFunc
}

func (d *planGistDecoder) decodeInt() int {
	val, err := binary.ReadVarint(&d.buf)
	if err != nil {
		panic(fmt.Sprintf("decode error: %v", err))
	}
	return int(val)
}

func (d *planGistDecoder) decodeByte() byte {
	val, err := d.buf.ReadByte()
	if err != nil {
		panic(fmt.Sprintf("decode error: %v", err))
	}
	return val
}

func (d *planGistDecoder) decodeBool() bool {
	return d.decodeByte() != 0
}

func (d *planGistDecoder) decodeID() int64 {
	return int64(d.decodeInt())
}

func (d *planGistDecoder) decodeTable() (int64, string) {
	id := d.decodeID()
	name := "?"
	if d.TableLookupFn != nil {
		if n := d.TableLookupFn(id); n != "" {
			name = n
		}
	}
	return id, name
}

func (d *planGistDecoder) decodeIndex(tableID int64) (int64, string) {
	id := d.decodeID()
	name := "?"
	if d.IndexLookupFn != nil {
		if n := d.IndexLookupFn(tableID, id); n != "" {
			name = n
		}
	}
	return id, name
}

func (d *planGistDecoder) decodeUvarint() uint64 {
	val, err := binary.ReadUvarint(&d.buf)
	if err != nil {
		panic(fmt.Sprintf("decode error: %v", err))
	}
	return val
}

// decodeIntSet decodes CockroachDB's intsets.Fast encoding.
// Format: length (uvarint), then either:
//   - if length == 0: 64-bit bitmap (uvarint)
//   - if length > 0: length pairs of (start, end) uvarints
func (d *planGistDecoder) decodeIntSet() {
	length := d.decodeUvarint()
	if length == 0 {
		// Special case: 64-bit bitmap encoded directly
		d.decodeUvarint()
	} else {
		// Read length number of (start, end) pairs
		for i := uint64(0); i < length; i++ {
			d.decodeUvarint() // start
			d.decodeUvarint() // end
		}
	}
}

func (d *planGistDecoder) decodeScanParams() map[string]interface{} {
	// Decode needed columns (intset)
	d.decodeIntSet()

	// Decode index constraint (number of spans)
	numSpans := d.decodeInt()

	// Decode inverted constraint
	numInvertedSpans := d.decodeInt()

	// Decode hard limit
	hardLimit := d.decodeInt()

	params := make(map[string]interface{})
	if numSpans > 0 {
		if numSpans == 1 {
			params["spans"] = "1 span"
		} else {
			params["spans"] = fmt.Sprintf("%d spans", numSpans)
		}
	}
	if numInvertedSpans > 0 {
		params["inverted_constraint"] = true
	}
	if hardLimit != 0 {
		params["limit"] = "limited"
	}

	return params
}

func (d *planGistDecoder) decodeNodeColumnOrdinals() []int {
	l := d.decodeInt()
	if l < 0 {
		return nil
	}
	return make([]int, l)
}

func (d *planGistDecoder) decodeResultColumns() int {
	return d.decodeInt()
}

func (d *planGistDecoder) decodeJoinType() string {
	jt := d.decodeByte()
	joinTypes := []string{
		"inner", "left outer", "right outer", "full outer",
		"semi", "anti", "intersect all", "except all",
	}
	if int(jt) < len(joinTypes) {
		return joinTypes[jt]
	}
	return fmt.Sprintf("join type %d", jt)
}

func (d *planGistDecoder) decodeRows() int {
	return d.decodeInt()
}

func (d *planGistDecoder) popChild() *Node {
	l := len(d.nodeStack)
	if l == 0 {
		return nil
	}
	n := d.nodeStack[l-1]
	d.nodeStack = d.nodeStack[:l-1]
	return n
}

func (d *planGistDecoder) decodeOperatorBody(op execOperator) (*Node, error) {
	n := &Node{
		op:   op,
		args: make(map[string]interface{}),
	}

	switch op {
	case scanOp:
		tableID, tableName := d.decodeTable()
		indexID, indexName := d.decodeIndex(tableID)
		params := d.decodeScanParams()
		n.args["table"] = tableName
		n.args["index"] = indexName
		n.args["table_id"] = tableID
		n.args["index_id"] = indexID
		for k, v := range params {
			n.args[k] = v
		}

	case valuesOp:
		numRows := d.decodeRows()
		numCols := d.decodeResultColumns()
		n.args["rows"] = numRows
		n.args["columns"] = numCols

	case filterOp:
		n.children = append(n.children, d.popChild())

	case invertedFilterOp:
		n.children = append(n.children, d.popChild())

	case simpleProjectOp, serializingProjectOp:
		_ = d.decodeNodeColumnOrdinals() // cols
		n.children = append(n.children, d.popChild())

	case renderOp:
		numCols := d.decodeResultColumns()
		n.args["columns"] = numCols
		n.children = append(n.children, d.popChild())

	case hashJoinOp:
		joinType := d.decodeJoinType()
		leftEqCols := d.decodeNodeColumnOrdinals()
		rightEqCols := d.decodeNodeColumnOrdinals()
		leftKey := d.decodeBool()
		rightKey := d.decodeBool()
		n.args["type"] = joinType
		n.args["left_eq_cols"] = len(leftEqCols)
		n.args["right_eq_cols"] = len(rightEqCols)
		if leftKey {
			n.args["left_key"] = true
		}
		if rightKey {
			n.args["right_key"] = true
		}
		right := d.popChild()
		left := d.popChild()
		n.children = append(n.children, left, right)

	case mergeJoinOp:
		joinType := d.decodeJoinType()
		_ = d.decodeBool() // leftKey
		_ = d.decodeBool() // rightKey
		n.args["type"] = joinType
		right := d.popChild()
		left := d.popChild()
		n.children = append(n.children, left, right)

	case groupByOp:
		_ = d.decodeNodeColumnOrdinals() // groupCols
		n.children = append(n.children, d.popChild())

	case scalarGroupByOp:
		n.children = append(n.children, d.popChild())

	case distinctOp:
		n.children = append(n.children, d.popChild())

	case sortOp:
		n.children = append(n.children, d.popChild())

	case limitOp:
		n.children = append(n.children, d.popChild())

	case topKOp:
		k := d.decodeInt()
		n.args["k"] = k
		n.children = append(n.children, d.popChild())

	case indexJoinOp:
		tableID, tableName := d.decodeTable()
		_ = d.decodeNodeColumnOrdinals() // keyCols
		n.args["table"] = tableName
		n.args["table_id"] = tableID
		n.children = append(n.children, d.popChild())

	case lookupJoinOp:
		joinType := d.decodeJoinType()
		tableID, tableName := d.decodeTable()
		_, indexName := d.decodeIndex(tableID)
		_ = d.decodeNodeColumnOrdinals() // eqCols
		_ = d.decodeBool()                // eqColsAreKey
		n.args["type"] = joinType
		n.args["table"] = tableName
		n.args["index"] = indexName
		n.children = append(n.children, d.popChild())

	case invertedJoinOp:
		joinType := d.decodeJoinType()
		tableID, tableName := d.decodeTable()
		_, indexName := d.decodeIndex(tableID)
		_ = d.decodeNodeColumnOrdinals() // prefixEqCols
		n.args["type"] = joinType
		n.args["table"] = tableName
		n.args["index"] = indexName
		n.children = append(n.children, d.popChild())

	case unionAllOp, hashSetOpOp, streamingSetOpOp:
		right := d.popChild()
		left := d.popChild()
		n.children = append(n.children, left, right)

	case insertOp:
		tableID, tableName := d.decodeTable()
		d.decodeIntSet() // InsertCols
		d.decodeIntSet() // ReturnCols
		d.decodeIntSet() // CheckCols
		d.decodeBool()   // AutoCommit
		n.args["table"] = tableName
		n.args["table_id"] = tableID
		n.children = append(n.children, d.popChild())

	case updateOp:
		tableID, tableName := d.decodeTable()
		n.args["table"] = tableName
		n.args["table_id"] = tableID
		n.children = append(n.children, d.popChild())

	case deleteOp:
		tableID, tableName := d.decodeTable()
		d.decodeIntSet() // FetchCols
		d.decodeIntSet() // ReturnCols
		d.decodeBool()   // AutoCommit
		n.args["table"] = tableName
		n.args["table_id"] = tableID
		n.children = append(n.children, d.popChild())

	case upsertOp:
		tableID, tableName := d.decodeTable()
		d.decodeIntSet() // InsertCols
		d.decodeIntSet() // FetchCols
		d.decodeIntSet() // UpdateCols
		d.decodeIntSet() // ReturnCols
		d.decodeIntSet() // Checks
		d.decodeBool()   // AutoCommit
		n.args["table"] = tableName
		n.args["table_id"] = tableID
		n.children = append(n.children, d.popChild())

	case errorIfRowsOp:
		n.children = append(n.children, d.popChild())

	default:
		// For unknown operators, try to pop a child if one exists
		if len(d.nodeStack) > 0 {
			n.children = append(n.children, d.popChild())
		}
	}

	return n, nil
}

func (d *planGistDecoder) decodeOp() execOperator {
	val, err := d.buf.ReadByte()
	if err != nil || val == 0 {
		return unknownOp
	}

	n, err := d.decodeOperatorBody(execOperator(val))
	if err != nil {
		panic(err)
	}
	d.nodeStack = append(d.nodeStack, n)

	return n.op
}

// DecodePlanGist decodes a base64-encoded CockroachDB plan gist into a plan tree.
//
// The tableLookup and indexLookup functions are optional. If nil, table and index
// names will be shown as "?". These functions should map CockroachDB internal IDs
// to human-readable names.
//
// Example:
//
//	node, err := DecodePlanGist(gist, tableLookup, indexLookup)
//	if err != nil {
//	    return err
//	}
//	output := FormatPlan(node)
//	fmt.Print(output)
func DecodePlanGist(gist string, tableLookup TableLookupFunc, indexLookup IndexLookupFunc) (*Node, error) {
	b, err := base64.StdEncoding.DecodeString(gist)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}

	var d planGistDecoder
	d.buf.Reset(b)
	d.TableLookupFn = tableLookup
	d.IndexLookupFn = indexLookup

	ver := d.decodeInt()
	if ver != gistVersion {
		return nil, fmt.Errorf("unsupported gist version %d (expected %d)", ver, gistVersion)
	}

	var checks []*Node
	for {
		op := d.decodeOp()
		if op == unknownOp {
			break
		}
		if op == errorIfRowsOp {
			checks = append(checks, d.popChild())
		}
	}

	root := d.popChild()

	// Attach checks if any
	if len(checks) > 0 {
		wrapper := &Node{
			op:       unknownOp,
			args:     map[string]interface{}{"checks": len(checks)},
			children: append([]*Node{root}, checks...),
		}
		return wrapper, nil
	}

	return root, nil
}
