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

func TestImplicitFeaturesAlwaysKept(t *testing.T) {
	// When Features is set but does not include "database",
	// database should still be kept because it's implicit.
	content := strings.Join([]string{
		"before",
		"// setup:feature:database:start",
		"database code",
		"// setup:feature:database:end",
		"after",
	}, "\n")
	// Simulate removeOptionalContent logic: build removeTags with implicit features kept
	removeTags := make(map[string]bool)
	keep := make(map[string]bool)
	// User selected no features
	for _, f := range ImplicitFeatures {
		keep[f] = true
	}
	for _, f := range AllFeatures {
		if !keep[f] {
			removeTags[f] = true
		}
	}
	got := stripBlocks(content, removeTags)
	require.Contains(t, got, "database code", "database blocks should be kept (implicit feature)")
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
