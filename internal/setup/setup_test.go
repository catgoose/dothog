package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// binaryNameFromApp
// ---------------------------------------------------------------------------

func TestBinaryNameFromApp(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "spaces to hyphens", in: "My App", want: "my-app"},
		{name: "trims whitespace", in: " Test ", want: "test"},
		{name: "upper to lower", in: "UPPER CASE", want: "upper-case"},
		{name: "already lowercase hyphenated", in: "already-lower", want: "already-lower"},
		{name: "empty string", in: "", want: ""},
		{name: "multiple spaces", in: "a  b  c", want: "a--b--c"},
		{name: "tabs and newlines trimmed", in: "\t Hello \n", want: "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := binaryNameFromApp(tt.in)
			require.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// isTextFile
// ---------------------------------------------------------------------------

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: ".go", path: "main.go", want: true},
		{name: ".templ", path: "page.templ", want: true},
		{name: ".json", path: "package.json", want: true},
		{name: ".js", path: "app.js", want: true},
		{name: ".ts", path: "app.ts", want: true},
		{name: ".css", path: "style.css", want: true},
		{name: ".html", path: "index.html", want: true},
		{name: ".md", path: "README.md", want: true},
		{name: ".txt", path: "notes.txt", want: true},
		{name: ".toml", path: ".air.toml", want: true},
		{name: ".yaml", path: "config.yaml", want: true},
		{name: ".yml", path: "config.yml", want: true},
		{name: ".mod", path: "go.mod", want: true},
		{name: ".sum", path: "go.sum", want: true},
		{name: ".png binary", path: "logo.png", want: false},
		{name: ".exe binary", path: "app.exe", want: false},
		{name: ".db binary", path: "demo.db", want: false},
		{name: "no extension", path: "Makefile", want: false},
		{name: "empty string", path: "", want: false},
		{name: "nested path .go", path: "internal/routes/handler.go", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTextFile(tt.path)
			require.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// featureFileTag
// ---------------------------------------------------------------------------

func TestFeatureFileTag(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "sse tag",
			content: "// setup:feature:sse\ncode here",
			want:    "sse",
		},
		{
			name:    "auth tag with leading blank line",
			content: "\n// setup:feature:auth\ncode here",
			want:    "auth",
		},
		{
			name:    "block marker not file marker",
			content: "// setup:feature:sse:start\ncode here",
			want:    "",
		},
		{
			name:    "end marker not file marker",
			content: "// setup:feature:sse:end\ncode here",
			want:    "",
		},
		{
			name:    "not first non-blank line",
			content: "code\n// setup:feature:sse",
			want:    "",
		},
		{
			name:    "empty content",
			content: "",
			want:    "",
		},
		{
			name:    "only blank lines",
			content: "\n\n\n",
			want:    "",
		},
		{
			name:    "database tag",
			content: "// setup:feature:database\npackage db",
			want:    "database",
		},
		{
			name:    "multiple leading blank lines",
			content: "\n\n\n// setup:feature:caddy\ndata",
			want:    "caddy",
		},
		{
			name:    "avatar tag",
			content: "// setup:feature:avatar\npackage graph",
			want:    "avatar",
		},
		{
			name:    "demo tag",
			content: "// setup:feature:demo\npackage demo",
			want:    "demo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := featureFileTag(tt.content)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPrunePackageJSON_RemovesCapacitorDepsWhenFeatureIsStripped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	err := os.WriteFile(path, []byte(`{
  "dependencies": {
    "@alpinejs/csp": "^3.15.11",
    "@capacitor/cli": "^8.3.0",
    "@capacitor/core": "^8.3.0",
    "@capacitor/ios": "^8.3.0"
  },
  "devDependencies": {
    "daisyui": "^5.0.0"
  }
}
`), 0644)
	require.NoError(t, err)

	err = prunePackageJSON(path, map[string]bool{FeatureCapacitor: true})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	require.Contains(t, content, `"@alpinejs/csp"`)
	require.NotContains(t, content, `"@capacitor/cli"`)
	require.NotContains(t, content, `"@capacitor/core"`)
	require.NotContains(t, content, `"@capacitor/ios"`)
}

func TestPrunePackageJSON_LeavesCapacitorDepsWhenFeatureIsKept(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	err := os.WriteFile(path, []byte(`{
  "dependencies": {
    "@capacitor/cli": "^8.3.0"
  }
}
`), 0644)
	require.NoError(t, err)

	err = prunePackageJSON(path, map[string]bool{})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), `"@capacitor/cli"`)
}

// ---------------------------------------------------------------------------
// parseFeatureBlockStart
// ---------------------------------------------------------------------------

func TestParseFeatureBlockStart(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantTag string
		wantOK  bool
	}{
		{
			name:    "auth start",
			input:   "// setup:feature:auth:start",
			wantTag: "auth",
			wantOK:  true,
		},
		{
			name:    "sse start",
			input:   "// setup:feature:sse:start",
			wantTag: "sse",
			wantOK:  true,
		},
		{
			name:    "database start",
			input:   "// setup:feature:database:start",
			wantTag: "database",
			wantOK:  true,
		},
		{
			name:    "end marker - not start",
			input:   "// setup:feature:auth:end",
			wantTag: "",
			wantOK:  false,
		},
		{
			name:    "file marker - no start suffix",
			input:   "// setup:feature:auth",
			wantTag: "",
			wantOK:  false,
		},
		{
			name:    "random line",
			input:   "some other line",
			wantTag: "",
			wantOK:  false,
		},
		{
			name:    "empty tag",
			input:   "// setup:feature::start",
			wantTag: "",
			wantOK:  false,
		},
		{
			name:    "empty string",
			input:   "",
			wantTag: "",
			wantOK:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, ok := parseFeatureBlockStart(tt.input)
			require.Equal(t, tt.wantTag, tag)
			require.Equal(t, tt.wantOK, ok)
		})
	}
}

// ---------------------------------------------------------------------------
// parseFeatureBlockEnd
// ---------------------------------------------------------------------------

func TestParseFeatureBlockEnd(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantTag string
		wantOK  bool
	}{
		{
			name:    "auth end",
			input:   "// setup:feature:auth:end",
			wantTag: "auth",
			wantOK:  true,
		},
		{
			name:    "database end",
			input:   "// setup:feature:database:end",
			wantTag: "database",
			wantOK:  true,
		},
		{
			name:    "sse end",
			input:   "// setup:feature:sse:end",
			wantTag: "sse",
			wantOK:  true,
		},
		{
			name:    "start marker - not end",
			input:   "// setup:feature:auth:start",
			wantTag: "",
			wantOK:  false,
		},
		{
			name:    "file marker - no end suffix",
			input:   "// setup:feature:auth",
			wantTag: "",
			wantOK:  false,
		},
		{
			name:    "random line",
			input:   "random",
			wantTag: "",
			wantOK:  false,
		},
		{
			name:    "empty tag",
			input:   "// setup:feature::end",
			wantTag: "",
			wantOK:  false,
		},
		{
			name:    "empty string",
			input:   "",
			wantTag: "",
			wantOK:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tag, ok := parseFeatureBlockEnd(tt.input)
			require.Equal(t, tt.wantTag, tag)
			require.Equal(t, tt.wantOK, ok)
		})
	}
}

// ---------------------------------------------------------------------------
// collapseBlankLines
// ---------------------------------------------------------------------------

func TestCollapseBlankLines(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		lines []string
	}{
		{
			name:  "3 blanks collapse to 2",
			lines: []string{"a", "", "", "", "b"},
			want:  "a\n\n\nb",
		},
		{
			name:  "1 blank preserved",
			lines: []string{"a", "", "b"},
			want:  "a\n\nb",
		},
		{
			name:  "2 blanks preserved",
			lines: []string{"a", "", "", "b"},
			want:  "a\n\n\nb",
		},
		{
			name:  "4 blanks collapse to 2",
			lines: []string{"a", "", "", "", "", "b"},
			want:  "a\n\n\nb",
		},
		{
			name:  "5 blanks collapse to 2",
			lines: []string{"a", "", "", "", "", "", "b"},
			want:  "a\n\n\nb",
		},
		{
			name:  "no blanks",
			lines: []string{"a", "b", "c"},
			want:  "a\nb\nc",
		},
		{
			name:  "empty input",
			lines: []string{},
			want:  "",
		},
		{
			name:  "single line",
			lines: []string{"hello"},
			want:  "hello",
		},
		{
			name:  "multiple collapse regions",
			lines: []string{"a", "", "", "", "b", "", "", "", "c"},
			want:  "a\n\n\nb\n\n\nc",
		},
		{
			name:  "whitespace-only lines count as blank",
			lines: []string{"a", "  ", "\t", "   ", "b"},
			want:  "a\n  \n\t\nb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := collapseBlankLines(tt.lines)
			require.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// stripBlocks
// ---------------------------------------------------------------------------

func TestStripBlocks(t *testing.T) {
	t.Run("demo blocks removed when tag in removeTags", func(t *testing.T) {
		content := strings.Join([]string{
			"before",
			"// setup:feature:demo:start",
			"demo code",
			"// setup:feature:demo:end",
			"after",
		}, "\n")
		got := stripBlocks(content, map[string]bool{"demo": true})
		require.Contains(t, got, "before")
		require.Contains(t, got, "after")
		require.NotContains(t, got, "demo code")
		require.NotContains(t, got, "setup:feature:demo")
	})

	t.Run("demo blocks kept when tag not in removeTags", func(t *testing.T) {
		content := strings.Join([]string{
			"before",
			"// setup:feature:demo:start",
			"demo code",
			"// setup:feature:demo:end",
			"after",
		}, "\n")
		got := stripBlocks(content, map[string]bool{})
		require.Contains(t, got, "before")
		require.Contains(t, got, "after")
		require.Contains(t, got, "demo code")
		// Marker lines are always stripped even when keeping content
		require.NotContains(t, got, "setup:feature:demo:start")
		require.NotContains(t, got, "setup:feature:demo:end")
	})

	t.Run("feature block removed when tag in removeTags", func(t *testing.T) {
		content := strings.Join([]string{
			"before",
			"// setup:feature:auth:start",
			"auth code",
			"// setup:feature:auth:end",
			"after",
		}, "\n")
		got := stripBlocks(content, map[string]bool{"auth": true})
		require.Contains(t, got, "before")
		require.Contains(t, got, "after")
		require.NotContains(t, got, "auth code")
		require.NotContains(t, got, "setup:feature:auth")
	})

	t.Run("feature block kept when tag not in removeTags", func(t *testing.T) {
		content := strings.Join([]string{
			"before",
			"// setup:feature:auth:start",
			"auth code",
			"// setup:feature:auth:end",
			"after",
		}, "\n")
		got := stripBlocks(content, map[string]bool{})
		require.Contains(t, got, "before")
		require.Contains(t, got, "after")
		require.Contains(t, got, "auth code")
		// Marker lines are always stripped even when keeping content
		require.NotContains(t, got, "setup:feature:auth:start")
		require.NotContains(t, got, "setup:feature:auth:end")
	})

	t.Run("feature block kept when removeTags is nil", func(t *testing.T) {
		content := strings.Join([]string{
			"before",
			"// setup:feature:sse:start",
			"sse code",
			"// setup:feature:sse:end",
			"after",
		}, "\n")
		got := stripBlocks(content, nil)
		require.Contains(t, got, "sse code")
	})

	t.Run("inner feature block inside demo block all removed", func(t *testing.T) {
		content := strings.Join([]string{
			"// setup:feature:demo:start",
			"outer",
			"// setup:feature:sse:start",
			"sse stuff",
			"// setup:feature:sse:end",
			"more outer",
			"// setup:feature:demo:end",
		}, "\n")
		got := stripBlocks(content, map[string]bool{"demo": true})
		require.NotContains(t, got, "outer")
		require.NotContains(t, got, "sse stuff")
		require.NotContains(t, got, "more outer")
	})

	t.Run("nested feature blocks - outer removed takes inner", func(t *testing.T) {
		content := strings.Join([]string{
			"// setup:feature:auth:start",
			"auth code",
			"// setup:feature:graph:start",
			"graph in auth",
			"// setup:feature:graph:end",
			"more auth",
			"// setup:feature:auth:end",
		}, "\n")
		got := stripBlocks(content, map[string]bool{"auth": true})
		require.NotContains(t, got, "auth code")
		require.NotContains(t, got, "graph in auth")
		require.NotContains(t, got, "more auth")
	})

	t.Run("nested feature blocks - inner removed keeps outer", func(t *testing.T) {
		content := strings.Join([]string{
			"// setup:feature:auth:start",
			"auth code",
			"// setup:feature:graph:start",
			"graph in auth",
			"// setup:feature:graph:end",
			"more auth",
			"// setup:feature:auth:end",
		}, "\n")
		got := stripBlocks(content, map[string]bool{"graph": true})
		require.Contains(t, got, "auth code")
		require.NotContains(t, got, "graph in auth")
		require.Contains(t, got, "more auth")
	})

	t.Run("marker lines always stripped even when keeping content", func(t *testing.T) {
		content := strings.Join([]string{
			"line1",
			"// setup:feature:sse:start",
			"sse code",
			"// setup:feature:sse:end",
			"line2",
		}, "\n")
		got := stripBlocks(content, map[string]bool{})
		require.NotContains(t, got, "// setup:feature:sse:start")
		require.NotContains(t, got, "// setup:feature:sse:end")
		require.Contains(t, got, "sse code")
		require.Contains(t, got, "line1")
		require.Contains(t, got, "line2")
	})

	t.Run("avatar block removed while graph block kept", func(t *testing.T) {
		content := strings.Join([]string{
			"// setup:feature:graph:start",
			"graph code",
			"// setup:feature:graph:end",
			"between",
			"// setup:feature:avatar:start",
			"avatar code",
			"// setup:feature:avatar:end",
			"after",
		}, "\n")
		got := stripBlocks(content, map[string]bool{"avatar": true})
		require.Contains(t, got, "graph code")
		require.Contains(t, got, "between")
		require.NotContains(t, got, "avatar code")
		require.Contains(t, got, "after")
	})

	t.Run("no markers returns content unchanged except collapse", func(t *testing.T) {
		content := "line1\nline2\nline3"
		got := stripBlocks(content, map[string]bool{})
		require.Equal(t, content, got)
	})

	t.Run("blank lines after removal get collapsed", func(t *testing.T) {
		content := strings.Join([]string{
			"before",
			"",
			"// setup:feature:demo:start",
			"demo",
			"// setup:feature:demo:end",
			"",
			"",
			"",
			"after",
		}, "\n")
		got := stripBlocks(content, map[string]bool{"demo": true})
		// The 4 blank lines (1 before + 3 after) should collapse
		require.Contains(t, got, "before")
		require.Contains(t, got, "after")
		require.NotContains(t, got, "demo")
	})

	t.Run("multiple demo blocks removed", func(t *testing.T) {
		content := strings.Join([]string{
			"keep1",
			"// setup:feature:demo:start",
			"demo1",
			"// setup:feature:demo:end",
			"keep2",
			"// setup:feature:demo:start",
			"demo2",
			"// setup:feature:demo:end",
			"keep3",
		}, "\n")
		got := stripBlocks(content, map[string]bool{"demo": true})
		require.Contains(t, got, "keep1")
		require.Contains(t, got, "keep2")
		require.Contains(t, got, "keep3")
		require.NotContains(t, got, "demo1")
		require.NotContains(t, got, "demo2")
	})
}

// ---------------------------------------------------------------------------
// goPkgName
// ---------------------------------------------------------------------------

func TestGoPkgName(t *testing.T) {
	tests := []struct {
		name       string
		importPath string
		want       string
	}{
		{name: "stdlib simple", importPath: "fmt", want: "fmt"},
		{name: "stdlib nested", importPath: "net/http", want: "http"},
		{name: "versioned module v4", importPath: "github.com/labstack/echo/v4", want: "echo"},
		{name: "versioned module v2", importPath: "github.com/foo/bar/v2", want: "bar"},
		{name: "gopkg.in versioned", importPath: "gopkg.in/natefinsh/lumberjack.v2", want: "lumberjack"},
		{name: "regular external", importPath: "github.com/regular/pkg", want: "pkg"},
		{name: "stdlib os", importPath: "os", want: "os"},
		{name: "stdlib path/filepath", importPath: "path/filepath", want: "filepath"},
		{name: "deeply nested", importPath: "github.com/org/repo/internal/pkg/sub", want: "sub"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := goPkgName(tt.importPath)
			require.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// removeOrphanedImportLines
// ---------------------------------------------------------------------------

func TestRemoveOrphanedImportLines(t *testing.T) {
	// Create a temp directory structure to simulate an internal package layout.
	tmpDir := t.TempDir()
	modulePath := "example.com/myapp"

	// Create an existing internal package directory.
	existingPkg := filepath.Join(tmpDir, "internal", "existing")
	require.NoError(t, os.MkdirAll(existingPkg, 0755))
	// The "internal/missing" directory does NOT exist.

	t.Run("internal import with existing dir is kept", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	"example.com/myapp/internal/existing"`,
			")",
			"",
			"func main() {",
			"	existing.Do()",
			"}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.Contains(t, got, `"example.com/myapp/internal/existing"`)
	})

	t.Run("internal import with missing dir is removed", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	"example.com/myapp/internal/missing"`,
			")",
			"",
			"func main() {",
			"	missing.Do()",
			"}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.NotContains(t, got, `"example.com/myapp/internal/missing"`)
	})

	t.Run("stdlib import used in body is kept", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	"fmt"`,
			")",
			"",
			"func main() {",
			"	fmt.Println()",
			"}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.Contains(t, got, `"fmt"`)
	})

	t.Run("stdlib import unused in body is removed", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	"fmt"`,
			`	"os"`,
			")",
			"",
			"func main() {",
			"	fmt.Println()",
			"}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.Contains(t, got, `"fmt"`)
		require.NotContains(t, got, `"os"`)
	})

	t.Run("external package import is always kept", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	"github.com/labstack/echo/v4"`,
			")",
			"",
			"func main() {}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.Contains(t, got, `"github.com/labstack/echo/v4"`)
	})

	t.Run("side-effect import is always kept", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	_ "net/http/pprof"`,
			")",
			"",
			"func main() {}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.Contains(t, got, `_ "net/http/pprof"`)
	})

	t.Run("aliased import with alias used is kept", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	myfmt "fmt"`,
			")",
			"",
			"func main() {",
			"	myfmt.Println()",
			"}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.Contains(t, got, `myfmt "fmt"`)
	})

	t.Run("aliased import with alias unused is removed", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	myfmt "fmt"`,
			")",
			"",
			"func main() {",
			"}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.NotContains(t, got, `myfmt "fmt"`)
	})

	t.Run("no import block returns content unchanged", func(t *testing.T) {
		content := "package main\n\nfunc main() {}\n"
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.Equal(t, content, got)
	})

	t.Run("all imports kept returns content unchanged", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	"fmt"`,
			")",
			"",
			"func main() {",
			"	fmt.Println()",
			"}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		require.Equal(t, content, got)
	})

	t.Run("mixed: some removed some kept", func(t *testing.T) {
		content := strings.Join([]string{
			"package main",
			"",
			"import (",
			`	"fmt"`,
			`	"os"`,
			`	"strings"`,
			`	_ "net/http/pprof"`,
			`	"github.com/external/lib"`,
			`	"example.com/myapp/internal/existing"`,
			`	"example.com/myapp/internal/missing"`,
			")",
			"",
			"func main() {",
			"	fmt.Println()",
			"	existing.Do()",
			"}",
		}, "\n")
		got := removeOrphanedImportLines(content, tmpDir, modulePath)
		// Kept
		require.Contains(t, got, `"fmt"`)
		require.Contains(t, got, `_ "net/http/pprof"`)
		require.Contains(t, got, `"github.com/external/lib"`)
		require.Contains(t, got, `"example.com/myapp/internal/existing"`)
		// Removed
		require.NotContains(t, got, `"os"`)
		require.NotContains(t, got, `"strings"`)
		require.NotContains(t, got, `"example.com/myapp/internal/missing"`)
	})
}

// ---------------------------------------------------------------------------
// ImplicitFeatures
// ---------------------------------------------------------------------------

func TestDatabaseBlocksStrippedWhenNoMSSQLOrPostgres(t *testing.T) {
	// "database" is no longer implicit: scaffolds without MSSQL or PostgreSQL
	// strip the app-data marker blocks entirely.
	content := strings.Join([]string{
		"before",
		"// setup:feature:database:start",
		"database code",
		"// setup:feature:database:end",
		"after",
	}, "\n")
	removeTags := make(map[string]bool)
	keep := make(map[string]bool)
	for _, f := range ImplicitFeatures {
		keep[f] = true
	}
	for _, f := range AllFeatures {
		if !keep[f] {
			removeTags[f] = true
		}
	}
	got := stripBlocks(content, removeTags)
	require.NotContains(t, got, "database code", "database blocks should be stripped when no mssql/postgres selected")
	require.Contains(t, got, "before")
	require.Contains(t, got, "after")
}

func TestDatabaseBlocksKeptWhenMSSQLSelected(t *testing.T) {
	// MSSQL implies the internal "database" feature via featureDeps, so
	// selecting MSSQL must keep the app-data marker blocks intact.
	content := strings.Join([]string{
		"before",
		"// setup:feature:database:start",
		"database code",
		"// setup:feature:database:end",
		"after",
	}, "\n")
	expanded := ExpandFeatureDeps([]string{FeatureMSSQL})
	keep := make(map[string]bool)
	for _, f := range expanded {
		keep[f] = true
	}
	for _, f := range ImplicitFeatures {
		keep[f] = true
	}
	removeTags := make(map[string]bool)
	for _, f := range AllFeatures {
		if !keep[f] {
			removeTags[f] = true
		}
	}
	got := stripBlocks(content, removeTags)
	require.Contains(t, got, "database code", "database blocks should be kept when mssql is selected")
	require.Contains(t, got, "before")
	require.Contains(t, got, "after")
}

func TestMSSQLBlocksStrippedWhenNotSelected(t *testing.T) {
	content := strings.Join([]string{
		"before",
		"// setup:feature:mssql:start",
		"mssql code",
		"// setup:feature:mssql:end",
		"after",
	}, "\n")
	got := stripBlocks(content, map[string]bool{"mssql": true})
	require.Contains(t, got, "before")
	require.Contains(t, got, "after")
	require.NotContains(t, got, "mssql code")
}

func TestMSSQLBlocksKeptWhenSelected(t *testing.T) {
	content := strings.Join([]string{
		"before",
		"// setup:feature:mssql:start",
		"mssql code",
		"// setup:feature:mssql:end",
		"after",
	}, "\n")
	got := stripBlocks(content, map[string]bool{})
	require.Contains(t, got, "before")
	require.Contains(t, got, "after")
	require.Contains(t, got, "mssql code")
}

func TestStripFeatureFileMarker(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips marker as first line",
			input: "// setup:feature:sse\npackage ssebroker\n\nfunc Foo() {}\n",
			want:  "package ssebroker\n\nfunc Foo() {}\n",
		},
		{
			name:  "strips marker after blank lines",
			input: "\n\n// setup:feature:graph\npackage graph\n",
			want:  "\n\npackage graph\n",
		},
		{
			name:  "no marker leaves content unchanged",
			input: "package main\n\nfunc main() {}\n",
			want:  "package main\n\nfunc main() {}\n",
		},
		{
			name:  "block marker not stripped",
			input: "// setup:feature:auth:start\ncode\n// setup:feature:auth:end\n",
			want:  "// setup:feature:auth:start\ncode\n// setup:feature:auth:end\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripFeatureFileMarker(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// stripEnvBlocks
// ---------------------------------------------------------------------------

func TestStripEnvBlocks(t *testing.T) {
	t.Run("removes block when tag is in removeTags", func(t *testing.T) {
		content := strings.Join([]string{
			"# before",
			"",
			"# setup:feature:graph:start",
			"# AZURE_TENANT_ID=",
			"# AZURE_CLIENT_ID=",
			"# setup:feature:graph:end",
			"",
			"# after",
		}, "\n")
		got := stripEnvBlocks(content, map[string]bool{"graph": true})
		require.NotContains(t, got, "AZURE_TENANT_ID")
		require.NotContains(t, got, "AZURE_CLIENT_ID")
		require.Contains(t, got, "# before")
		require.Contains(t, got, "# after")
	})

	t.Run("keeps block content when tag is not in removeTags", func(t *testing.T) {
		content := strings.Join([]string{
			"# setup:feature:graph:start",
			"# AZURE_TENANT_ID=",
			"# AZURE_CLIENT_ID=",
			"# AZURE_CLIENT_SECRET=",
			"# setup:feature:graph:end",
		}, "\n")
		got := stripEnvBlocks(content, map[string]bool{})
		require.Contains(t, got, "# AZURE_TENANT_ID=")
		require.Contains(t, got, "# AZURE_CLIENT_ID=")
		require.Contains(t, got, "# AZURE_CLIENT_SECRET=")
		require.NotContains(t, got, "setup:feature:graph")
	})

	t.Run("always strips marker lines", func(t *testing.T) {
		content := strings.Join([]string{
			"# setup:feature:graph:start",
			"# AZURE_TENANT_ID=",
			"# setup:feature:graph:end",
		}, "\n")
		got := stripEnvBlocks(content, map[string]bool{})
		require.NotContains(t, got, "# setup:feature:graph:start")
		require.NotContains(t, got, "# setup:feature:graph:end")
		require.Contains(t, got, "# AZURE_TENANT_ID=")
	})

	t.Run("no markers returns content unchanged", func(t *testing.T) {
		content := "# plain env file\nSERVER_PORT=5000\n"
		got := stripEnvBlocks(content, map[string]bool{"graph": true})
		require.Equal(t, content, got)
	})
}

// ---------------------------------------------------------------------------
// CopyRepoTo — cert exclusion
// ---------------------------------------------------------------------------

func TestCopyRepoToExcludesLocalhostCerts(t *testing.T) {
	src := t.TempDir()
	dest := filepath.Join(t.TempDir(), "out")

	files := map[string]string{
		"go.mod":         "module example.test\n",
		"localhost.crt":  "FAKE CERT\n",
		"localhost.key":  "FAKE KEY\n",
		"keep/keep.txt":  "keep me\n",
		"node_modules/x": "junk\n",
	}
	for rel, body := range files {
		full := filepath.Join(src, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(body), 0o644))
	}

	excludes := []string{".git", "node_modules", "localhost.crt", "localhost.key"}
	require.NoError(t, CopyRepoTo(src, dest, excludes))

	// localhost.* must be excluded.
	_, err := os.Stat(filepath.Join(dest, "localhost.crt"))
	require.True(t, os.IsNotExist(err), "localhost.crt should not be copied: %v", err)
	_, err = os.Stat(filepath.Join(dest, "localhost.key"))
	require.True(t, os.IsNotExist(err), "localhost.key should not be copied: %v", err)

	// node_modules dir must still be excluded.
	_, err = os.Stat(filepath.Join(dest, "node_modules"))
	require.True(t, os.IsNotExist(err), "node_modules should not be copied: %v", err)

	// Regular files should still copy.
	gotMod, err := os.ReadFile(filepath.Join(dest, "go.mod"))
	require.NoError(t, err)
	require.Equal(t, "module example.test\n", string(gotMod))
	gotKeep, err := os.ReadFile(filepath.Join(dest, "keep", "keep.txt"))
	require.NoError(t, err)
	require.Equal(t, "keep me\n", string(gotKeep))
}

// ---------------------------------------------------------------------------
// resolvePlatform
// ---------------------------------------------------------------------------

func TestResolvePlatform(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		want      string
		expectErr bool
	}{
		{name: "linux explicit", in: "linux", want: PlatformLinux},
		{name: "windows explicit", in: "windows", want: PlatformWindows},
		{name: "case insensitive", in: "LINUX", want: PlatformLinux},
		{name: "whitespace tolerated", in: "  windows  ", want: PlatformWindows},
		{name: "rejects darwin", in: "darwin", expectErr: true},
		{name: "rejects unknown", in: "freebsd", expectErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolvePlatform(tt.in)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// applyPlatformAdjustments — .air/* rewrites per platform
// ---------------------------------------------------------------------------

func writeAirFixture(t *testing.T, dir string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".air"), 0o755))
	server := `root = "."
tmp_dir = "tmp"

[build]
  args_bin = ["-env=development"]
  bin = "./tmp/main"
  cmd = "go tool mage airBuild"
  exclude_dir = ["assets", "tmp"]
  include_ext = ["go"]
`
	lint := `root = "."

[build]
  args_bin = []
  bin = "/bin/echo"
  cmd = "mage lint"
  include_ext = ["go"]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".air", "server.toml"), []byte(server), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".air", "lint.toml"), []byte(lint), 0o644))
}

func TestApplyPlatformAdjustments_LinuxLeavesFilesUntouched(t *testing.T) {
	dir := t.TempDir()
	writeAirFixture(t, dir)

	srvBefore, err := os.ReadFile(filepath.Join(dir, ".air", "server.toml"))
	require.NoError(t, err)
	lintBefore, err := os.ReadFile(filepath.Join(dir, ".air", "lint.toml"))
	require.NoError(t, err)

	require.NoError(t, applyPlatformAdjustments(dir, PlatformLinux))

	srvAfter, err := os.ReadFile(filepath.Join(dir, ".air", "server.toml"))
	require.NoError(t, err)
	lintAfter, err := os.ReadFile(filepath.Join(dir, ".air", "lint.toml"))
	require.NoError(t, err)

	require.Equal(t, string(srvBefore), string(srvAfter), "Linux must not rewrite server.toml")
	require.Equal(t, string(lintBefore), string(lintAfter), "Linux must not rewrite lint.toml")
}

func TestApplyPlatformAdjustments_WindowsRewritesAir(t *testing.T) {
	dir := t.TempDir()
	writeAirFixture(t, dir)

	require.NoError(t, applyPlatformAdjustments(dir, PlatformWindows))

	srv, err := os.ReadFile(filepath.Join(dir, ".air", "server.toml"))
	require.NoError(t, err)
	srvStr := string(srv)
	require.Contains(t, srvStr, `bin = "./tmp/main.exe"`)
	require.Contains(t, srvStr, `cmd = "go tool mage airBuild"`)
	require.NotContains(t, srvStr, `bin = "./tmp/main"`+"\n", "must rewrite the Linux bin line")

	lint, err := os.ReadFile(filepath.Join(dir, ".air", "lint.toml"))
	require.NoError(t, err)
	lintStr := string(lint)
	require.Contains(t, lintStr, `bin = "cmd"`)
	require.Contains(t, lintStr, `args_bin = ["/c", "exit"]`)
	require.NotContains(t, lintStr, `/bin/echo`)
}

// buildQuickStart platform shaping
func TestBuildQuickStart_PlatformShapesFromSource(t *testing.T) {
	linux := buildQuickStart("asdfasdf", "33848", PlatformLinux)
	require.Contains(t, linux, "go build -o asdfasdf .")
	require.Contains(t, linux, "./asdfasdf\n")
	require.NotContains(t, linux, "asdfasdf.exe\n")

	windows := buildQuickStart("asdfasdf", "33848", PlatformWindows)
	require.Contains(t, windows, "go build -o asdfasdf.exe .")
	require.Contains(t, windows, ".\\asdfasdf.exe\n")
	require.Contains(t, windows, "```powershell")
}

// ---------------------------------------------------------------------------
// ExpandFeatureDeps — tag-graph closure semantics
// ---------------------------------------------------------------------------

func TestExpandFeatureDeps_NoDeps(t *testing.T) {
	got := ExpandFeatureDeps([]string{FeatureCSRF, FeatureSessionSettings})
	require.Contains(t, got, FeatureCSRF)
	require.Contains(t, got, FeatureSessionSettings)
	require.NotContains(t, got, FeatureDemo)
	require.NotContains(t, got, FeatureDatabase)
}

func TestExpandFeatureDeps_AvatarPullsGraph(t *testing.T) {
	// Avatar imports the Graph package directly; the strip set must keep
	// Graph or avatar.go can't compile.
	got := ExpandFeatureDeps([]string{FeatureAvatar})
	require.Contains(t, got, FeatureAvatar)
	require.Contains(t, got, FeatureGraph)
}

func TestExpandFeatureDeps_MSSQLPullsDatabase(t *testing.T) {
	got := ExpandFeatureDeps([]string{FeatureMSSQL})
	require.Contains(t, got, FeatureMSSQL)
	require.Contains(t, got, FeatureDatabase,
		"MSSQL must imply the (hidden) database scaffolding")
}

func TestExpandFeatureDeps_PostgresPullsDatabase(t *testing.T) {
	got := ExpandFeatureDeps([]string{FeaturePostgres})
	require.Contains(t, got, FeaturePostgres)
	require.Contains(t, got, FeatureDatabase)
}

func TestExpandFeatureDeps_DemoPullsSessionSettings(t *testing.T) {
	got := ExpandFeatureDeps([]string{FeatureDemo})
	require.Contains(t, got, FeatureDemo)
	require.Contains(t, got, FeatureSessionSettings,
		"demo content reads session settings; the dep must close")
}

func TestExpandFeatureDeps_SSEPullsCaddy(t *testing.T) {
	// SSE implies the hidden Caddy tag so the dev HTTPS/H3 front-proxy ships
	// alongside the SSE broker. Caddy is not a standalone selectable feature.
	got := ExpandFeatureDeps([]string{FeatureSSE})
	require.Contains(t, got, FeatureSSE)
	require.Contains(t, got, FeatureCaddy,
		"SSE must close to Caddy via featureDeps")
}

func TestExpandFeatureDeps_TransitiveOrderIndependent(t *testing.T) {
	// Avatar → Graph; passing them in reverse order must still close to the
	// same set (the closure walks to a fixed point).
	a := ExpandFeatureDeps([]string{FeatureAvatar, FeatureGraph})
	b := ExpandFeatureDeps([]string{FeatureGraph, FeatureAvatar})
	require.ElementsMatch(t, a, b)
}

func TestExpandFeatureDeps_PreservesAllFeaturesOrdering(t *testing.T) {
	// The output ordering follows AllFeatures so scaffold-generation code
	// can rely on deterministic iteration even when callers pass features
	// in arbitrary order.
	got := ExpandFeatureDeps([]string{FeatureSSE, FeatureAuth})
	authIdx := -1
	sseIdx := -1
	for i, f := range got {
		switch f {
		case FeatureAuth:
			authIdx = i
		case FeatureSSE:
			sseIdx = i
		}
	}
	require.NotEqual(t, -1, authIdx)
	require.NotEqual(t, -1, sseIdx)
	require.Less(t, authIdx, sseIdx,
		"AllFeatures lists auth before sse; ExpandFeatureDeps must preserve that")
}

// ---------------------------------------------------------------------------
// Representative-bundle setup-strip verification
//
// These tests mirror the wizard presets defined in mage_setup.go and assert
// that real strip behavior matches the current setup model:
//   - link_relations is gone (baseline, never a strip tag)
//   - database is hidden and only ships with MSSQL/PostgreSQL
//   - Avatar closes to Graph through featureDeps
//   - SSE pulls the hidden Caddy tag
// ---------------------------------------------------------------------------

// representativeBundleContent is a synthetic source file with one tagged
// block per AllFeatures entry. Each block's payload is "<tag>-CODE" so a
// test can grep for survivors after stripBlocks runs.
func representativeBundleContent(t *testing.T) string {
	t.Helper()
	var lines []string
	lines = append(lines, "preamble")
	for _, tag := range AllFeatures {
		lines = append(lines,
			"// setup:feature:"+tag+":start",
			tag+"-CODE",
			"// setup:feature:"+tag+":end",
		)
	}
	lines = append(lines, "tail")
	return strings.Join(lines, "\n")
}

// stripWithBundle expands the bundle's deps + implicits, computes the
// remove-tag set the way internal/setup uses it, and returns the stripped
// content for representativeBundleContent.
func stripWithBundle(t *testing.T, bundle []string) string {
	t.Helper()
	expanded := ExpandFeatureDeps(bundle)
	keep := make(map[string]bool, len(expanded)+len(ImplicitFeatures))
	for _, f := range expanded {
		keep[f] = true
	}
	for _, f := range ImplicitFeatures {
		keep[f] = true
	}
	remove := make(map[string]bool)
	for _, f := range AllFeatures {
		if !keep[f] {
			remove[f] = true
		}
	}
	return stripBlocks(representativeBundleContent(t), remove)
}

// assertSurvived checks that every tag in survivors has its CODE marker in
// got, and every tag in stripped does not. Other tags are not asserted.
func assertBundleSurvival(t *testing.T, got string, survivors, stripped []string) {
	t.Helper()
	for _, tag := range survivors {
		require.Contains(t, got, tag+"-CODE",
			"bundle should keep %q's code blocks", tag)
	}
	for _, tag := range stripped {
		require.NotContains(t, got, tag+"-CODE",
			"bundle should strip %q's code blocks", tag)
	}
}

func TestRepresentativeBundle_Minimal(t *testing.T) {
	got := stripWithBundle(t, []string{})
	// Minimal selects nothing. Every user-facing feature plus the hidden
	// database tag strips out.
	assertBundleSurvival(t, got,
		nil,
		[]string{FeatureAuth, FeatureGraph, FeatureDatabase, FeatureMSSQL,
			FeaturePostgres, FeatureSSE, FeatureCaddy, FeatureAvatar,
			FeatureDemo, FeatureSessionSettings, FeatureCapacitor, FeatureCSRF},
	)
	require.Contains(t, got, "preamble")
	require.Contains(t, got, "tail")
}

func TestRepresentativeBundle_Public(t *testing.T) {
	// "public" preset from mage_setup.go: sessions + SSE. SSE pulls the
	// hidden caddy tag in via featureDeps.
	got := stripWithBundle(t, []string{FeatureSessionSettings, FeatureSSE})
	assertBundleSurvival(t, got,
		[]string{FeatureSessionSettings, FeatureSSE, FeatureCaddy},
		[]string{FeatureAuth, FeatureGraph, FeatureDatabase, FeatureMSSQL,
			FeaturePostgres, FeatureAvatar, FeatureDemo, FeatureCapacitor, FeatureCSRF},
	)
}

func TestRepresentativeBundle_Internal(t *testing.T) {
	// "internal" preset: auth, CSRF, sessions, SSE. SSE pulls Caddy in via
	// featureDeps. No MSSQL/Postgres → no database. No avatar → no graph.
	got := stripWithBundle(t, []string{FeatureAuth, FeatureCSRF,
		FeatureSessionSettings, FeatureSSE})
	assertBundleSurvival(t, got,
		[]string{FeatureAuth, FeatureCSRF, FeatureSessionSettings,
			FeatureSSE, FeatureCaddy},
		[]string{FeatureGraph, FeatureDatabase, FeatureMSSQL, FeaturePostgres,
			FeatureAvatar, FeatureDemo, FeatureCapacitor},
	)
}

func TestRepresentativeBundle_MicrosoftInternal(t *testing.T) {
	// "microsoft-internal" preset: sessions, CSRF, auth, Graph, avatar, MSSQL,
	// SSE. Avatar pulls Graph (already explicit), MSSQL pulls the hidden
	// database tag, SSE pulls the hidden caddy tag.
	got := stripWithBundle(t, []string{FeatureSessionSettings, FeatureCSRF,
		FeatureAuth, FeatureGraph, FeatureAvatar, FeatureMSSQL, FeatureSSE})
	assertBundleSurvival(t, got,
		[]string{FeatureSessionSettings, FeatureCSRF, FeatureAuth,
			FeatureGraph, FeatureAvatar, FeatureMSSQL, FeatureDatabase,
			FeatureSSE, FeatureCaddy},
		[]string{FeaturePostgres, FeatureDemo, FeatureCapacitor},
	)
}

// TestRepresentativeBundle_AvatarPullsGraph guards the Avatar→Graph hard dep
// against regression at the bundle level: a bundle that names Avatar without
// Graph must still keep Graph (avatar.go imports the Graph package directly).
func TestRepresentativeBundle_AvatarPullsGraph(t *testing.T) {
	got := stripWithBundle(t, []string{FeatureAvatar})
	assertBundleSurvival(t, got,
		[]string{FeatureAvatar, FeatureGraph},
		[]string{FeatureDatabase, FeatureMSSQL, FeaturePostgres,
			FeatureSSE, FeatureCaddy, FeatureDemo, FeatureSessionSettings,
			FeatureCapacitor, FeatureCSRF, FeatureAuth},
	)
}

// TestRepresentativeBundle_SSEPullsCaddy checks that SSE implies the hidden
// Caddy tag through featureDeps, so the Caddyfile + install path ship
// alongside the SSE broker.
func TestRepresentativeBundle_SSEPullsCaddy(t *testing.T) {
	got := stripWithBundle(t, []string{FeatureSSE})
	assertBundleSurvival(t, got,
		[]string{FeatureSSE, FeatureCaddy},
		[]string{FeatureDatabase, FeatureMSSQL, FeaturePostgres,
			FeatureDemo, FeatureSessionSettings,
			FeatureCapacitor, FeatureCSRF, FeatureAuth,
			FeatureGraph, FeatureAvatar},
	)
}

// TestRepresentativeBundle_CaddyStripsWithoutSSE pairs with SSEPullsCaddy: a
// bundle that does NOT name SSE strips the hidden Caddy tag too, even if some
// other unrelated feature is selected.
func TestRepresentativeBundle_CaddyStripsWithoutSSE(t *testing.T) {
	got := stripWithBundle(t, []string{FeatureAuth, FeatureCSRF, FeatureSessionSettings})
	assertBundleSurvival(t, got,
		[]string{FeatureAuth, FeatureCSRF, FeatureSessionSettings},
		[]string{FeatureSSE, FeatureCaddy, FeatureDatabase,
			FeatureMSSQL, FeaturePostgres, FeatureDemo,
			FeatureCapacitor, FeatureGraph, FeatureAvatar},
	)
}

// TestLinkRelationsIsNotAFeatureTag checks that the link-relations registry
// stays baseline framework behavior instead of returning as a strip tag.
// AllFeatures must not advertise it.
func TestLinkRelationsIsNotAFeatureTag(t *testing.T) {
	for _, tag := range AllFeatures {
		require.NotEqual(t, "link_relations", tag,
			"link_relations should remain baseline behavior, not a user-facing feature tag")
	}
}
