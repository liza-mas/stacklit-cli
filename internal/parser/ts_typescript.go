package parser

import (
	"strings"

	gts "github.com/odvcencio/gotreesitter"
)

// extractTypeScript extracts imports, exports, and type definitions from a
// TypeScript or JavaScript AST produced by tree-sitter.
func extractTypeScript(root *gts.Node, lang *gts.Language, src []byte, path string) *FileInfo {
	info := &FileInfo{
		Language: tsLangFromPath(path),
		TypeDefs: make(map[string]string),
	}

	seenImport := map[string]bool{}
	seenExport := map[string]bool{}

	addImport := func(s string) {
		s = strings.Trim(s, `"'`+"`")
		if s != "" && !seenImport[s] {
			seenImport[s] = true
			info.Imports = append(info.Imports, s)
		}
	}

	gts.Walk(root, func(node *gts.Node, depth int) gts.WalkAction {
		kind := node.Type(lang)

		switch kind {
		case "import_statement":
			// import ... from 'module' OR import 'module' (side-effect)
			src2 := childByType(node, lang, "string")
			if src2 != nil {
				addImport(nodeText(src2, src))
			}
			return gts.WalkSkipChildren

		case "call_expression":
			// require('module') or import('module')
			fn := childByType(node, lang, "identifier")
			if fn == nil {
				// Could be a member expression; handle via parent scan below.
				break
			}
			fnName := nodeText(fn, src)
			if fnName == "require" || fnName == "import" {
				args := childByType(node, lang, "arguments")
				if args != nil {
					for i := 0; i < args.ChildCount(); i++ {
						arg := args.Child(i)
						argKind := arg.Type(lang)
						if argKind == "string" || argKind == "template_string" {
							addImport(nodeText(arg, src))
							break
						}
					}
				}
			}

		case "export_statement":
			tsHandleExport(node, lang, src, info, seenExport)
			return gts.WalkSkipChildren
		}

		return gts.WalkContinue
	})

	info.Imports = dedup(info.Imports)
	info.Exports = dedup(info.Exports)

	if len(info.TypeDefs) == 0 {
		info.TypeDefs = nil
	}

	return info
}

// tsHandleExport processes an export_statement node to extract the exported
// name and signature, populating info.Exports and info.TypeDefs.
func tsHandleExport(node *gts.Node, lang *gts.Language, src []byte, info *FileInfo, seen map[string]bool) {
	addExport := func(name, sig string) {
		if name != "" && !seen[name] {
			seen[name] = true
			info.Exports = append(info.Exports, truncSig(sig, 80))
		}
	}

	// Walk direct children of the export_statement to find the declaration.
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		childKind := child.Type(lang)

		switch childKind {
		case "function_declaration", "generator_function_declaration":
			name := tsGetIdentifier(child, lang, src)
			if name == "" {
				break
			}
			params := tsGetParams(child, lang, src)
			ret := tsGetReturnType(child, lang, src)
			sig := name + "(" + params + ")"
			if ret != "" {
				sig += " " + ret
			}
			addExport(name, sig)

		case "class_declaration":
			name := tsGetIdentifier(child, lang, src)
			if name == "" {
				name = tsGetTypeIdentifier(child, lang, src)
			}
			if name == "" {
				break
			}
			addExport(name, name)
			// Extract public method names for TypeDefs.
			methods := tsExtractClassMethods(child, lang, src)
			if len(methods) > 0 {
				info.TypeDefs[name] = strings.Join(methods, ", ")
			}

		case "interface_declaration":
			name := tsGetTypeIdentifier(child, lang, src)
			if name == "" {
				break
			}
			addExport(name, "interface "+name)
			// Extract field names for TypeDefs.
			fields := tsExtractInterfaceFields(child, lang, src)
			if len(fields) > 0 {
				info.TypeDefs[name] = strings.Join(fields, ", ")
			}

		case "type_alias_declaration":
			name := tsGetTypeIdentifier(child, lang, src)
			if name == "" {
				break
			}
			addExport(name, "type "+name)
			// Extract fields if it's an object type.
			fields := tsExtractTypeAliasFields(child, lang, src)
			if len(fields) > 0 {
				info.TypeDefs[name] = strings.Join(fields, ", ")
			}

		case "enum_declaration":
			name := tsGetIdentifier(child, lang, src)
			if name == "" {
				break
			}
			addExport(name, "enum "+name)

		case "lexical_declaration", "variable_declaration":
			// export const/let/var name = ...
			// May have multiple declarators; extract each.
			for j := 0; j < child.ChildCount(); j++ {
				decl := child.Child(j)
				if decl.Type(lang) == "variable_declarator" {
					nameNode := childByType(decl, lang, "identifier")
					if nameNode == nil {
						nameNode = childByType(decl, lang, "type_identifier")
					}
					if nameNode != nil {
						name := nodeText(nameNode, src)
						addExport(name, name)
					}
				}
			}

		case "abstract_class_declaration":
			name := tsGetIdentifier(child, lang, src)
			if name == "" {
				break
			}
			addExport(name, name)
			methods := tsExtractClassMethods(child, lang, src)
			if len(methods) > 0 {
				info.TypeDefs[name] = strings.Join(methods, ", ")
			}

		case "export_clause":
			// export { Foo, Bar } — named re-exports or local re-exports.
			for j := 0; j < child.ChildCount(); j++ {
				spec := child.Child(j)
				if spec.Type(lang) == "export_specifier" {
					nameNode := childByType(spec, lang, "identifier")
					if nameNode != nil {
						name := nodeText(nameNode, src)
						addExport(name, name)
					}
				}
			}

		case "namespace_export":
			// export * as Namespace from '...'
			nameNode := childByType(child, lang, "identifier")
			if nameNode != nil {
				name := nodeText(nameNode, src)
				addExport(name, name)
			}
		}
	}
}

// tsGetIdentifier finds the first identifier child of a node and returns its text.
func tsGetIdentifier(node *gts.Node, lang *gts.Language, src []byte) string {
	n := childByType(node, lang, "identifier")
	if n == nil {
		return ""
	}
	return nodeText(n, src)
}

// tsGetTypeIdentifier finds the first type_identifier child and returns its text.
func tsGetTypeIdentifier(node *gts.Node, lang *gts.Language, src []byte) string {
	n := childByType(node, lang, "type_identifier")
	if n == nil {
		return ""
	}
	return nodeText(n, src)
}

// tsGetParams returns a comma-separated parameter list from the formal_parameters child.
func tsGetParams(node *gts.Node, lang *gts.Language, src []byte) string {
	params := childByType(node, lang, "formal_parameters")
	if params == nil {
		return ""
	}
	// Return the raw text of the params node, minus surrounding parens.
	raw := nodeText(params, src)
	raw = strings.TrimPrefix(raw, "(")
	raw = strings.TrimSuffix(raw, ")")
	raw = strings.TrimSpace(raw)
	// Collapse whitespace.
	return strings.Join(strings.Fields(raw), " ")
}

// tsGetReturnType returns the return type annotation text if present.
func tsGetReturnType(node *gts.Node, lang *gts.Language, src []byte) string {
	rt := childByType(node, lang, "type_annotation")
	if rt == nil {
		return ""
	}
	raw := nodeText(rt, src)
	// type_annotation text looks like ": ReturnType"
	raw = strings.TrimPrefix(raw, ":")
	return strings.TrimSpace(raw)
}

// tsExtractClassMethods returns public method names from a class body.
func tsExtractClassMethods(classNode *gts.Node, lang *gts.Language, src []byte) []string {
	body := childByType(classNode, lang, "class_body")
	if body == nil {
		return nil
	}
	var methods []string
	seen := map[string]bool{}
	for i := 0; i < body.ChildCount(); i++ {
		member := body.Child(i)
		memberKind := member.Type(lang)
		if memberKind == "method_definition" || memberKind == "abstract_method_signature" {
			// Skip private/protected members (those starting with # or having accessibility modifier).
			access := childByType(member, lang, "accessibility_modifier")
			if access != nil {
				accessText := nodeText(access, src)
				if accessText == "private" || accessText == "protected" {
					continue
				}
			}
			nameNode := childByType(member, lang, "property_identifier")
			if nameNode == nil {
				nameNode = childByType(member, lang, "identifier")
			}
			if nameNode != nil {
				name := nodeText(nameNode, src)
				// Skip private fields (#name).
				if strings.HasPrefix(name, "#") {
					continue
				}
				if !seen[name] {
					seen[name] = true
					methods = append(methods, name)
				}
			}
		}
	}
	return methods
}

// tsExtractInterfaceFields returns field names from an interface body.
func tsExtractInterfaceFields(ifaceNode *gts.Node, lang *gts.Language, src []byte) []string {
	body := childByType(ifaceNode, lang, "object_type")
	if body == nil {
		return nil
	}
	var fields []string
	seen := map[string]bool{}
	for i := 0; i < body.ChildCount(); i++ {
		member := body.Child(i)
		memberKind := member.Type(lang)
		switch memberKind {
		case "property_signature", "method_signature", "index_signature":
			nameNode := childByType(member, lang, "property_identifier")
			if nameNode == nil {
				nameNode = childByType(member, lang, "identifier")
			}
			if nameNode != nil {
				name := nodeText(nameNode, src)
				if !seen[name] {
					seen[name] = true
					fields = append(fields, name)
				}
			}
		}
	}
	return fields
}

// tsExtractTypeAliasFields returns field names if a type alias is an object type.
func tsExtractTypeAliasFields(typeNode *gts.Node, lang *gts.Language, src []byte) []string {
	// type_alias_declaration: type Name = TypeValue
	// Find the object_type child if present.
	var result []string
	for i := 0; i < typeNode.ChildCount(); i++ {
		child := typeNode.Child(i)
		if child.Type(lang) == "object_type" {
			seen := map[string]bool{}
			for j := 0; j < child.ChildCount(); j++ {
				member := child.Child(j)
				memberKind := member.Type(lang)
				if memberKind == "property_signature" || memberKind == "method_signature" {
					nameNode := childByType(member, lang, "property_identifier")
					if nameNode == nil {
						nameNode = childByType(member, lang, "identifier")
					}
					if nameNode != nil {
						name := nodeText(nameNode, src)
						if !seen[name] {
							seen[name] = true
							result = append(result, name)
						}
					}
				}
			}
			break
		}
	}
	return result
}

// tsLangFromPath returns the stacklit language name for a TS/JS file.
func tsLangFromPath(path string) string {
	if extIs(path, ".tsx") {
		return "tsx"
	}
	if extIs(path, ".jsx") {
		return "jsx"
	}
	if extIs(path, ".ts", ".mts", ".cts") {
		return "typescript"
	}
	return "javascript"
}
