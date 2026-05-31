package parser

import (
	"os"
	"strings"
	"testing"
)

func TestTypeScriptParserParse_ImportsESM(t *testing.T) {
	p := &TreeSitterParser{}

	content, err := os.ReadFile("../../testdata/ts-project/src/index.ts")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	info, err := p.Parse("../../testdata/ts-project/src/index.ts", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if info.Language != "typescript" {
		t.Errorf("Language = %q, want %q", info.Language, "typescript")
	}

	wantImports := []string{"express", "./router", "./types"}
	for _, want := range wantImports {
		found := false
		for _, got := range info.Imports {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("import %q not found in %v", want, info.Imports)
		}
	}
}

func TestTypeScriptParserParse_ImportsCJS(t *testing.T) {
	p := &TreeSitterParser{}

	content, err := os.ReadFile("../../testdata/ts-project/src/router.ts")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	info, err := p.Parse("../../testdata/ts-project/src/router.ts", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Should find both ESM import and CJS require.
	foundESM := false
	foundCJS := false
	for _, imp := range info.Imports {
		if imp == "express" {
			foundESM = true
		}
		if imp == "./middleware" {
			foundCJS = true
		}
	}

	if !foundESM {
		t.Errorf("ESM import %q not found in %v", "express", info.Imports)
	}
	if !foundCJS {
		t.Errorf("CJS require %q not found in %v", "./middleware", info.Imports)
	}
}

func TestTypeScriptParserParse_Exports(t *testing.T) {
	p := &TreeSitterParser{}

	content, err := os.ReadFile("../../testdata/ts-project/src/router.ts")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	info, err := p.Parse("../../testdata/ts-project/src/router.ts", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Exports are now full signatures containing the name.
	wantExports := []string{"Router", "RouteHandler"}
	for _, want := range wantExports {
		found := false
		for _, got := range info.Exports {
			if strings.Contains(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("export %q not found in %v", want, info.Exports)
		}
	}
}

func TestTypeScriptParserParse_InlineContent(t *testing.T) {
	p := &TreeSitterParser{}

	content := []byte(`import React from 'react';
import { useState } from 'react';
import type { FC } from 'react';
const path = require('path');

export function MyComponent() {}
export const myValue = 42;
export type MyType = string;
export interface MyInterface {}
export class MyClass {}
export enum MyEnum { A, B }
`)

	info, err := p.Parse("test.tsx", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// 'react' should appear once (deduplicated).
	reactCount := 0
	for _, imp := range info.Imports {
		if imp == "react" {
			reactCount++
		}
	}
	if reactCount != 1 {
		t.Errorf("'react' import count = %d, want 1 (deduplicated)", reactCount)
	}

	// Exports are now full signatures containing the name.
	wantExports := []string{"MyComponent", "myValue", "MyType", "MyInterface", "MyClass", "MyEnum"}
	for _, want := range wantExports {
		found := false
		for _, got := range info.Exports {
			if strings.Contains(got, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("export %q not found in %v", want, info.Exports)
		}
	}
}
