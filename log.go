package xsql

import "time"

type Log struct {
	Time         time.Duration `json:"time"`
	SQL          string        `json:"sql"`
	SQLPrint     string        `json:"sql_print"`
	Bindings     []interface{} `json:"bindings"`
	RowsAffected int64         `json:"rowsAffected"`
	Error        error         `json:"error"`
}

type DebugFunc func(l *Log)
