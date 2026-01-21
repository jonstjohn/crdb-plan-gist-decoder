package gistdecoder

// execOperator represents different plan operators in CockroachDB.
type execOperator byte

const (
	unknownOp execOperator = iota
	scanOp
	valuesOp
	filterOp
	invertedFilterOp
	simpleProjectOp
	serializingProjectOp
	renderOp
	applyJoinOp
	hashJoinOp
	mergeJoinOp
	groupByOp
	scalarGroupByOp
	distinctOp
	hashSetOpOp
	streamingSetOpOp
	unionAllOp
	sortOp
	ordinalityOp
	indexJoinOp
	lookupJoinOp
	invertedJoinOp
	zigzagJoinOp
	limitOp
	topKOp
	max1RowOp
	projectSetOp
	windowOp
	explainOptOp
	explainOp
	showTraceOp
	insertOp
	insertFastPathOp
	updateOp
	upsertOp
	deleteOp
	deleteRangeOp
	createTableOp
	createTableAsOp
	createViewOp
	sequenceSelectOp
	saveTableOp
	errorIfRowsOp
	opaqueOp
	alterTableSplitOp
	alterTableUnsplitOp
	alterTableUnsplitAllOp
	alterTableRelocateOp
	bufferOp
	scanBufferOp
	recursiveCTEOp
	controlJobsOp
	controlSchedulesOp
	cancelQueriesOp
	cancelSessionsOp
	createStatisticsOp
	exportOp
	alterRangeRelocateOp
	createFunctionOp
	literalValuesOp
	showCompletionsOp
	callOp
	createTriggerOp
	vectorSearchOp
	vectorMutationSearchOp
	updateSwapOp
	deleteSwapOp
)

// opNames maps operator codes to human-readable names.
var opNames = map[execOperator]string{
	scanOp:               "scan",
	valuesOp:             "values",
	filterOp:             "filter",
	invertedFilterOp:     "inverted filter",
	simpleProjectOp:      "simple project",
	serializingProjectOp: "serializing project",
	renderOp:             "render",
	applyJoinOp:          "apply join",
	hashJoinOp:           "hash join",
	mergeJoinOp:          "merge join",
	groupByOp:            "group by",
	scalarGroupByOp:      "scalar group by",
	distinctOp:           "distinct",
	hashSetOpOp:          "hash set op",
	streamingSetOpOp:     "streaming set op",
	unionAllOp:           "union all",
	sortOp:               "sort",
	ordinalityOp:         "ordinality",
	indexJoinOp:          "index join",
	lookupJoinOp:         "lookup join",
	invertedJoinOp:       "inverted join",
	zigzagJoinOp:         "zigzag join",
	limitOp:              "limit",
	topKOp:               "top-k",
	max1RowOp:            "max1row",
	projectSetOp:         "project set",
	windowOp:             "window",
	insertOp:             "insert",
	updateOp:             "update",
	upsertOp:             "upsert",
	deleteOp:             "delete",
	deleteRangeOp:        "delete range",
	errorIfRowsOp:        "error if rows",
	bufferOp:             "buffer",
	scanBufferOp:         "scan buffer",
	recursiveCTEOp:       "recursive cte",
}
