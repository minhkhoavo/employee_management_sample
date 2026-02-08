package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/locvowork/employee_management_sample/apigateway/internal/repository/builder"
)

func main() {
	// Parse flags
	user := flag.String("user", "postgres", "PostgreSQL user")
	password := flag.String("password", "postgres", "PostgreSQL password")
	dbname := flag.String("db", "employee_test", "PostgreSQL database")
	host := flag.String("host", "localhost", "PostgreSQL host")
	port := flag.String("port", "5432", "PostgreSQL port")
	numParents := flag.Int("parents", 50000, "Number of parent records")
	numChildren := flag.Int("children", 500000, "Number of child records")
	flag.Parse()

	fmt.Println("╔════════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                   LARGE-SCALE BATCH UPSERT PERFORMANCE TEST                   ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Display configuration
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Database: %s@%s:%s / %s\n", *user, *host, *port, *dbname)
	fmt.Printf("  Parents:  %d records\n", *numParents)
	fmt.Printf("  Children: %d records\n", *numChildren)
	fmt.Println()

	// Connect to database
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable",
		*user, *password, *dbname, *host, *port)

	logger := NewTimingLogger()
	logger.Start("Connection & Setup")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		fmt.Printf("❌ Failed to connect to PostgreSQL\n")
		fmt.Printf("   Error: %v\n", err)
		fmt.Printf("\n   Make sure PostgreSQL is running with credentials:\n")
		fmt.Printf("   user=%s password=%s dbname=%s\n", *user, "***", *dbname)
		os.Exit(1)
	}

	fmt.Println("✓ Connected to PostgreSQL")

	// Setup tables
	setupTables(db, logger)
	logger.End("Connection & Setup")
	fmt.Println()

	// Main transaction
	logger.Start("UPSERT Transaction")

	txCtx, txCancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer txCancel()

	tx, err := db.BeginTx(txCtx, nil)
	if err != nil {
		fmt.Printf("❌ Failed to begin transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback()

	// Batch 1: Parents (chunked to avoid parameter limit)
	fmt.Printf("\n📦 Batch 1: Parent Records (%d rows)\n", *numParents)
	fmt.Println(strings.Repeat("─", 80))

	// PostgreSQL has a 65535 parameter limit per query
	// With 5 columns (k1, k2, k3, k4, other_val), max rows per batch = 65535/5 = 13,107
	parentChunkSize := 13000
	parentNumChunks := (*numParents + parentChunkSize - 1) / parentChunkSize
	var totalParentAffected int64

	for parentChunk := 0; parentChunk < parentNumChunks; parentChunk++ {
		parentStart := parentChunk * parentChunkSize
		parentEnd := parentStart + parentChunkSize
		if parentEnd > *numParents {
			parentEnd = *numParents
		}
		parentChunkRows := parentEnd - parentStart

		logger.Start(fmt.Sprintf("Build Parent INSERT chunk %d/%d", parentChunk+1, parentNumChunks))
		parentBuilder := builder.NewSQLBuilder().
			Insert("large_parents", "k1", "k2", "k3", "k4", "other_val")

		for i := parentStart + 1; i <= parentEnd; i++ {
			// Use global row number to ensure unique composite keys across all chunks
			parentBuilder = parentBuilder.Values(
				fmt.Sprintf("p%05d", i),
				fmt.Sprintf("p%05d", i),
				fmt.Sprintf("p%05d", i),
				fmt.Sprintf("p%05d", i),
				fmt.Sprintf("Parent data %d", i),
			)
		}

		parentBuilder = parentBuilder.OnConflict("(k1, k2, k3, k4) DO UPDATE SET version = large_parents.version + 1, other_val = EXCLUDED.other_val")
		logger.End(fmt.Sprintf("Build Parent INSERT chunk %d/%d", parentChunk+1, parentNumChunks))

		parentSQL, parentArgs := parentBuilder.Build()
		fmt.Printf("  Chunk %d: SQL size: %.2f KB, Arguments: %d\n", parentChunk+1, float64(len(parentSQL))/1024, len(parentArgs))

		logger.Start(fmt.Sprintf("Execute Parent INSERT chunk %d/%d (%d rows)", parentChunk+1, parentNumChunks, parentChunkRows))
		parentResult, err := tx.ExecContext(txCtx, parentSQL, parentArgs...)
		logger.End(fmt.Sprintf("Execute Parent INSERT chunk %d/%d (%d rows)", parentChunk+1, parentNumChunks, parentChunkRows))

		if err != nil {
			fmt.Printf("❌ Failed to execute parent insert: %v\n", err)
			os.Exit(1)
		}

		parentAffected, _ := parentResult.RowsAffected()
		totalParentAffected += parentAffected
		fmt.Printf("    Rows affected: %d ✓\n", parentAffected)
	}

	fmt.Printf("  Total: %d rows ✓\n", totalParentAffected)
	fmt.Println()

	// Batch 2: Children (chunked to avoid parameter limit)
	fmt.Printf("📦 Batch 2: Child Records (%d rows)\n", *numChildren)
	fmt.Println(strings.Repeat("─", 80))

	// PostgreSQL has a 65535 parameter limit per query
	// With 6 columns (k1, k2, k3, k4, child_id, data), max rows per batch = 65535/6 = 10,922
	childChunkSize := 10000
	childNumChunks := (*numChildren + childChunkSize - 1) / childChunkSize
	var totalChildrenAffected int64

	for chunk := 0; chunk < childNumChunks; chunk++ {
		start := chunk * childChunkSize
		end := start + childChunkSize
		if end > *numChildren {
			end = *numChildren
		}
		chunkRows := end - start

		logger.Start(fmt.Sprintf("Build Child INSERT chunk %d/%d", chunk+1, childNumChunks))
		childBuilder := builder.NewSQLBuilder().
			Insert("large_children", "k1", "k2", "k3", "k4", "child_id", "data")

		for i := start + 1; i <= end; i++ {
			// Each child belongs to a parent (divide children among parents)
			// With 500K children and 50K parents, ~10 children per parent
			parentNum := ((i - 1) / 10) + 1
			childNum := ((i - 1) % 10) + 1

			childBuilder = childBuilder.Values(
				fmt.Sprintf("p%05d", parentNum),
				fmt.Sprintf("p%05d", parentNum),
				fmt.Sprintf("p%05d", parentNum),
				fmt.Sprintf("p%05d", parentNum),
				fmt.Sprintf("child_%05d", childNum), // Unique within parent
				fmt.Sprintf("Child data %d", i),
			)
		}

		childBuilder = childBuilder.OnConflict("(k1, k2, k3, k4, child_id) DO UPDATE SET data = EXCLUDED.data, version = large_children.version + 1")
		logger.End(fmt.Sprintf("Build Child INSERT chunk %d/%d", chunk+1, childNumChunks))

		childSQL, childArgs := childBuilder.Build()

		logger.Start(fmt.Sprintf("Execute Child INSERT chunk %d/%d (%d rows)", chunk+1, childNumChunks, chunkRows))
		childResult, err := tx.ExecContext(txCtx, childSQL, childArgs...)
		logger.End(fmt.Sprintf("Execute Child INSERT chunk %d/%d (%d rows)", chunk+1, childNumChunks, chunkRows))

		if err != nil {
			fmt.Printf("❌ Failed to execute child insert: %v\n", err)
			os.Exit(1)
		}

		childAffected, _ := childResult.RowsAffected()
		totalChildrenAffected += childAffected
		fmt.Printf("  Chunk %d: %d rows in %v ✓\n", chunk+1, childAffected, logger.GetDuration(fmt.Sprintf("Execute Child INSERT chunk %d/%d (%d rows)", chunk+1, childNumChunks, chunkRows)))
	}

	fmt.Printf("  Total: %d rows ✓\n", totalChildrenAffected)
	fmt.Println()

	// Commit
	logger.Start("Commit Transaction")
	if err := tx.Commit(); err != nil {
		fmt.Printf("❌ Failed to commit: %v\n", err)
		os.Exit(1)
	}
	logger.End("Commit Transaction")

	fmt.Println()

	// Verify
	logger.Start("Verify Data")
	var parentCount, childCount int
	_ = db.QueryRowContext(txCtx, "SELECT COUNT(*) FROM large_parents").Scan(&parentCount)
	_ = db.QueryRowContext(txCtx, "SELECT COUNT(*) FROM large_children").Scan(&childCount)
	logger.End("Verify Data")

	fmt.Printf("  Parents: %d ✓\n", parentCount)
	fmt.Printf("  Children: %d ✓\n", childCount)
	fmt.Println()

	logger.End("UPSERT Transaction")

	// Print summary
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println("TIMING SUMMARY")
	fmt.Println(strings.Repeat("═", 80))

	type timing struct {
		name     string
		duration time.Duration
	}

	var timings []timing
	for name, duration := range logger.timings {
		timings = append(timings, timing{name, duration})
	}

	for _, t := range timings {
		indent := ""
		if strings.HasPrefix(t.name, "  ") {
			indent = "  "
		}
		fmt.Printf("%s%-58s %v\n", indent, t.name, t.duration)
	}

	fmt.Println()

	// Performance stats
	fmt.Println(strings.Repeat("═", 80))
	fmt.Println("PERFORMANCE STATISTICS")
	fmt.Println(strings.Repeat("═", 80))

	// Calculate total durations from chunks
	var parentDuration time.Duration
	for parentChunk := 1; parentChunk <= parentNumChunks; parentChunk++ {
		d := logger.GetDuration(fmt.Sprintf("Execute Parent INSERT chunk %d/%d (%d rows)", parentChunk, parentNumChunks, parentChunkSize))
		if d > 0 {
			parentDuration += d
		}
	}

	var childDuration time.Duration
	for chunk := 1; chunk <= childNumChunks; chunk++ {
		// Get actual chunk size for this chunk
		chunkStart := (chunk - 1) * childChunkSize
		chunkEnd := chunkStart + childChunkSize
		if chunkEnd > *numChildren {
			chunkEnd = *numChildren
		}
		chunkSize := chunkEnd - chunkStart

		d := logger.GetDuration(fmt.Sprintf("Execute Child INSERT chunk %d/%d (%d rows)", chunk, childNumChunks, chunkSize))
		if d > 0 {
			childDuration += d
		}
	}

	totalDuration := logger.GetDuration("UPSERT Transaction")

	fmt.Printf("\nParents:\n")
	fmt.Printf("  Count:      %d\n", totalParentAffected)
	fmt.Printf("  Time:       %v\n", parentDuration)
	if parentDuration > 0 {
		fmt.Printf("  Rows/sec:   %.2f\n", float64(totalParentAffected)/parentDuration.Seconds())
		fmt.Printf("  ms/row:     %.4f\n", float64(parentDuration.Milliseconds())/float64(totalParentAffected))
	}

	fmt.Printf("\nChildren:\n")
	fmt.Printf("  Count:      %d\n", totalChildrenAffected)
	fmt.Printf("  Time:       %v\n", childDuration)
	if childDuration > 0 {
		fmt.Printf("  Rows/sec:   %.2f\n", float64(totalChildrenAffected)/childDuration.Seconds())
		fmt.Printf("  ms/row:     %.4f\n", float64(childDuration.Milliseconds())/float64(totalChildrenAffected))
	}

	totalRecords := totalParentAffected + totalChildrenAffected
	fmt.Printf("\nTotal:\n")
	fmt.Printf("  Records:    %d\n", totalRecords)
	fmt.Printf("  Time:       %v\n", totalDuration)
	if totalDuration > 0 {
		fmt.Printf("  Rows/sec:   %.2f\n", float64(totalRecords)/totalDuration.Seconds())
		fmt.Printf("  ms/row:     %.4f\n", float64(totalDuration.Milliseconds())/float64(totalRecords))
	}

	fmt.Println(strings.Repeat("═", 80))
	fmt.Println("✓ Test completed successfully!")
}

func setupTables(db *sql.DB, logger *TimingLogger) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Start("Create tables")

	createParentSQL := `
	CREATE TABLE IF NOT EXISTS large_parents (
		k1 VARCHAR(50),
		k2 VARCHAR(50),
		k3 VARCHAR(50),
		k4 VARCHAR(50),
		other_val TEXT,
		version INTEGER DEFAULT 1,
		PRIMARY KEY (k1, k2, k3, k4)
	)
	`

	createChildSQL := `
	CREATE TABLE IF NOT EXISTS large_children (
		k1 VARCHAR(50),
		k2 VARCHAR(50),
		k3 VARCHAR(50),
		k4 VARCHAR(50),
		child_id VARCHAR(100),
		data TEXT,
		version INTEGER DEFAULT 1,
		PRIMARY KEY (k1, k2, k3, k4, child_id),
		FOREIGN KEY (k1, k2, k3, k4) REFERENCES large_parents(k1, k2, k3, k4) ON DELETE CASCADE
	)
	`

	db.ExecContext(ctx, createParentSQL)
	db.ExecContext(ctx, createChildSQL)
	db.ExecContext(ctx, "TRUNCATE large_children, large_parents")

	logger.End("Create tables")
}

// TimingLogger tracks timing for operations
type TimingLogger struct {
	timings  map[string]time.Duration
	startMap map[string]time.Time
}

func NewTimingLogger() *TimingLogger {
	return &TimingLogger{
		timings:  make(map[string]time.Duration),
		startMap: make(map[string]time.Time),
	}
}

func (tl *TimingLogger) Start(name string) {
	tl.startMap[name] = time.Now()
	fmt.Printf("⏱  %s\n", name)
}

func (tl *TimingLogger) End(name string) {
	elapsed := time.Since(tl.startMap[name])
	tl.timings[name] = elapsed
	fmt.Printf("✓  %s: %v\n", name, elapsed)
}

func (tl *TimingLogger) GetDuration(name string) time.Duration {
	if d, ok := tl.timings[name]; ok {
		return d
	}
	return 0
}
