package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"schema-mapper-cli/internal/convert"
	"schema-mapper-cli/internal/mapping"
	"schema-mapper-cli/internal/schema"
)

func usage() {
	fmt.Fprintln(os.Stderr, "usage: schema-mapper-cli <command> [flags]")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  parse <file>              parse a schema file (json/csv/sql by extension)")
	fmt.Fprintln(os.Stderr, "  diff <a.json> <b.json>    show diff between two JSON schemas")
	fmt.Fprintln(os.Stderr, "  suggest <src> <dst>       suggest mapping rules between two schemas")
	fmt.Fprintln(os.Stderr, "  convert -src <csv> -map <rules.json> [-out <out.json>]")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var err error
	switch cmd {
	case "parse":
		err = cmdParse(args)
	case "diff":
		err = cmdDiff(args)
	case "suggest":
		err = cmdSuggest(args)
	case "convert":
		err = cmdConvert(args)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func loadSchema(path string) (*schema.Schema, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return schema.ParseJSONSchema(strings.NewReader(string(data)))
	case ".csv":
		lines := strings.Split(string(data), "\n")
		if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
			return nil, fmt.Errorf("csv has no header")
		}
		header := strings.Split(strings.TrimSpace(lines[0]), ",")
		return schema.ParseCSVHeader(header), nil
	case ".sql", ".ddl":
		return schema.ParseDDL(string(data))
	default:
		return schema.ParseJSONSchema(strings.NewReader(string(data)))
	}
}

func cmdParse(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("parse requires <file>")
	}
	s, err := loadSchema(args[0])
	if err != nil {
		return err
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(s)
}

func cmdDiff(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("diff requires <a.json> <b.json>")
	}
	a, err := loadSchema(args[0])
	if err != nil {
		return err
	}
	b, err := loadSchema(args[1])
	if err != nil {
		return err
	}
	d := schema.Diff(a, b)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(d)
}

func cmdSuggest(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("suggest requires <src> <dst>")
	}
	src, err := loadSchema(args[0])
	if err != nil {
		return err
	}
	dst, err := loadSchema(args[1])
	if err != nil {
		return err
	}
	rules := mapping.Suggest(src, dst, 4)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rules)
}

func cmdConvert(args []string) error {
	var srcPath, mapPath, outPath string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-src":
			if i+1 < len(args) {
				srcPath = args[i+1]
				i++
			}
		case "-map":
			if i+1 < len(args) {
				mapPath = args[i+1]
				i++
			}
		case "-out":
			if i+1 < len(args) {
				outPath = args[i+1]
				i++
			}
		}
	}
	if srcPath == "" || mapPath == "" {
		return fmt.Errorf("convert requires -src and -map")
	}
	srcData, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	recs, _, err := convert.ReadCSV(strings.NewReader(string(srcData)))
	if err != nil {
		return err
	}
	mapData, err := os.ReadFile(mapPath)
	if err != nil {
		return err
	}
	var rules []mapping.Rule
	if err := json.Unmarshal(mapData, &rules); err != nil {
		return err
	}
	dst := &schema.Schema{}
	for _, r := range rules {
		dst.Fields = append(dst.Fields, schema.Field{Name: r.Target, Type: schema.TString, Optional: true})
	}
	out, err := convert.RecordsToTarget(recs, rules, dst)
	if err != nil {
		return err
	}
	var w io.Writer = os.Stdout
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
