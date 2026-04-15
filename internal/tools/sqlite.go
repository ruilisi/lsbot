package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruilisi/lsbot/internal/config"
	_ "modernc.org/sqlite"
)

// dbPath returns the path for a named database, stored in the lsbot data dir.
func dbPath(name string) (string, error) {
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return "", fmt.Errorf("invalid database name %q: only letters, numbers, _ and - allowed", name)
		}
	}
	dir := filepath.Join(config.HubDir(), "db")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create db dir: %w", err)
	}
	return filepath.Join(dir, name+".db"), nil
}

// SQLiteExec executes one or more SQL statements (DDL or DML).
func SQLiteExec(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbName, _ := req.Params.Arguments["db"].(string)
	if dbName == "" {
		dbName = "default"
	}
	sqlStr, ok := req.Params.Arguments["sql"].(string)
	if !ok || strings.TrimSpace(sqlStr) == "" {
		return mcp.NewToolResultError("sql is required"), nil
	}

	path, err := dbPath(dbName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open db: %v", err)), nil
	}
	defer db.Close()

	result, err := db.ExecContext(ctx, sqlStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("exec error: %v", err)), nil
	}

	rows, _ := result.RowsAffected()
	return mcp.NewToolResultText(fmt.Sprintf("OK, %d row(s) affected", rows)), nil
}

// SQLiteQuery executes a SELECT and returns results as JSON.
func SQLiteQuery(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dbName, _ := req.Params.Arguments["db"].(string)
	if dbName == "" {
		dbName = "default"
	}
	sqlStr, ok := req.Params.Arguments["sql"].(string)
	if !ok || strings.TrimSpace(sqlStr) == "" {
		return mcp.NewToolResultError("sql is required"), nil
	}

	path, err := dbPath(dbName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open db: %v", err)), nil
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx, sqlStr)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("query error: %v", err)), nil
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("columns error: %v", err)), nil
	}

	var results []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("scan error: %v", err)), nil
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			if b, ok := vals[i].([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = vals[i]
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("rows error: %v", err)), nil
	}

	if results == nil {
		results = []map[string]any{}
	}
	out, _ := json.MarshalIndent(results, "", "  ")
	return mcp.NewToolResultText(string(out)), nil
}
