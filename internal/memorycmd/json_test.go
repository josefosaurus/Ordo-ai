package memorycmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// documentsRecord mirrors one record of a golden dataset's documents.json:
// no title or content fields, nested objects, scalar arrays, and an array of
// objects.
const documentsRecord = `{
  "id": "AUTH-001",
  "titulo": "Inicio de sesion con RUT y clave",
  "dominio": "auth",
  "content_type": "flujo-funcional",
  "obsoleto": false,
  "version": 2,
  "modulos": ["login"],
  "tags": ["auth", "critica"],
  "notas": null,
  "vacio": "",
  "lista_vacia": [],
  "objeto_vacio": {},
  "punto_entrada": {
    "descripcion": "LoginActivity",
    "archivo": "login/LoginActivity.kt",
    "detalle": {"linea": 42}
  },
  "pasos": [
    {"orden": 1, "pantalla": "LoginView", "clase": "LoginEventManager.onProceed"},
    {"orden": 2, "pantalla": "envio", "extra": {"nota": "x"}, "refs": ["a", "b"]}
  ]
}`

const documentsContent = `Dominio: auth
Content type: flujo-funcional
Obsoleto: false
Version: 2
Modulos: login
Tags: auth, critica
Punto entrada:
  Descripcion: LoginActivity
  Archivo: login/LoginActivity.kt
  Detalle:
    Linea: 42
Pasos:
  1. orden: 1
     pantalla: LoginView
     clase: LoginEventManager.onProceed
  2. orden: 2
     pantalla: envio
     extra:
       nota: x
     refs: a, b`

func collectOne(t *testing.T, file, content string) []Entry {
	t.Helper()
	path := filepath.Join(t.TempDir(), file)
	writeFile(t, path, content)
	entries, err := Collect(path)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	return entries
}

func TestCollectJSONRendersDatasetRecords(t *testing.T) {
	entries := collectOne(t, "documents.json", "["+documentsRecord+"]")
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	e := entries[0]
	if e.Title != "AUTH-001 — Inicio de sesion con RUT y clave" {
		t.Errorf("title = %q", e.Title)
	}
	if e.Content != documentsContent {
		t.Errorf("content =\n%s\nwant\n%s", e.Content, documentsContent)
	}
	if e.Type != "" || e.Source != "documents.json" {
		t.Errorf("type/source = %q/%q", e.Type, e.Source)
	}
}

func TestCollectJSONMatchesJSONL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.jsonl"), "{\"title\":\"One\",\"content\":\" c1 \",\"type\":\"decision\"}\n{\"title\":\"Two\",\"content\":\"c2\"}\n")
	writeFile(t, filepath.Join(dir, "b.json"), `[{"title":"One","content":" c1 ","type":"decision"},{"title":"Two","content":"c2"}]`)
	jsonl, err := Collect(filepath.Join(dir, "a.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	js, err := Collect(filepath.Join(dir, "b.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(js) != len(jsonl) {
		t.Fatalf("json entries = %d, jsonl = %d", len(js), len(jsonl))
	}
	for i := range js {
		if js[i].Title != jsonl[i].Title || js[i].Content != jsonl[i].Content || js[i].Type != jsonl[i].Type {
			t.Errorf("entry %d: json %+v != jsonl %+v", i, js[i], jsonl[i])
		}
	}
}

func TestCollectJSONTitles(t *testing.T) {
	tests := []struct {
		name, content, want string
	}{
		{"object holding one array", `{"documents":[{"title":"T","content":"c"}],"version":1}`, "T"},
		{"numeric id with name", `[{"id":7,"name":"Seven","x":"y"}]`, "7 — Seven"},
		{"numeric id alone", `[{"id":7,"x":"y"}]`, "7"},
		{"id equal to title", `[{"id":"Same","title":"Same","x":"y"}]`, "Same"},
		{"title wins over titulo and name", `[{"name":"N","titulo":"Ti","title":"T","x":"y"}]`, "T"},
		{"empty title falls back to titulo", `[{"title":" ","titulo":"Ti","x":"y"}]`, "Ti"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := collectOne(t, "x.json", tt.content)
			if got := strings.Join(titles(entries), "|"); got != tt.want {
				t.Fatalf("titles = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCollectJSONRenderExcludesTitleFieldsAndType(t *testing.T) {
	entries := collectOne(t, "x.json", `[{"name":"N","titulo":"Ti","id":"I","type":"decision","content":"","x-y":"z"}]`)
	e := entries[0]
	if e.Title != "I — Ti" || e.Type != "decision" {
		t.Fatalf("entry = %+v", e)
	}
	if e.Content != "Name: N\nX y: z" {
		t.Fatalf("content = %q", e.Content)
	}
}

func TestCollectJSONRendersNonStringContent(t *testing.T) {
	entries := collectOne(t, "x.json", `[
{"id":"A","content":{"steps":["one","two"]},"note":"n"},
{"id":"B","content":["x","y"]},
{"id":"C","content":42},
{"id":"D","content":"   ","note":"kept"}]`)
	want := []string{
		"Content:\n  Steps: one, two\nNote: n",
		"Content: x, y",
		"Content: 42",
		"Note: kept",
	}
	for i, w := range want {
		if entries[i].Content != w {
			t.Errorf("%s content = %q, want %q", entries[i].Title, entries[i].Content, w)
		}
	}
}

func TestCollectJSONErrors(t *testing.T) {
	tests := []struct {
		name, content, wantErr string
	}{
		{"scalar top level", `42`, "x.json: expected an array of objects or an object holding one array"},
		{"object without arrays", `{"a":1}`, "x.json: expected an array of objects or an object holding one array"},
		{"object with two arrays", `{"a":[],"b":[]}`, "x.json: expected an array of objects or an object holding one array"},
		{"non-object element", `[{"title":"T","content":"c"},"oops"]`, "x.json: record 2: not an object"},
		{"no title fields", `[{"x":"y"}]`, "x.json: record 1: no title field (title, titulo, name or id); use --title-field"},
		{"empty rendered content", `[{"title":"T","notes":null}]`, `x.json: record 1: entry "T": content is empty`},
		{"duplicate titles", `[{"title":"T","content":"a"},{"title":"T","content":"b"}]`, `x.json: duplicate entry "T"`},
		{"invalid json", `[{"title":`, "x.json:"},
		{"trailing data", `[] []`, "x.json:"},
		{"empty array", `[]`, "no entries found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "x.json")
			writeFile(t, path, tt.content)
			_, err := Collect(path)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCollectJSONTitleField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.json")
	writeFile(t, path, `[{"code":12,"title":"Ignored","body":"b"}]`)
	entries, err := collect(path, collectOptions{titleField: "code"})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if entries[0].Title != "12" || entries[0].Content != "Title: Ignored\nBody: b" {
		t.Fatalf("entry = %+v", entries[0])
	}
	if _, err := collect(path, collectOptions{titleField: "missing"}); err == nil ||
		!strings.Contains(err.Error(), `x.json: record 1: title field "missing" is missing or empty`) {
		t.Fatalf("err = %v", err)
	}
}

func TestRunDryRunHonorsTitleField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	writeFile(t, path, `{"items":[{"slug":"first","body":"one"},{"slug":"second","body":"two"}]}`)
	var out bytes.Buffer
	if err := Run([]string{"import", path, "--project", "demo", "--title-field", "slug", "--dry-run"}, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, want := range []string{"  first  (data.json)", "  second  (data.json)", "2 entries (dry run, nothing written)"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("dry-run output missing %q:\n%s", want, out.String())
		}
	}
}

func TestCollectDirectoryIgnoresJSONByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "## A\na\n")
	writeFile(t, filepath.Join(dir, "data.json"), `[{"id":"D-1","content":"d"}]`)
	writeFile(t, filepath.Join(dir, "manifest.json"), `{"icons":[{"src":"a.png"}]}`)
	writeFile(t, filepath.Join(dir, "tsconfig.json"), "{\n  // comment\n}")
	entries, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if got := strings.Join(titles(entries), "|"); got != "A" {
		t.Fatalf("titles = %q", got)
	}
}

func TestCollectDirectoryIncludesJSONWhenAsked(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "## A\na\n")
	writeFile(t, filepath.Join(dir, "sub", "b.json"), `[{"id":"B-1","content":"b"}]`)
	entries, err := collect(dir, collectOptions{includeJSON: true})
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	if got := strings.Join(titles(entries), "|"); got != "A|B-1" {
		t.Fatalf("titles = %q", got)
	}
	if entries[1].Source != "sub/b.json" {
		t.Errorf("source = %q", entries[1].Source)
	}
}

func TestCollectDirectoryIncludedJSONMustBeDatasets(t *testing.T) {
	for name, body := range map[string]string{
		"package.json": `{"name":"x","files":["dist"],"keywords":["k"]}`,
		"broken.json":  `[{"id":"D-1","content":"d"}`,
		"data.json":    `[{"id":"D-1","content":"d"},{"content":"no title"}]`,
	} {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, name), body)
		if _, err := collect(dir, collectOptions{includeJSON: true}); err == nil || !strings.HasPrefix(err.Error(), name+": ") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestCollectSingleJSONFileNeedsNoFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "documents.json")
	writeFile(t, path, `[{"id":"D-1","content":"d"}]`)
	entries, err := Collect(path)
	if err != nil || len(entries) != 1 {
		t.Fatalf("Collect = %d entries, %v", len(entries), err)
	}
}

func TestRunDryRunIncludeJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "## A\na\n")
	writeFile(t, filepath.Join(dir, "b.json"), `[{"id":"B-1","content":"b"}]`)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{nil, "1 entries"},
		{[]string{"--include-json"}, "2 entries"},
	} {
		var out bytes.Buffer
		args := append([]string{"import", dir, "--project", "demo", "--dry-run"}, tc.args...)
		if err := Run(args, &out); err != nil {
			t.Fatalf("Run %v: %v", tc.args, err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Errorf("Run %v output missing %q:\n%s", tc.args, tc.want, out.String())
		}
	}
}

func TestCollectUnsupportedListsJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.txt")
	writeFile(t, path, "hi")
	if _, err := Collect(path); err == nil || !strings.Contains(err.Error(), ".jsonl or .json") {
		t.Fatalf("err = %v", err)
	}
}
