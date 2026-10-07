package memorycmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxJSONDepth bounds nesting so a pathological .json file fails cleanly
// instead of recursing without limit.
const maxJSONDepth = 1000

// jsonKind classifies one decoded JSON value.
type jsonKind int

const (
	jsonNull jsonKind = iota
	jsonString
	jsonNumber
	jsonBool
	jsonObject
	jsonArray
)

// jsonValue is a decoded JSON value that keeps object keys in source order,
// which Go maps would lose.
type jsonValue struct {
	kind   jsonKind
	text   string // string, number, or bool as written
	fields []jsonField
	items  []jsonValue
}

type jsonField struct {
	key   string
	value jsonValue
}

// parseJSON reads a .json dataset: an array of objects, or an object holding
// exactly one array of objects. Each record becomes one entry; see jsonEntry.
func parseJSON(data []byte, titleField string) ([]Entry, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	top, err := decodeJSONValue(dec, 0)
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		// Empty or truncated input; a bare "EOF" would not say what went wrong.
		return nil, errors.New("unexpected end of JSON input")
	}
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			err = errors.New("unexpected data after the top-level value")
		}
		return nil, err
	}
	records, ok := jsonRecords(top)
	if !ok {
		return nil, errors.New("expected an array of objects or an object holding one array")
	}
	entries := make([]Entry, 0, len(records))
	for i, rec := range records {
		if rec.kind != jsonObject {
			return nil, fmt.Errorf("record %d: not an object", i+1)
		}
		e, err := jsonEntry(rec, titleField)
		if err == nil {
			err = validate(e)
		}
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", i+1, err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// decodeJSONValue reads the next value from dec as an ordered tree.
func decodeJSONValue(dec *json.Decoder, depth int) (jsonValue, error) {
	if depth > maxJSONDepth {
		return jsonValue{}, fmt.Errorf("nesting exceeds %d levels", maxJSONDepth)
	}
	tok, err := dec.Token()
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return jsonValue{}, err
	}
	switch t := tok.(type) {
	case json.Delim:
		v := jsonValue{kind: jsonArray}
		if t == '{' {
			v.kind = jsonObject
		}
		for dec.More() {
			var key string
			if v.kind == jsonObject {
				keyTok, err := dec.Token()
				if err != nil {
					return jsonValue{}, err
				}
				key, _ = keyTok.(string)
			}
			child, err := decodeJSONValue(dec, depth+1)
			if err != nil {
				return jsonValue{}, err
			}
			if v.kind == jsonObject {
				v.fields = append(v.fields, jsonField{key: key, value: child})
			} else {
				v.items = append(v.items, child)
			}
		}
		if _, err := dec.Token(); err != nil { // closing delimiter
			return jsonValue{}, err
		}
		return v, nil
	case string:
		return jsonValue{kind: jsonString, text: t}, nil
	case json.Number:
		return jsonValue{kind: jsonNumber, text: t.String()}, nil
	case bool:
		return jsonValue{kind: jsonBool, text: strconv.FormatBool(t)}, nil
	default:
		return jsonValue{kind: jsonNull}, nil
	}
}

// jsonRecords returns the record list of a dataset's top-level value.
func jsonRecords(top jsonValue) ([]jsonValue, bool) {
	switch top.kind {
	case jsonArray:
		return top.items, true
	case jsonObject:
		var records []jsonValue
		arrays := 0
		for _, f := range top.fields {
			if f.value.kind == jsonArray {
				arrays++
				records = f.value.items
			}
		}
		return records, arrays == 1
	}
	return nil, false
}

// field returns the first value stored under key in an object.
func (v jsonValue) field(key string) (jsonValue, bool) {
	for _, f := range v.fields {
		if f.key == key {
			return f.value, true
		}
	}
	return jsonValue{}, false
}

// stringField returns the trimmed string under key, or "".
func (v jsonValue) stringField(key string) string {
	if f, ok := v.field(key); ok && f.kind == jsonString {
		return strings.TrimSpace(f.text)
	}
	return ""
}

// scalarField returns the trimmed string or number under key, or "".
func (v jsonValue) scalarField(key string) string {
	if f, ok := v.field(key); ok && (f.kind == jsonString || f.kind == jsonNumber) {
		return strings.TrimSpace(f.text)
	}
	return ""
}

// jsonEntry builds one entry from a record. The title is titleField's value
// when set; otherwise the first of title, titulo, or name, prefixed by id as
// "<id> — <title>" when id is present and different. A non-empty string
// content is used as is; otherwise every other field (except the title fields
// and type) renders as labeled text in source order, including a non-string
// content, so no record data is dropped silently.
func jsonEntry(rec jsonValue, titleField string) (Entry, error) {
	skip := map[string]bool{"type": true}
	var title string
	if titleField != "" {
		title = rec.scalarField(titleField)
		if title == "" {
			return Entry{}, fmt.Errorf("title field %q is missing or empty", titleField)
		}
		skip[titleField] = true
	} else {
		var name string
		for _, key := range []string{"title", "titulo", "name"} {
			if name = rec.stringField(key); name != "" {
				skip[key] = true
				break
			}
		}
		id := rec.scalarField("id")
		if id != "" {
			skip["id"] = true
		}
		switch {
		case id != "" && name != "" && id != name:
			title = id + " — " + name
		case name != "":
			title = name
		case id != "":
			title = id
		default:
			return Entry{}, errors.New("no title field (title, titulo, name or id); use --title-field")
		}
	}
	e := Entry{Title: title, Type: rec.stringField("type"), Content: rec.stringField("content")}
	if e.Content == "" {
		e.Content = strings.Join(renderJSONFields(rec.fields, skip, true), "\n")
	}
	return e, nil
}

// renderJSONFields renders an object's fields as "Name: value" lines. With
// labels, keys become labels ("punto_entrada" -> "Punto entrada"); otherwise
// they are written as is. Null and empty values are skipped.
func renderJSONFields(fields []jsonField, skip map[string]bool, labels bool) []string {
	var lines []string
	for _, f := range fields {
		if skip[f.key] {
			continue
		}
		name := f.key
		if labels {
			name = jsonLabel(f.key)
		}
		lines = append(lines, renderJSONField(name, f.value, labels)...)
	}
	return lines
}

func renderJSONField(name string, v jsonValue, labels bool) []string {
	switch v.kind {
	case jsonObject:
		return nestJSON(name, renderJSONFields(v.fields, nil, labels))
	case jsonArray:
		if scalars, ok := jsonScalars(v.items); ok {
			if len(scalars) == 0 {
				return nil
			}
			return labeledJSON(name, strings.Join(scalars, ", "))
		}
		return nestJSON(name, renderJSONItems(v.items))
	}
	if s, ok := v.scalarText(); ok {
		return labeledJSON(name, s)
	}
	return nil
}

// labeledJSON renders "name: text"; continuation lines of a multi-line text
// are indented two spaces under the label line.
func labeledJSON(name, text string) []string {
	lines := strings.Split(name+": "+text, "\n")
	for i := 1; i < len(lines); i++ {
		lines[i] = "  " + lines[i]
	}
	return lines
}

// renderJSONItems numbers the non-empty items of a mixed or object array;
// an item's continuation lines align under its first line.
func renderJSONItems(items []jsonValue) []string {
	var lines []string
	n := 0
	for _, item := range items {
		var body []string
		switch item.kind {
		case jsonObject:
			body = renderJSONFields(item.fields, nil, false)
		case jsonArray:
			if scalars, ok := jsonScalars(item.items); ok {
				if len(scalars) > 0 {
					body = strings.Split(strings.Join(scalars, ", "), "\n")
				}
			} else {
				body = renderJSONItems(item.items)
			}
		default:
			if s, ok := item.scalarText(); ok {
				body = strings.Split(s, "\n")
			}
		}
		if len(body) == 0 {
			continue
		}
		n++
		prefix := strconv.Itoa(n) + ". "
		pad := strings.Repeat(" ", len(prefix))
		lines = append(lines, prefix+body[0])
		for _, l := range body[1:] {
			lines = append(lines, pad+l)
		}
	}
	return lines
}

// nestJSON puts child lines under a "name:" header, or drops an empty group.
func nestJSON(name string, child []string) []string {
	if len(child) == 0 {
		return nil
	}
	out := make([]string, 0, len(child)+1)
	out = append(out, name+":")
	for _, l := range child {
		out = append(out, "  "+l)
	}
	return out
}

// jsonScalars returns the non-empty texts of items when none is an object or
// an array.
func jsonScalars(items []jsonValue) ([]string, bool) {
	var out []string
	for _, item := range items {
		if item.kind == jsonObject || item.kind == jsonArray {
			return nil, false
		}
		if s, ok := item.scalarText(); ok {
			out = append(out, s)
		}
	}
	return out, true
}

// scalarText returns a scalar's text; null and blank strings report false.
func (v jsonValue) scalarText() (string, bool) {
	switch v.kind {
	case jsonString:
		s := strings.TrimSpace(v.text)
		return s, s != ""
	case jsonNumber, jsonBool:
		return v.text, true
	}
	return "", false
}

// jsonLabel turns a key into a label: "_" and "-" become spaces and the first
// rune is upper-cased.
func jsonLabel(key string) string {
	label := strings.NewReplacer("_", " ", "-", " ").Replace(key)
	r, size := utf8.DecodeRuneInString(label)
	if r == utf8.RuneError {
		return label
	}
	return string(unicode.ToUpper(r)) + label[size:]
}
