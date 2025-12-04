package pgexcel

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// template_loader.go - Template loading and validation

// LoadTemplate loads a report template from a YAML file
func LoadTemplate(path string) (*ReportTemplate, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening template file: %w", err)
	}
	defer file.Close()

	return LoadTemplateFromReader(file)
}

// LoadTemplateFromReader loads a template from an io.Reader
func LoadTemplateFromReader(r io.Reader) (*ReportTemplate, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading template: %w", err)
	}

	var template ReportTemplate
	if err := yaml.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("parsing YAML template: %w", err)
	}

	// Apply defaults and validate
	if err := template.applyDefaults(); err != nil {
		return nil, fmt.Errorf("applying defaults: %w", err)
	}

	if err := ValidateTemplate(&template); err != nil {
		return nil, fmt.Errorf("validating template: %w", err)
	}

	return &template, nil
}

// LoadTemplateFromString loads a template from a YAML string
func LoadTemplateFromString(yamlContent string) (*ReportTemplate, error) {
	return LoadTemplateFromReader(strings.NewReader(yamlContent))
}

// ValidateTemplate validates the template structure
func ValidateTemplate(t *ReportTemplate) error {
	if t == nil {
		return fmt.Errorf("template is nil")
	}

	if len(t.Sheets) == 0 {
		return fmt.Errorf("template must have at least one sheet")
	}

	for i, sheet := range t.Sheets {
		if err := validateSheet(&sheet, i); err != nil {
			return err
		}
	}

	return nil
}

func validateSheet(s *SheetTemplate, index int) error {
	if s.Name == "" {
		return fmt.Errorf("sheet[%d]: name is required", index)
	}

	if s.Query == "" && s.QueryFile == "" {
		return fmt.Errorf("sheet[%d] '%s': either query or query_file is required", index, s.Name)
	}

	if s.Query != "" && s.QueryFile != "" {
		return fmt.Errorf("sheet[%d] '%s': cannot specify both query and query_file", index, s.Name)
	}

	// Validate column names are unique
	colNames := make(map[string]bool)
	for j, col := range s.Columns {
		if col.Name == "" {
			return fmt.Errorf("sheet[%d] '%s' column[%d]: name is required", index, s.Name, j)
		}
		if colNames[col.Name] {
			return fmt.Errorf("sheet[%d] '%s': duplicate column name '%s'", index, s.Name, col.Name)
		}
		colNames[col.Name] = true
	}

	// Validate protection settings
	if s.Protection != nil {
		if err := validateProtection(s.Protection, s.Name); err != nil {
			return err
		}
	}

	return nil
}

func validateProtection(p *ProtectionTemplate, sheetName string) error {
	// Validate unlocked ranges format
	for _, rng := range p.UnlockedRanges {
		if !isValidCellRange(rng) {
			return fmt.Errorf("sheet '%s': invalid range format '%s' (expected A1:B10)", sheetName, rng)
		}
	}

	// Validate locked rows format
	for _, row := range p.LockedRows {
		if row != "header" && !isValidRowRange(row) {
			return fmt.Errorf("sheet '%s': invalid row range '%s' (expected number, range like 1-5, or 'header')", sheetName, row)
		}
	}

	return nil
}

// isValidCellRange checks if a string is a valid Excel range (e.g., "A1:B10")
func isValidCellRange(s string) bool {
	pattern := `^[A-Z]+\d+:[A-Z]+\d+$`
	matched, _ := regexp.MatchString(pattern, strings.ToUpper(s))
	return matched
}

// isValidRowRange checks if a string is a valid row range (e.g., "1", "1-5")
func isValidRowRange(s string) bool {
	// Single row number
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}

	// Row range like "1-5"
	parts := strings.Split(s, "-")
	if len(parts) == 2 {
		_, err1 := strconv.Atoi(parts[0])
		_, err2 := strconv.Atoi(parts[1])
		return err1 == nil && err2 == nil
	}

	return false
}

// applyDefaults applies default values to the template
func (t *ReportTemplate) applyDefaults() error {
	// Set version if not specified
	if t.Version == "" {
		t.Version = "1.0"
	}

	// Initialize variables map if nil
	if t.Variables == nil {
		t.Variables = make(map[string]string)
	}

	// Apply defaults to each sheet
	for i := range t.Sheets {
		if err := t.Sheets[i].applyDefaults(t.Defaults); err != nil {
			return fmt.Errorf("sheet '%s': %w", t.Sheets[i].Name, err)
		}
	}

	return nil
}

// applyDefaults applies template defaults to a sheet
func (s *SheetTemplate) applyDefaults(defaults *TemplateDefaults) error {
	// Initialize layout with defaults if not specified
	if s.Layout == nil {
		s.Layout = &LayoutTemplate{}
	}

	// Apply default styles if sheet doesn't have its own
	if s.Style == nil && defaults != nil {
		s.Style = &SheetStyleTemplate{
			HeaderStyle: defaults.HeaderStyle,
			DataStyle:   defaults.DataStyle,
		}
	} else if s.Style != nil && defaults != nil {
		// Merge with defaults (sheet style takes precedence)
		if s.Style.HeaderStyle == nil {
			s.Style.HeaderStyle = defaults.HeaderStyle
		}
		if s.Style.DataStyle == nil {
			s.Style.DataStyle = defaults.DataStyle
		}
	}

	// Set column headers to column names if not specified
	for i := range s.Columns {
		if s.Columns[i].Header == "" {
			s.Columns[i].Header = s.Columns[i].Name
		}
	}

	return nil
}

// ResolveVariables substitutes ${VAR_NAME} placeholders with actual values
func (t *ReportTemplate) ResolveVariables(runtimeVars map[string]interface{}) error {
	// Merge template variables with runtime variables (runtime takes precedence)
	mergedVars := make(map[string]string)
	for k, v := range t.Variables {
		mergedVars[k] = v
	}
	for k, v := range runtimeVars {
		mergedVars[k] = fmt.Sprintf("%v", v)
	}

	// Resolve in sheet queries
	for i := range t.Sheets {
		t.Sheets[i].Query = resolveString(t.Sheets[i].Query, mergedVars)
		t.Sheets[i].Name = resolveString(t.Sheets[i].Name, mergedVars)
	}

	return nil
}

// resolveString replaces ${VAR} placeholders in a string
func resolveString(s string, vars map[string]string) string {
	result := s
	for k, v := range vars {
		placeholder := "${" + k + "}"
		result = strings.ReplaceAll(result, placeholder, v)
	}
	return result
}

// LoadQueryFile loads SQL from an external file
func LoadQueryFile(basePath, queryFile string) (string, error) {
	// Construct full path relative to template location
	fullPath := queryFile
	if basePath != "" && !strings.HasPrefix(queryFile, "/") {
		fullPath = strings.TrimSuffix(basePath, "/") + "/" + queryFile
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("reading query file '%s': %w", fullPath, err)
	}

	return string(data), nil
}

// GetColumnByName finds a column template by database column name
func (s *SheetTemplate) GetColumnByName(name string) *ColumnTemplate {
	for i := range s.Columns {
		if s.Columns[i].Name == name {
			return &s.Columns[i]
		}
	}
	return nil
}

// ToProtectionRules converts ProtectionTemplate to existing ProtectionRule slice
func (p *ProtectionTemplate) ToProtectionRules() []ProtectionRule {
	if p == nil {
		return nil
	}

	var rules []ProtectionRule

	// Handle unlocked columns
	if len(p.UnlockedColumns) > 0 {
		rules = append(rules, Columns(p.UnlockedColumns...))
	}

	// Handle locked columns
	if len(p.LockedColumns) > 0 {
		rules = append(rules, LockColumns(p.LockedColumns...))
	}

	// Handle unlocked ranges
	if len(p.UnlockedRanges) > 0 {
		rules = append(rules, UnlockRange(p.UnlockedRanges...))
	}

	// Handle locked rows
	for _, rowSpec := range p.LockedRows {
		if rowSpec == "header" {
			rules = append(rules, LockRows(1))
		} else if strings.Contains(rowSpec, "-") {
			parts := strings.Split(rowSpec, "-")
			start, _ := strconv.Atoi(parts[0])
			end, _ := strconv.Atoi(parts[1])
			rules = append(rules, &lockRowsRule{rows: []RowRange{{Start: start, End: end}}})
		} else {
			row, err := strconv.Atoi(rowSpec)
			if err == nil {
				rules = append(rules, LockRows(row))
			}
		}
	}

	return rules
}

// ToSheetProtection converts ProtectionTemplate to SheetProtection
func (p *ProtectionTemplate) ToSheetProtection() *SheetProtection {
	if p == nil || !p.LockSheet {
		return nil
	}

	sp := NewSheetProtection()
	sp.Password = p.Password
	sp.ProtectSheet = p.LockSheet
	sp.AllowFilter = p.AllowFilter
	sp.AllowSort = p.AllowSort
	sp.AllowFormatCells = p.AllowFormatCells
	sp.AllowFormatColumns = p.AllowFormatColumns
	sp.AllowFormatRows = p.AllowFormatRows
	sp.AllowInsertRows = p.AllowInsertRows
	sp.AllowInsertColumns = p.AllowInsertColumns
	sp.AllowInsertHyperlinks = p.AllowInsertHyperlinks
	sp.AllowDeleteRows = p.AllowDeleteRows
	sp.AllowDeleteColumns = p.AllowDeleteColumns
	sp.AllowPivotTables = p.AllowPivotTables

	// Apply protection rules
	for _, rule := range p.ToProtectionRules() {
		rule.Apply(sp)
	}

	return sp
}
