package tools

// =============================================================================
// FileRead
// =============================================================================

// FileReadInput is the input for the Read tool.
type FileReadInput struct {
	FilePath string `json:"file_path"`
	Offset   *int   `json:"offset,omitempty"`
	Limit    *int   `json:"limit,omitempty"`
	Pages    string `json:"pages,omitempty"` // PDF page range, e.g. "1-5"
}

// FileReadOutput is a union of file read result types.
// Check the Type field to determine the variant.
type FileReadOutput struct {
	Type string          `json:"type"` // "text", "image", "notebook", "pdf", "parts", "file_unchanged"
	File FileReadPayload `json:"file"`
}

// FileReadPayload contains the file data. Fields are populated based on the
// parent FileReadOutput.Type.
type FileReadPayload struct {
	// Text variant fields
	FilePath  string `json:"filePath,omitempty"`
	Content   string `json:"content,omitempty"`
	NumLines  int    `json:"numLines,omitempty"`
	StartLine int    `json:"startLine,omitempty"`
	TotalLines int   `json:"totalLines,omitempty"`

	// Image variant fields
	Base64       string `json:"base64,omitempty"`
	ImageType    string `json:"type,omitempty"` // "image/jpeg", "image/png", etc.
	OriginalSize int    `json:"originalSize,omitempty"`
	Dimensions   *struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimensions,omitempty"`

	// Notebook variant fields
	Cells []any `json:"cells,omitempty"`

	// Parts variant fields
	Count     int    `json:"count,omitempty"`
	OutputDir string `json:"outputDir,omitempty"`
}

// =============================================================================
// FileWrite
// =============================================================================

// FileWriteInput is the input for the Write tool.
type FileWriteInput struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// FileWriteOutput is the result of a Write tool execution.
type FileWriteOutput struct {
	Type            string            `json:"type"` // "create" or "update"
	FilePath        string            `json:"filePath"`
	Content         string            `json:"content"`
	StructuredPatch []StructuredPatch `json:"structuredPatch"`
	OriginalFile    *string           `json:"originalFile"` // null for new files
	GitDiff         *GitDiff          `json:"gitDiff,omitempty"`
}

// =============================================================================
// FileEdit
// =============================================================================

// FileEditInput is the input for the Edit tool.
type FileEditInput struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

// FileEditOutput is the result of an Edit tool execution.
type FileEditOutput struct {
	FilePath        string            `json:"filePath"`
	OldString       string            `json:"oldString"`
	NewString       string            `json:"newString"`
	OriginalFile    string            `json:"originalFile"`
	StructuredPatch []StructuredPatch `json:"structuredPatch"`
	UserModified    bool              `json:"userModified"`
	ReplaceAll      bool              `json:"replaceAll"`
	GitDiff         *GitDiff          `json:"gitDiff,omitempty"`
}

// =============================================================================
// Glob
// =============================================================================

// GlobInput is the input for the Glob tool.
type GlobInput struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path,omitempty"`
}

// GlobOutput is the result of a Glob tool execution.
type GlobOutput struct {
	DurationMs int      `json:"durationMs"`
	NumFiles   int      `json:"numFiles"`
	Filenames  []string `json:"filenames"`
	Truncated  bool     `json:"truncated"`
}

// =============================================================================
// Grep
// =============================================================================

// GrepInput is the input for the Grep tool.
type GrepInput struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path,omitempty"`
	Glob       string `json:"glob,omitempty"`
	OutputMode string `json:"output_mode,omitempty"` // "content", "files_with_matches", "count"
	Before     *int   `json:"-B,omitempty"`
	After      *int   `json:"-A,omitempty"`
	Context    *int   `json:"-C,omitempty"`
	ContextAlt *int   `json:"context,omitempty"`
	LineNumbers *bool `json:"-n,omitempty"`
	IgnoreCase *bool  `json:"-i,omitempty"`
	Type       string `json:"type,omitempty"`
	HeadLimit  *int   `json:"head_limit,omitempty"`
	Offset     *int   `json:"offset,omitempty"`
	Multiline  *bool  `json:"multiline,omitempty"`
}

// GrepOutput is the result of a Grep tool execution.
type GrepOutput struct {
	Mode          string   `json:"mode,omitempty"` // "content", "files_with_matches", "count"
	NumFiles      int      `json:"numFiles"`
	Filenames     []string `json:"filenames"`
	Content       string   `json:"content,omitempty"`
	NumLines      *int     `json:"numLines,omitempty"`
	NumMatches    *int     `json:"numMatches,omitempty"`
	AppliedLimit  *int     `json:"appliedLimit,omitempty"`
	AppliedOffset *int     `json:"appliedOffset,omitempty"`
}

// =============================================================================
// NotebookEdit
// =============================================================================

// NotebookEditInput is the input for the NotebookEdit tool.
type NotebookEditInput struct {
	NotebookPath string `json:"notebook_path"`
	NewSource    string `json:"new_source"`
	CellID       string `json:"cell_id,omitempty"`
	CellType     string `json:"cell_type,omitempty"` // "code" or "markdown"
	EditMode     string `json:"edit_mode,omitempty"` // "replace", "insert", "delete"
}

// NotebookEditOutput is the result of a NotebookEdit tool execution.
type NotebookEditOutput struct {
	NewSource    string `json:"new_source"`
	CellID       string `json:"cell_id,omitempty"`
	CellType     string `json:"cell_type"`
	Language     string `json:"language"`
	EditMode     string `json:"edit_mode"`
	Error        string `json:"error,omitempty"`
	NotebookPath string `json:"notebook_path"`
	OriginalFile string `json:"original_file"`
	UpdatedFile  string `json:"updated_file"`
}
