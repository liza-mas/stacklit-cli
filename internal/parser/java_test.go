package parser

import (
	"os"
	"testing"
)

func TestJavaParserParse_Imports(t *testing.T) {
	p := &TreeSitterParser{}

	content, err := os.ReadFile("../../testdata/java-project/src/Main.java")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	info, err := p.Parse("../../testdata/java-project/src/Main.java", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if info.Language != "java" {
		t.Errorf("Language = %q, want %q", info.Language, "java")
	}

	wantImports := []string{"java.util", "com.example.service"}
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

func TestJavaParserParse_Exports(t *testing.T) {
	p := &TreeSitterParser{}

	content, err := os.ReadFile("../../testdata/java-project/src/Main.java")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	info, err := p.Parse("../../testdata/java-project/src/Main.java", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	wantExports := []string{"Main"}
	for _, want := range wantExports {
		found := false
		for _, got := range info.Exports {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("export %q not found in %v", want, info.Exports)
		}
	}
}

func TestJavaParserParse_Entrypoint(t *testing.T) {
	p := &TreeSitterParser{}

	content, err := os.ReadFile("../../testdata/java-project/src/Main.java")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	info, err := p.Parse("../../testdata/java-project/src/Main.java", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !info.IsEntrypoint {
		t.Error("IsEntrypoint = false, want true for file with public static void main")
	}
}

func TestJavaParserParse_NoEntrypoint(t *testing.T) {
	p := &TreeSitterParser{}

	content := []byte(`package com.example;

public class UserService {
    public String getUser(int id) {
        return "user";
    }
}
`)

	info, err := p.Parse("UserService.java", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if info.IsEntrypoint {
		t.Error("IsEntrypoint = true, want false for file without main method")
	}
}
