# 8. Storage & Caching

## 8.1 Atomic File Writer

```go
// internal/storage/writer.go

package storage

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"

    "github.com/yourorg/switchboard-extractor/internal/models"
)

// Writer handles JSON file output
type Writer struct {
    outputDir string
}

// NewWriter creates a new file writer
func NewWriter(outputDir string) (*Writer, error) {
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return nil, fmt.Errorf("creating output directory: %w", err)
    }
    return &Writer{outputDir: outputDir}, nil
}

// WriteAll writes all output files atomically
func (w *Writer) WriteAll(output *models.FullOutput) error {
    // Full output (indented)
    fullPath := filepath.Join(w.outputDir, "switchboard-oracle-data.json")
    if err := w.writeJSON(fullPath, output, true); err != nil {
        return fmt.Errorf("writing full output: %w", err)
    }

    // Minified output
    minPath := filepath.Join(w.outputDir, "switchboard-oracle-data.min.json")
    if err := w.writeJSON(minPath, output, false); err != nil {
        return fmt.Errorf("writing minified output: %w", err)
    }

    // Summary output (current snapshot only)
    summary := models.SummaryOutput{
        Version:  output.Version,
        Oracle:   output.Oracle,
        Metadata: output.Metadata,
        Summary:  output.Summary,
        Metrics:  output.Metrics,
    }
    summaryPath := filepath.Join(w.outputDir, "switchboard-summary.json")
    if err := w.writeJSON(summaryPath, summary, true); err != nil {
        return fmt.Errorf("writing summary: %w", err)
    }

    return nil
}

// writeJSON writes a JSON file with optional indentation
func (w *Writer) writeJSON(path string, data interface{}, indent bool) error {
    var jsonData []byte
    var err error

    if indent {
        jsonData, err = json.MarshalIndent(data, "", "  ")
    } else {
        jsonData, err = json.Marshal(data)
    }

    if err != nil {
        return fmt.Errorf("marshaling JSON: %w", err)
    }

    return WriteAtomic(path, jsonData, 0644)
}

// WriteAtomic writes data to a file atomically using temp file + rename
func WriteAtomic(path string, data []byte, perm os.FileMode) error {
    dir := filepath.Dir(path)

    // Create temp file in same directory for atomic rename
    tmpFile, err := os.CreateTemp(dir, ".tmp-*")
    if err != nil {
        return fmt.Errorf("creating temp file: %w", err)
    }
    tmpPath := tmpFile.Name()

    // Clean up on any error
    defer func() {
        if tmpPath != "" {
            os.Remove(tmpPath)
        }
    }()

    // Write data
    if _, err := tmpFile.Write(data); err != nil {
        tmpFile.Close()
        return fmt.Errorf("writing data: %w", err)
    }

    // Sync to disk
    if err := tmpFile.Sync(); err != nil {
        tmpFile.Close()
        return fmt.Errorf("syncing file: %w", err)
    }

    // Close file
    if err := tmpFile.Close(); err != nil {
        return fmt.Errorf("closing file: %w", err)
    }

    // Set permissions
    if err := os.Chmod(tmpPath, perm); err != nil {
        return fmt.Errorf("setting permissions: %w", err)
    }

    // Atomic rename
    if err := os.Rename(tmpPath, path); err != nil {
        return fmt.Errorf("renaming file: %w", err)
    }

    tmpPath = "" // Prevent deferred cleanup
    return nil
}
```

---
