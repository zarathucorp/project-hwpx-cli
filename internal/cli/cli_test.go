package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "..", "test", "fixtures", "minimal")
}

func fixtureArchive(t *testing.T) string {
	t.Helper()
	workDir := t.TempDir()
	archivePath := filepath.Join(workDir, "fixture.hwpx")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"pack", fixtureDir(t), "--output", archivePath}, stdout, stderr); err != nil {
		t.Fatalf("pack fixture: %v stderr=%s", err, stderr.String())
	}

	return archivePath
}

func TestTextJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"text", archivePath, "--format", "json"}, stdout, stderr)
	if err != nil {
		t.Fatalf("text json should succeed: %v stderr=%s", err, stderr.String())
	}

	var envelope struct {
		Command string `json:"command"`
		Success bool   `json:"success"`
		Data    struct {
			Text           string `json:"text"`
			LineCount      int    `json:"lineCount"`
			CharacterCount int    `json:"characterCount"`
		} `json:"data"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode envelope: %v", decodeErr)
	}

	if !envelope.Success || envelope.Command != "text" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if envelope.Data.Text != "Hello HWPX\nSecond paragraph" {
		t.Fatalf("unexpected text payload: %q", envelope.Data.Text)
	}
	if envelope.Data.LineCount != 2 {
		t.Fatalf("unexpected line count: %d", envelope.Data.LineCount)
	}
}

func TestValidateJSONFailure(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tempDir, "mimetype"), []byte("application/hwp+zip"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"validate", tempDir, "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("validate should fail for invalid directory")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
		Data struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode envelope: %v", decodeErr)
	}

	if envelope.Success {
		t.Fatal("validate failure should have success=false")
	}
	if envelope.Error.Code != "validation_failed" {
		t.Fatalf("unexpected error code: %s", envelope.Error.Code)
	}
	if envelope.Data.Report.Valid {
		t.Fatal("invalid report should remain invalid")
	}
}

func TestMutatingCommandFailsWhenDirectoryIsLocked(t *testing.T) {
	editableDir := filepath.Join(t.TempDir(), "editable")
	runCLI(t, "create", "--output", editableDir, "--format", "json")

	lockPath := filepath.Join(editableDir, documentLockFileName)
	if err := os.WriteFile(lockPath, lockMetadataForTest("append-text", os.Getpid()), 0o644); err != nil {
		t.Fatalf("write lock file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"append-text", editableDir, "--text", "잠금 테스트", "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("append-text should fail while lock is held")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode lock failure: %v", decodeErr)
	}
	if envelope.Success {
		t.Fatal("lock failure should have success=false")
	}
	if envelope.Error.Code != "resource_busy" {
		t.Fatalf("unexpected error code: %s", envelope.Error.Code)
	}
	if !strings.Contains(envelope.Error.Message, "locked") {
		t.Fatalf("expected lock message, got: %s", envelope.Error.Message)
	}
}

func TestMutatingCommandRecoversFromStaleLock(t *testing.T) {
	editableDir := filepath.Join(t.TempDir(), "editable")
	runCLI(t, "create", "--output", editableDir, "--format", "json")

	lockPath := filepath.Join(editableDir, documentLockFileName)
	if err := os.WriteFile(lockPath, lockMetadataForTest("append-text", 0), 0o644); err != nil {
		t.Fatalf("write stale lock file: %v", err)
	}

	appendStdout := runCLI(t, "append-text", editableDir, "--text", "stale lock recovery", "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(appendStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode append-text response: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("expected append-text success after stale lock cleanup: %s", appendStdout.String())
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale lock to be removed, stat err=%v", err)
	}
}

func TestSchemaCommandReturnsJSON(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"schema"}, stdout, stderr)
	if err != nil {
		t.Fatalf("schema should succeed: %v stderr=%s", err, stderr.String())
	}

	var document struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Environment []any  `json:"environment"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &document); decodeErr != nil {
		t.Fatalf("decode schema: %v", decodeErr)
	}

	if document.Name != "hwpxctl" {
		t.Fatalf("unexpected schema name: %s", document.Name)
	}
	if document.Version == "" || len(document.Environment) == 0 {
		t.Fatalf("schema should include version and environment: %+v", document)
	}
}

func TestSubcommandHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"inspect", "--help"}, stdout, stderr)
	if err != nil {
		t.Fatalf("inspect help should succeed: %v stderr=%s", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "Inspect HWPX metadata") {
		t.Fatalf("unexpected help output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--format") {
		t.Fatalf("help should include inherited format flag: %s", stdout.String())
	}
}

func TestUnknownCommandJSONFailure(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"missing-command", "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("unknown command should fail")
	}

	var envelope struct {
		Command string `json:"command"`
		Success bool   `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode envelope: %v", decodeErr)
	}

	if envelope.Command != "missing-command" {
		t.Fatalf("unexpected command: %s", envelope.Command)
	}
	if envelope.Success {
		t.Fatal("unknown command should have success=false")
	}
	if envelope.Error.Code != "unknown_command" {
		t.Fatalf("unexpected error code: %s", envelope.Error.Code)
	}
}

func copyFirstTableParagraphToSection(t *testing.T, editableDir string, sectionIndex int) {
	t.Helper()

	sourceDoc := etree.NewDocument()
	if err := sourceDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read source section xml: %v", err)
	}
	targetDoc := etree.NewDocument()
	targetPath := filepath.Join(editableDir, "Contents", "section"+strconv.Itoa(sectionIndex)+".xml")
	if err := targetDoc.ReadFromFile(targetPath); err != nil {
		t.Fatalf("read target section xml: %v", err)
	}

	sourceRoot := sourceDoc.Root()
	if sourceRoot == nil {
		t.Fatal("expected source section root")
	}
	var tableParagraph *etree.Element
	for _, paragraph := range sourceRoot.FindElements("./hp:p") {
		if paragraph.FindElement(".//hp:tbl") != nil {
			tableParagraph = paragraph
			break
		}
	}
	if tableParagraph == nil {
		t.Fatal("expected table paragraph in source section")
	}
	targetRoot := targetDoc.Root()
	if targetRoot == nil {
		t.Fatal("expected target section root")
	}
	targetRoot.AddChild(tableParagraph.Copy())
	if err := targetDoc.WriteToFile(targetPath); err != nil {
		t.Fatalf("write target section xml: %v", err)
	}
}

func copyFirstEditableParagraphToSection(t *testing.T, editableDir string, sectionIndex int) {
	t.Helper()

	sourceDoc := etree.NewDocument()
	if err := sourceDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read source section xml: %v", err)
	}
	targetDoc := etree.NewDocument()
	targetPath := filepath.Join(editableDir, "Contents", "section"+strconv.Itoa(sectionIndex)+".xml")
	if err := targetDoc.ReadFromFile(targetPath); err != nil {
		t.Fatalf("read target section xml: %v", err)
	}

	sourceRoot := sourceDoc.Root()
	targetRoot := targetDoc.Root()
	if sourceRoot == nil || targetRoot == nil {
		t.Fatal("expected section roots")
	}

	var editableParagraph *etree.Element
	for _, paragraph := range sourceRoot.FindElements("./hp:p") {
		if paragraph.FindElement(".//hp:secPr") != nil {
			continue
		}
		editableParagraph = paragraph
		break
	}
	if editableParagraph == nil {
		t.Fatal("expected editable paragraph in source section")
	}

	targetRoot.AddChild(editableParagraph.Copy())
	if err := targetDoc.WriteToFile(targetPath); err != nil {
		t.Fatalf("write target section xml: %v", err)
	}
}

func paragraphPlainTextForTest(paragraph *etree.Element) string {
	if paragraph == nil {
		return ""
	}

	var builder strings.Builder
	for _, textElement := range paragraph.FindElements(".//hp:t") {
		builder.WriteString(textElement.Text())
	}
	return builder.String()
}

func firstLineSegAttrsForTest(element *etree.Element) map[string]string {
	if element == nil {
		return nil
	}
	var lineSeg *etree.Element
	var walk func(*etree.Element)
	walk = func(current *etree.Element) {
		if current == nil || lineSeg != nil {
			return
		}
		if strings.HasSuffix(current.Tag, "lineseg") {
			lineSeg = current
			return
		}
		for _, child := range current.ChildElements() {
			walk(child)
		}
	}
	walk(element)
	if lineSeg == nil {
		return nil
	}
	attrs := make(map[string]string, len(lineSeg.Attr))
	for _, attr := range lineSeg.Attr {
		attrs[attr.Key] = attr.Value
	}
	return attrs
}

func ensureLineSegAttrsForTest(paragraph *etree.Element, attrs map[string]string) {
	if paragraph == nil {
		return
	}
	if existing := paragraph.FindElement("./hp:linesegarray"); existing != nil {
		paragraph.RemoveChild(existing)
	}
	lineSegArray := paragraph.CreateElement("hp:linesegarray")
	lineSeg := lineSegArray.CreateElement("hp:lineseg")
	for key, value := range attrs {
		lineSeg.CreateAttr(key, value)
	}
}

func appendSectionParagraphForTest(t *testing.T, sectionPath, text string) {
	t.Helper()

	doc := etree.NewDocument()
	if err := doc.ReadFromFile(sectionPath); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	root := doc.Root()
	if root == nil {
		t.Fatal("expected section root")
	}

	var templateParagraph *etree.Element
	for _, paragraph := range root.FindElements("./hp:p") {
		if paragraph.FindElement(".//hp:secPr") != nil {
			continue
		}
		templateParagraph = paragraph
		break
	}
	if templateParagraph == nil {
		t.Fatal("expected editable paragraph template")
	}

	paragraph := templateParagraph.Copy()
	for _, child := range append([]*etree.Element{}, paragraph.ChildElements()...) {
		paragraph.RemoveChild(child)
	}

	run := paragraph.CreateElement("hp:run")
	run.CreateAttr("charPrIDRef", "0")
	run.CreateElement("hp:t").SetText(text)

	root.AddChild(paragraph)
	if err := doc.WriteToFile(sectionPath); err != nil {
		t.Fatalf("write section xml: %v", err)
	}
}

func TestUnpackJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr)
	if err != nil {
		t.Fatalf("unpack json should succeed: %v stderr=%s", err, stderr.String())
	}

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			OutputPath string `json:"outputPath"`
			Report     struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode unpack envelope: %v", decodeErr)
	}

	if !envelope.Success || !envelope.Data.Report.Valid {
		t.Fatalf("unexpected unpack envelope: %+v", envelope)
	}
	if envelope.Data.OutputPath == "" {
		t.Fatal("json output should include outputPath")
	}
}

func TestAnalyzeTemplateJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for analyze-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "{{PROJECT_TITLE}}")
	appendSectionParagraphForTest(t, sectionPath, "※ 작성요령: 이 항목을 작성하세요")
	appendSectionParagraphForTest(t, sectionPath, "사업비 총괄표")
	runCLI(t, "add-table", outputDir, "--cells", "과제명,프로젝트X;주관기관,예시 기관", "--format", "json")

	analyzeStdout := runCLI(t, "analyze-template", outputDir, "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Analysis struct {
				SectionCount     int `json:"sectionCount"`
				TableCount       int `json:"tableCount"`
				ParagraphCount   int `json:"paragraphCount"`
				PlaceholderCount int `json:"placeholderCount"`
				GuideCount       int `json:"guideCount"`
				Sections         []struct {
					SectionIndex int `json:"sectionIndex"`
				} `json:"sections"`
				Tables []struct {
					LabelText string `json:"labelText"`
					Cells     []struct {
						Row  int    `json:"row"`
						Col  int    `json:"col"`
						Text string `json:"text"`
					} `json:"cells"`
				} `json:"tables"`
				Paragraphs []struct {
					Text         string `json:"text"`
					StyleSummary string `json:"styleSummary"`
				} `json:"paragraphs"`
				Placeholders []struct {
					Text   string `json:"text"`
					Reason string `json:"reason"`
				} `json:"placeholders"`
				Guides []struct {
					Text   string `json:"text"`
					Reason string `json:"reason"`
				} `json:"guides"`
			} `json:"analysis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(analyzeStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode analyze-template response: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("unexpected analyze-template response: %s", analyzeStdout.String())
	}
	if envelope.Data.Analysis.SectionCount != 1 {
		t.Fatalf("unexpected section count: %+v", envelope.Data.Analysis)
	}
	if envelope.Data.Analysis.ParagraphCount == 0 {
		t.Fatalf("expected paragraph discovery results: %+v", envelope.Data.Analysis)
	}
	if envelope.Data.Analysis.PlaceholderCount == 0 || envelope.Data.Analysis.GuideCount == 0 {
		t.Fatalf("expected placeholder and guide candidates: %+v", envelope.Data.Analysis)
	}
	if envelope.Data.Analysis.TableCount == 0 || len(envelope.Data.Analysis.Tables) == 0 || len(envelope.Data.Analysis.Tables[0].Cells) == 0 {
		t.Fatalf("expected detailed table discovery results: %+v", envelope.Data.Analysis.Tables)
	}
	if len(envelope.Data.Analysis.Sections) != 1 || envelope.Data.Analysis.Sections[0].SectionIndex != 0 {
		t.Fatalf("unexpected sections: %+v", envelope.Data.Analysis.Sections)
	}
	if envelope.Data.Analysis.Placeholders[0].Reason == "" || !strings.Contains(envelope.Data.Analysis.Placeholders[0].Text, "PROJECT_TITLE") {
		t.Fatalf("unexpected placeholder candidate: %+v", envelope.Data.Analysis.Placeholders)
	}
	if envelope.Data.Analysis.Guides[0].Reason == "" || !strings.Contains(envelope.Data.Analysis.Guides[0].Text, "작성요령") {
		t.Fatalf("unexpected guide candidate: %+v", envelope.Data.Analysis.Guides)
	}
	if envelope.Data.Analysis.Tables[0].LabelText == "" || !strings.Contains(envelope.Data.Analysis.Tables[0].LabelText, "사업비 총괄표") {
		t.Fatalf("expected inferred table label: %+v", envelope.Data.Analysis.Tables)
	}
	if envelope.Data.Analysis.Tables[0].Cells[0].Text == "" {
		t.Fatalf("expected cell text preview: %+v", envelope.Data.Analysis.Tables[0].Cells)
	}
	if len(envelope.Data.Analysis.Paragraphs) == 0 || envelope.Data.Analysis.Paragraphs[0].Text == "" {
		t.Fatalf("expected paragraph details: %+v", envelope.Data.Analysis.Paragraphs)
	}
}

func TestFindTargetsJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for find-targets: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "{{PROJECT_TITLE}}")
	appendSectionParagraphForTest(t, sectionPath, "사업비 총괄표")
	runCLI(t, "add-table", outputDir, "--cells", "과제명,프로젝트X;주관기관,예시 기관", "--format", "json")

	findStdout := runCLI(t, "find-targets", outputDir, "--anchor", "주관기관", "--table-label", "사업비 총괄표", "--placeholder", "PROJECT_TITLE", "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Kind      string `json:"kind"`
				QueryType string `json:"queryType"`
				LabelText string `json:"labelText"`
				Text      string `json:"text"`
				Reason    string `json:"reason"`
				Cell      *struct {
					Row int `json:"row"`
					Col int `json:"col"`
				} `json:"cell"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(findStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode find-targets response: %v", err)
	}
	if !envelope.Success || envelope.Data.Count < 3 {
		t.Fatalf("unexpected find-targets response: %s", findStdout.String())
	}

	var foundCell bool
	var foundTable bool
	var foundPlaceholder bool
	for _, match := range envelope.Data.Matches {
		if match.Kind == "cell" && match.QueryType == "anchor" && match.Cell != nil && strings.Contains(match.Text, "주관기관") {
			foundCell = true
		}
		if match.Kind == "table" && match.QueryType == "table-label" && strings.Contains(match.LabelText, "사업비 총괄표") {
			foundTable = true
		}
		if match.Kind == "placeholder" && match.QueryType == "placeholder" && strings.Contains(match.Text, "PROJECT_TITLE") && match.Reason != "" {
			foundPlaceholder = true
		}
	}
	if !foundCell || !foundTable || !foundPlaceholder {
		t.Fatalf("expected anchor, table-label, and placeholder matches: %+v", envelope.Data.Matches)
	}
}

func TestRemoveGuidesDryRunJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for remove-guides: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "작성요령: 이 항목을 채우세요")
	appendSectionParagraphForTest(t, sectionPath, "※ 참고: 안내문입니다")

	beforeBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section before remove-guides: %v", err)
	}

	removeStdout := runCLI(t, "remove-guides", outputDir, "--dry-run", "true", "--reason", "guide-text", "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			DryRun           bool  `json:"dryRun"`
			Count            int   `json:"count"`
			AffectedSections []int `json:"affectedSections"`
			GuideCandidates  []struct {
				SectionIndex int    `json:"sectionIndex"`
				Reason       string `json:"reason"`
				Text         string `json:"text"`
			} `json:"guideCandidates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(removeStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode remove-guides response: %v", err)
	}
	if !envelope.Success || !envelope.Data.DryRun {
		t.Fatalf("unexpected remove-guides response: %s", removeStdout.String())
	}
	if envelope.Data.Count != 1 || len(envelope.Data.GuideCandidates) != 1 {
		t.Fatalf("expected one filtered guide candidate: %+v", envelope.Data)
	}
	if envelope.Data.GuideCandidates[0].Reason != "guide-text" || !strings.Contains(envelope.Data.GuideCandidates[0].Text, "작성요령") {
		t.Fatalf("unexpected guide candidate: %+v", envelope.Data.GuideCandidates)
	}
	if len(envelope.Data.AffectedSections) != 1 || envelope.Data.AffectedSections[0] != 0 {
		t.Fatalf("unexpected affected sections: %+v", envelope.Data.AffectedSections)
	}

	afterBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after remove-guides: %v", err)
	}
	if string(beforeBytes) != string(afterBytes) {
		t.Fatal("remove-guides dry-run should not mutate section xml")
	}
}

func TestRemoveGuidesApplyJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for remove-guides apply: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "※ 작성요령: 이 항목을 채우세요")
	appendSectionParagraphForTest(t, sectionPath, "정상 본문 문단")

	removeStdout := runCLI(t, "remove-guides", outputDir, "--dry-run", "false", "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			DryRun          bool `json:"dryRun"`
			Applied         bool `json:"applied"`
			Count           int  `json:"count"`
			GuideCandidates []struct {
				Text   string `json:"text"`
				Reason string `json:"reason"`
			} `json:"guideCandidates"`
			Report *struct {
				Valid      bool `json:"valid"`
				RenderSafe bool `json:"renderSafe"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(removeStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode remove-guides apply response: %v", err)
	}
	if !envelope.Success || envelope.Data.DryRun || !envelope.Data.Applied {
		t.Fatalf("unexpected remove-guides apply response: %s", removeStdout.String())
	}
	if envelope.Data.Count == 0 || envelope.Data.Report == nil || !envelope.Data.Report.Valid {
		t.Fatalf("expected applied guide removal report: %+v", envelope.Data)
	}

	sectionBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after remove-guides apply: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Contains(sectionText, "작성요령") {
		t.Fatalf("expected guide text to be cleared from section xml, got: %q", sectionText)
	}
	if !strings.Contains(sectionText, "정상 본문 문단") {
		t.Fatalf("expected non-guide text to remain in section xml, got: %q", sectionText)
	}

	analyzeStdout := runCLI(t, "analyze-template", outputDir, "--format", "json")
	var analyzeEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Analysis struct {
				GuideCount int `json:"guideCount"`
			} `json:"analysis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(analyzeStdout.Bytes(), &analyzeEnvelope); err != nil {
		t.Fatalf("decode analyze-template after remove-guides apply: %v", err)
	}
	if !analyzeEnvelope.Success || analyzeEnvelope.Data.Analysis.GuideCount != 0 {
		t.Fatalf("expected guide candidates to be removed after apply: %s", analyzeStdout.String())
	}
}

func TestFillTemplateJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for fill-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "{{PROJECT_TITLE}}")
	appendSectionParagraphForTest(t, sectionPath, "요약")
	appendSectionParagraphForTest(t, sectionPath, "기존 요약")
	appendSectionParagraphForTest(t, sectionPath, "세부 내용")
	appendSectionParagraphForTest(t, sectionPath, "사업비 총괄표")
	runCLI(t, "add-table", outputDir, "--cells", "과제명,기존 과제명;주관기관,기존 기관", "--format", "json")
	appendSectionParagraphForTest(t, sectionPath, "담당자 표")
	runCLI(t, "add-table", outputDir, "--cells", "담당자;기존 담당자", "--format", "json")
	appendSectionParagraphForTest(t, sectionPath, "참여기관 표")
	runCLI(t, "add-table", outputDir, "--cells", "참여기관;기존 참여기관 1;기존 참여기관 2;기존 참여기관 3", "--format", "json")
	appendSectionParagraphForTest(t, sectionPath, "대체 참여기관 표")
	runCLI(t, "add-table", outputDir, "--cells", "참여기관,건드리면 안 됨", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "mapping.json")
	mapping := `{
  "replacements": [
    {"placeholder": "{{PROJECT_TITLE}}", "value": "프로젝트 X"},
    {"nearText": "요약", "value": "새 요약 본문", "mode": "paragraph-next"},
    {"nearText": "세부 내용", "value": "세부 내용 최종본", "mode": "paragraph-replace"},
    {"anchor": "과제명", "value": "프로젝트 X"},
    {"anchor": "주관기관", "value": "예시 기관"},
    {"anchor": "담당자", "tableLabel": "담당자 표", "value": "홍길동", "mode": "table-down"},
    {"anchor": "참여기관", "tableLabel": "참여기관 표", "values": ["기관1", "기관2", "기관3"], "mode": "table-down-repeat"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			DryRun    bool `json:"dryRun"`
			Applied   bool `json:"applied"`
			Count     int  `json:"count"`
			MissCount int  `json:"missCount"`
			Changes   []struct {
				Kind       string `json:"kind"`
				Mode       string `json:"mode"`
				TableLabel string `json:"tableLabel"`
				Selector   string `json:"selector"`
				Text       string `json:"text"`
			} `json:"changes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode fill-template dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || !dryRunEnvelope.Data.DryRun || dryRunEnvelope.Data.Applied {
		t.Fatalf("unexpected fill-template dry-run response: %s", dryRunStdout.String())
	}
	if dryRunEnvelope.Data.MissCount != 0 {
		t.Fatalf("expected no misses for fill-template dry-run: %+v", dryRunEnvelope.Data)
	}
	if dryRunEnvelope.Data.Count < 9 {
		t.Fatalf("expected at least nine planned changes: %+v", dryRunEnvelope.Data)
	}
	var scopedRepeatCount int
	for _, change := range dryRunEnvelope.Data.Changes {
		if change.Mode == "table-down-repeat" && change.TableLabel == "참여기관 표" {
			scopedRepeatCount++
		}
	}
	if scopedRepeatCount != 3 {
		t.Fatalf("expected three scoped repeat changes for 참여기관 표: %+v", dryRunEnvelope.Data.Changes)
	}

	applyStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")
	var applyEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			DryRun    bool `json:"dryRun"`
			Applied   bool `json:"applied"`
			Count     int  `json:"count"`
			MissCount int  `json:"missCount"`
			Report    *struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(applyStdout.Bytes(), &applyEnvelope); err != nil {
		t.Fatalf("decode fill-template apply response: %v", err)
	}
	if !applyEnvelope.Success || applyEnvelope.Data.DryRun || !applyEnvelope.Data.Applied {
		t.Fatalf("unexpected fill-template apply response: %s", applyStdout.String())
	}
	if applyEnvelope.Data.MissCount != 0 {
		t.Fatalf("expected no misses for fill-template apply: %+v", applyEnvelope.Data)
	}
	if applyEnvelope.Data.Count < 9 || applyEnvelope.Data.Report == nil || !applyEnvelope.Data.Report.Valid {
		t.Fatalf("expected applied fill-template report: %+v", applyEnvelope.Data)
	}

	sectionBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"프로젝트 X", "예시 기관", "새 요약 본문", "세부 내용 최종본", "홍길동", "기관1", "기관2", "기관3"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after fill-template apply: %q", needle, sectionText)
		}
	}
	if !strings.Contains(sectionText, "건드리면 안 됨") {
		t.Fatalf("expected non-target table to remain unchanged: %q", sectionText)
	}
	if strings.Contains(sectionText, "{{PROJECT_TITLE}}") {
		t.Fatalf("expected placeholder to be replaced: %q", sectionText)
	}
	if strings.Contains(sectionText, "기존 요약") {
		t.Fatalf("expected near-text paragraph target to be replaced: %q", sectionText)
	}
}

func TestRoundtripCheckJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)

	roundtripStdout := runCLI(t, "roundtrip-check", archivePath, "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Check struct {
				Passed bool `json:"passed"`
				Before struct {
					TextDigest         string   `json:"textDigest"`
					ParagraphDigest    string   `json:"paragraphDigest"`
					TableDigest        string   `json:"tableDigest"`
					ObjectDigest       string   `json:"objectDigest"`
					HeaderFooterDigest string   `json:"headerFooterDigest"`
					ControlDigest      string   `json:"controlDigest"`
					SectionPaths       []string `json:"sectionPaths"`
				} `json:"before"`
				After struct {
					TextDigest         string   `json:"textDigest"`
					ParagraphDigest    string   `json:"paragraphDigest"`
					TableDigest        string   `json:"tableDigest"`
					ObjectDigest       string   `json:"objectDigest"`
					HeaderFooterDigest string   `json:"headerFooterDigest"`
					ControlDigest      string   `json:"controlDigest"`
					SectionPaths       []string `json:"sectionPaths"`
				} `json:"after"`
				Issues []struct {
					Code           string `json:"code"`
					Severity       string `json:"severity"`
					SectionPath    string `json:"sectionPath"`
					ParagraphIndex *int   `json:"paragraphIndex"`
					TableIndex     *int   `json:"tableIndex"`
					Before         string `json:"before"`
					After          string `json:"after"`
				} `json:"issues"`
			} `json:"check"`
		} `json:"data"`
	}
	if err := json.Unmarshal(roundtripStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode roundtrip-check response: %v", err)
	}
	if !envelope.Success || !envelope.Data.Check.Passed {
		t.Fatalf("unexpected roundtrip-check response: %s", roundtripStdout.String())
	}
	if envelope.Data.Check.Before.TextDigest == "" || envelope.Data.Check.Before.ParagraphDigest == "" || envelope.Data.Check.Before.TableDigest == "" {
		t.Fatalf("expected strict digests in before snapshot: %+v", envelope.Data.Check.Before)
	}
	if envelope.Data.Check.Before.ObjectDigest == "" || envelope.Data.Check.Before.HeaderFooterDigest == "" || envelope.Data.Check.Before.ControlDigest == "" {
		t.Fatalf("expected structure digests in before snapshot: %+v", envelope.Data.Check.Before)
	}
	if envelope.Data.Check.After.TextDigest == "" || envelope.Data.Check.After.ParagraphDigest == "" || envelope.Data.Check.After.TableDigest == "" {
		t.Fatalf("expected strict digests in after snapshot: %+v", envelope.Data.Check.After)
	}
	if envelope.Data.Check.After.ObjectDigest == "" || envelope.Data.Check.After.HeaderFooterDigest == "" || envelope.Data.Check.After.ControlDigest == "" {
		t.Fatalf("expected structure digests in after snapshot: %+v", envelope.Data.Check.After)
	}
	if len(envelope.Data.Check.Before.SectionPaths) == 0 || len(envelope.Data.Check.After.SectionPaths) == 0 {
		t.Fatalf("expected section paths in roundtrip snapshots: %+v", envelope.Data.Check)
	}
	if len(envelope.Data.Check.Issues) != 0 {
		t.Fatalf("expected no issues for stable fixture: %+v", envelope.Data.Check.Issues)
	}
}

func TestFillTemplateRepeatSkipsMergedTargets(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for merged repeat fill-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "병합 참여기관 표")
	runCLI(t, "add-table", outputDir, "--cells", "병합 참여기관;기존 병합 1;기존 병합 2;기존 병합 3;기존 병합 4", "--format", "json")
	runCLI(t, "merge-table-cells", outputDir, "--table", "0", "--start-row", "1", "--start-col", "0", "--end-row", "2", "--end-col", "0", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "merged-repeat-mapping.json")
	mapping := `{
  "replacements": [
    {"anchor": "병합 참여기관", "tableLabel": "병합 참여기관 표", "values": ["병합 기관1", "병합 기관2"], "mode": "table-down-repeat"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write merged repeat mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Changes []struct {
				Mode string `json:"mode"`
				Text string `json:"text"`
				Cell struct {
					Row int `json:"row"`
					Col int `json:"col"`
				} `json:"cell"`
			} `json:"changes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode merged repeat dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 2 {
		t.Fatalf("expected exactly two merged repeat changes: %s", dryRunStdout.String())
	}
	if dryRunEnvelope.Data.Changes[0].Cell.Row != 1 || dryRunEnvelope.Data.Changes[1].Cell.Row != 3 {
		t.Fatalf("expected merged repeat to skip duplicate anchor rows: %+v", dryRunEnvelope.Data.Changes)
	}
	if dryRunEnvelope.Data.Changes[0].Text != "병합 기관1" || dryRunEnvelope.Data.Changes[1].Text != "병합 기관2" {
		t.Fatalf("expected ordered merged repeat values: %+v", dryRunEnvelope.Data.Changes)
	}

	runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after merged repeat fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"병합 기관1", "병합 기관2"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after merged repeat apply: %q", needle, sectionText)
		}
	}
}

func TestSafePackJSONOutput(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for safe-pack: %v stderr=%s", err, stderr.String())
	}

	packedPath := filepath.Join(t.TempDir(), "safe-packed.hwpx")
	safePackStdout := runCLI(t, "safe-pack", outputDir, "--output", packedPath, "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Packed    bool     `json:"packed"`
			BlockedBy []string `json:"blockedBy"`
			Report    *struct {
				Valid bool `json:"valid"`
			} `json:"report"`
			Check *struct {
				Passed bool `json:"passed"`
			} `json:"check"`
		} `json:"data"`
	}
	if err := json.Unmarshal(safePackStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode safe-pack response: %v", err)
	}
	if !envelope.Success || !envelope.Data.Packed || envelope.Data.Report == nil || !envelope.Data.Report.Valid {
		t.Fatalf("unexpected safe-pack response: %s", safePackStdout.String())
	}
	if envelope.Data.Check == nil || !envelope.Data.Check.Passed {
		t.Fatalf("expected roundtrip-check to pass during safe-pack: %+v", envelope.Data)
	}
	if len(envelope.Data.BlockedBy) != 0 {
		t.Fatalf("expected no blocking reasons for stable fixture: %+v", envelope.Data.BlockedBy)
	}
	if _, err := os.Stat(packedPath); err != nil {
		t.Fatalf("expected safe-packed archive: %v", err)
	}
}

func TestSafePackBlocksUnsafeDirectory(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	unsafeOutput := filepath.Join(workDir, "unsafe.hwpx")
	runCLI(t, "create", "--output", editableDir, "--format", "json")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"safe-pack", editableDir, "--output", unsafeOutput, "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("safe-pack should block unsafe directory without --force")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
		Data struct {
			Packed    bool     `json:"packed"`
			BlockedBy []string `json:"blockedBy"`
			Report    *struct {
				RenderSafe bool `json:"renderSafe"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode safe-pack blocked response: %v", err)
	}
	if envelope.Success || envelope.Error.Code != "safety_blocked" {
		t.Fatalf("unexpected safe-pack blocked envelope: %s", stdout.String())
	}
	if envelope.Data.Packed {
		t.Fatalf("blocked safe-pack should not pack: %+v", envelope.Data)
	}
	if len(envelope.Data.BlockedBy) == 0 || envelope.Data.Report == nil || envelope.Data.Report.RenderSafe {
		t.Fatalf("expected blocking reasons and render-safe failure: %+v", envelope.Data)
	}
	if _, err := os.Stat(unsafeOutput); !os.IsNotExist(err) {
		t.Fatalf("blocked safe-pack should not create archive, stat err=%v", err)
	}

	forcedStdout := runCLI(t, "safe-pack", editableDir, "--output", unsafeOutput, "--force", "true", "--format", "json")
	var forcedEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Forced bool `json:"forced"`
			Packed bool `json:"packed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(forcedStdout.Bytes(), &forcedEnvelope); err != nil {
		t.Fatalf("decode forced safe-pack response: %v", err)
	}
	if !forcedEnvelope.Success || !forcedEnvelope.Data.Forced || !forcedEnvelope.Data.Packed {
		t.Fatalf("unexpected forced safe-pack response: %s", forcedStdout.String())
	}
	if _, err := os.Stat(unsafeOutput); err != nil {
		t.Fatalf("expected forced safe-pack archive: %v", err)
	}
}

func TestExportWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")
	markdownPath := filepath.Join(workDir, "result.md")
	htmlPath := filepath.Join(workDir, "result.html")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "제목\n본문 문단", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,값;이름,홍길동", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	markdownStdout := runCLI(t, "export-markdown", archivePath, "--output", markdownPath, "--format", "json")
	htmlStdout := runCLI(t, "export-html", archivePath, "--output", htmlPath, "--format", "json")

	var markdownEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			OutputPath string `json:"outputPath"`
			BlockCount int    `json:"blockCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(markdownStdout.Bytes(), &markdownEnvelope); err != nil {
		t.Fatalf("decode markdown response: %v", err)
	}
	if !markdownEnvelope.Success || markdownEnvelope.Data.OutputPath == "" || markdownEnvelope.Data.BlockCount < 3 {
		t.Fatalf("unexpected markdown response: %s", markdownStdout.String())
	}

	var htmlEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			OutputPath string `json:"outputPath"`
			BlockCount int    `json:"blockCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(htmlStdout.Bytes(), &htmlEnvelope); err != nil {
		t.Fatalf("decode html response: %v", err)
	}
	if !htmlEnvelope.Success || htmlEnvelope.Data.OutputPath == "" || htmlEnvelope.Data.BlockCount < 3 {
		t.Fatalf("unexpected html response: %s", htmlStdout.String())
	}

	markdownBytes, err := os.ReadFile(markdownPath)
	if err != nil {
		t.Fatalf("read markdown export: %v", err)
	}
	markdownText := string(markdownBytes)
	for _, needle := range []string{"제목", "본문 문단", "| 항목 | 값 |"} {
		if !strings.Contains(markdownText, needle) {
			t.Fatalf("expected %q in markdown export: %s", needle, markdownText)
		}
	}

	htmlBytes, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("read html export: %v", err)
	}
	htmlText := string(htmlBytes)
	for _, needle := range []string{"<table>", "<p>제목</p>", "<td>홍길동</td>"} {
		if !strings.Contains(htmlText, needle) {
			t.Fatalf("expected %q in html export: %s", needle, htmlText)
		}
	}
}

func TestCreateEditPackWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	imagePath := filepath.Join(workDir, "pixel.png")
	archivePath := filepath.Join(workDir, "result.hwpx")

	if err := os.WriteFile(imagePath, mustTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,내용;이름,홍길동", "--format", "json")
	runCLI(t, "set-table-cell", editableDir, "--table", "0", "--row", "1", "--col", "1", "--text", "김영희", "--format", "json")

	embedStdout := runCLI(t, "embed-image", editableDir, "--image", imagePath, "--format", "json")
	var embedEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			ItemID     string `json:"itemId"`
			BinaryPath string `json:"binaryPath"`
		} `json:"data"`
	}
	if err := json.Unmarshal(embedStdout.Bytes(), &embedEnvelope); err != nil {
		t.Fatalf("decode embed-image response: %v", err)
	}
	if !embedEnvelope.Success || embedEnvelope.Data.ItemID == "" || embedEnvelope.Data.BinaryPath == "" {
		t.Fatalf("unexpected embed response: %s", embedStdout.String())
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if want := "첫 문단\n둘째 문단\n항목\n내용\n이름\n김영희"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid   bool `json:"valid"`
				Summary struct {
					BinaryPath []string `json:"binaryPaths"`
				} `json:"summary"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
	if len(inspectEnvelope.Data.Report.Summary.BinaryPath) != 1 {
		t.Fatalf("expected one embedded binary path: %v", inspectEnvelope.Data.Report.Summary.BinaryPath)
	}
}

func TestParagraphEditWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단\n셋째 문단", "--format", "json")
	runCLI(t, "set-paragraph-text", editableDir, "--paragraph", "1", "--text", "수정된 둘째 문단", "--format", "json")
	runCLI(t, "delete-paragraph", editableDir, "--paragraph", "0", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Contains(sectionText, "첫 문단") {
		t.Fatalf("expected deleted paragraph to be removed: %s", sectionText)
	}
	if !strings.Contains(sectionText, "수정된 둘째 문단") {
		t.Fatalf("expected updated paragraph text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "수정된 둘째 문단\n셋째 문단"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestDeleteParagraphAfterTrackedStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "Alpha\nBeta", "--track-changes", "true", "--change-author", "tester", "--change-summary", "seed paragraphs", "--format", "json")
	runCLI(t, "set-paragraph-text", editableDir, "--paragraph", "0", "--text", "Alpha updated", "--track-changes", "true", "--change-author", "tester", "--change-summary", "rewrite alpha", "--format", "json")
	runCLI(t, "add-run-text", editableDir, "--paragraph", "0", "--text", " / extra", "--format", "json")
	runCLI(t, "set-run-text", editableDir, "--paragraph", "0", "--run", "1", "--text", " / final", "--format", "json")
	runCLI(t, "set-text-style", editableDir, "--paragraph", "0", "--run", "0", "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--track-changes", "true", "--change-author", "tester", "--change-summary", "emphasis alpha", "--format", "json")
	runCLI(t, "set-paragraph-layout", editableDir, "--paragraph", "0", "--align", "CENTER", "--space-after-mm", "4", "--line-spacing-percent", "160", "--track-changes", "true", "--change-author", "tester", "--change-summary", "center alpha", "--format", "json")
	runCLI(t, "set-paragraph-list", editableDir, "--paragraph", "1", "--kind", "bullet", "--level", "0", "--track-changes", "true", "--change-author", "tester", "--change-summary", "bullet beta", "--format", "json")
	runCLI(t, "replace-runs-by-style", editableDir, "--bold", "true", "--text", "[강조]", "--track-changes", "true", "--change-author", "tester", "--change-summary", "replace bold run", "--format", "json")
	runCLI(t, "delete-paragraph", editableDir, "--paragraph", "1", "--track-changes", "true", "--change-author", "tester", "--change-summary", "drop beta", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Contains(sectionText, "Beta") {
		t.Fatalf("expected deleted paragraph to be removed after tracked edits: %s", sectionText)
	}
	if !strings.Contains(sectionText, "[강조]") {
		t.Fatalf("expected replaced text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "[강조] / final"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestAddRunTextWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "add-run-text", editableDir, "--paragraph", "1", "--text", " (검토본)", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"둘째 문단", "(검토본)"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}
	if strings.Count(sectionText, "<hp:run") < 4 {
		t.Fatalf("expected inserted run in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "첫 문단\n둘째 문단 (검토본)"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestAppendTextTrackChangesWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "추적 문단", "--track-changes", "true", "--change-author", "tester", "--change-summary", "Added tracked paragraph", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	contentBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "content.hpf"))
	if err != nil {
		t.Fatalf("read content.hpf: %v", err)
	}
	contentText := string(contentBytes)
	if !strings.Contains(contentText, "Contents/history.xml") {
		t.Fatalf("expected history manifest item: %s", contentText)
	}

	historyBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "history.xml"))
	if err != nil {
		t.Fatalf("read history.xml: %v", err)
	}
	historyText := string(historyBytes)
	for _, needle := range []string{
		"command=\"append-text\"",
		"author=\"tester\"",
		"Added tracked paragraph",
	} {
		if !strings.Contains(historyText, needle) {
			t.Fatalf("expected %q in history xml: %s", needle, historyText)
		}
	}

	validateStdout := runCLI(t, "validate", archivePath, "--format", "json")
	var validateEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(validateStdout.Bytes(), &validateEnvelope); err != nil {
		t.Fatalf("decode validate response: %v", err)
	}
	if !validateEnvelope.Success || !validateEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected validate response: %s", validateStdout.String())
	}
}

func TestSetRunTextWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "add-run-text", editableDir, "--paragraph", "1", "--text", " (검토본)", "--format", "json")
	runCLI(t, "set-run-text", editableDir, "--paragraph", "1", "--run", "1", "--text", " (최종본)", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Contains(sectionText, "(검토본)") {
		t.Fatalf("expected previous run text to be replaced: %s", sectionText)
	}
	if !strings.Contains(sectionText, "(최종본)") {
		t.Fatalf("expected updated run text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "첫 문단\n둘째 문단 (최종본)"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestSetTextStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "set-text-style", editableDir, "--paragraph", "1", "--bold", "true", "--italic", "true", "--underline", "true", "--text-color", "#C00000", "--font-name", "맑은 고딕", "--font-size-pt", "16", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	headerBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "header.xml"))
	if err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	headerText := string(headerBytes)
	for _, needle := range []string{
		"textColor=\"#C00000\"",
		"height=\"1600\"",
		"face=\"맑은 고딕\"",
		"<hh:bold",
		"<hh:italic",
		"<hh:underline type=\"BOTTOM\"",
	} {
		if !strings.Contains(headerText, needle) {
			t.Fatalf("expected %q in header xml: %s", needle, headerText)
		}
	}

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "둘째 문단") {
		t.Fatalf("expected target paragraph text in section xml: %s", sectionText)
	}
	if strings.Count(sectionText, "charPrIDRef=\"0\"") < 1 {
		t.Fatalf("expected at least one default run to remain: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "첫 문단\n둘째 문단"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestFindRunsByStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "set-text-style", editableDir, "--paragraph", "1", "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--font-name", "맑은 고딕", "--font-size-pt", "12", "--format", "json")

	searchStdout := runCLI(t, "find-runs-by-style", editableDir, "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--font-name", "맑은 고딕", "--font-size-pt", "12", "--format", "json")
	var searchEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Paragraph  int     `json:"paragraph"`
				Run        int     `json:"run"`
				Text       string  `json:"text"`
				Bold       bool    `json:"bold"`
				Underline  bool    `json:"underline"`
				TextColor  string  `json:"textColor"`
				FontName   string  `json:"fontName"`
				FontSizePt float64 `json:"fontSizePt"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchStdout.Bytes(), &searchEnvelope); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if !searchEnvelope.Success || searchEnvelope.Data.Count == 0 {
		t.Fatalf("unexpected search response: %s", searchStdout.String())
	}
	match := searchEnvelope.Data.Matches[0]
	if match.Paragraph != 1 || match.Text != "둘째 문단" || !match.Bold || !match.Underline || match.TextColor != "#C00000" || match.FontName != "맑은 고딕" || match.FontSizePt != 12 {
		t.Fatalf("unexpected match: %+v", match)
	}

	emptyStdout := runCLI(t, "find-runs-by-style", editableDir, "--font-name", "없는 글꼴", "--format", "json")
	var emptyEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Paragraph int `json:"paragraph"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(emptyStdout.Bytes(), &emptyEnvelope); err != nil {
		t.Fatalf("decode empty search response: %v", err)
	}
	if !emptyEnvelope.Success || emptyEnvelope.Data.Count != 0 {
		t.Fatalf("unexpected empty search response: %s", emptyStdout.String())
	}
	if emptyEnvelope.Data.Matches == nil {
		t.Fatalf("expected empty matches slice, got nil: %s", emptyStdout.String())
	}
	if len(emptyEnvelope.Data.Matches) != 0 {
		t.Fatalf("expected zero matches, got: %+v", emptyEnvelope.Data.Matches)
	}
}

func TestReplaceRunsByStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")
	runCLI(t, "set-text-style", editableDir, "--paragraph", "1", "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--format", "json")

	replaceStdout := runCLI(t, "replace-runs-by-style", editableDir, "--bold", "true", "--underline", "true", "--text-color", "#C00000", "--text", "[강조]", "--format", "json")
	var replaceEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count        int `json:"count"`
			Replacements []struct {
				Paragraph    int    `json:"paragraph"`
				Run          int    `json:"run"`
				PreviousText string `json:"previousText"`
				Text         string `json:"text"`
				CharPrIDRef  string `json:"charPrIdRef"`
			} `json:"replacements"`
		} `json:"data"`
	}
	if err := json.Unmarshal(replaceStdout.Bytes(), &replaceEnvelope); err != nil {
		t.Fatalf("decode replace response: %v", err)
	}
	if !replaceEnvelope.Success || replaceEnvelope.Data.Count != 1 {
		t.Fatalf("unexpected replace response: %s", replaceStdout.String())
	}
	replacement := replaceEnvelope.Data.Replacements[0]
	if replacement.Paragraph != 1 || replacement.Run != 0 || replacement.PreviousText != "둘째 문단" || replacement.Text != "[강조]" {
		t.Fatalf("unexpected replacement: %+v", replacement)
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Contains(sectionText, "둘째 문단") {
		t.Fatalf("expected previous run text to be replaced: %s", sectionText)
	}
	if !strings.Contains(sectionText, "[강조]") {
		t.Fatalf("expected replacement run text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if want := "첫 문단\n[강조]"; textEnvelope.Data.Text != want {
		t.Fatalf("unexpected packed text: %q", textEnvelope.Data.Text)
	}
}

func TestFindObjectsWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	imagePath := filepath.Join(workDir, "pixel.png")

	if err := os.WriteFile(imagePath, mustTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "insert-image", editableDir, "--image", imagePath, "--width-mm", "20", "--format", "json")
	runCLI(t, "add-equation", editableDir, "--script", "a+b", "--format", "json")
	runCLI(t, "add-rectangle", editableDir, "--width-mm", "40", "--height-mm", "20", "--format", "json")
	runCLI(t, "add-line", editableDir, "--width-mm", "50", "--height-mm", "10", "--format", "json")
	runCLI(t, "add-ellipse", editableDir, "--width-mm", "40", "--height-mm", "20", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--format", "json")

	searchStdout := runCLI(t, "find-objects", editableDir, "--format", "json")
	var searchEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Type      string `json:"type"`
				Paragraph int    `json:"paragraph"`
				Run       int    `json:"run"`
				Path      string `json:"path"`
				Ref       string `json:"ref"`
				Text      string `json:"text"`
				Rows      int    `json:"rows"`
				Cols      int    `json:"cols"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchStdout.Bytes(), &searchEnvelope); err != nil {
		t.Fatalf("decode object search response: %v", err)
	}
	if !searchEnvelope.Success || searchEnvelope.Data.Count != 8 {
		t.Fatalf("unexpected object search response: %s", searchStdout.String())
	}

	types := map[string]int{}
	var sawNestedTable bool
	var sawImageRef bool
	var sawTextboxText bool
	for _, match := range searchEnvelope.Data.Matches {
		types[match.Type]++
		if match.Type == "table" && match.Rows == 2 && match.Cols == 2 && strings.Contains(match.Text, "내부1") {
			sawNestedTable = true
		}
		if match.Type == "image" && match.Ref != "" {
			sawImageRef = true
		}
		if match.Type == "textbox" && strings.Contains(match.Text, "글상자 첫 줄") {
			sawTextboxText = true
		}
	}
	for _, objectType := range []string{"table", "image", "equation", "rectangle", "line", "ellipse", "textbox"} {
		if types[objectType] == 0 {
			t.Fatalf("expected object type %q in matches: %+v", objectType, types)
		}
	}
	if types["table"] != 2 || !sawNestedTable || !sawImageRef || !sawTextboxText {
		t.Fatalf("unexpected object details: %+v", searchEnvelope.Data.Matches)
	}

	filteredStdout := runCLI(t, "find-objects", editableDir, "--type", "table,textbox", "--format", "json")
	var filteredEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Type string `json:"type"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(filteredStdout.Bytes(), &filteredEnvelope); err != nil {
		t.Fatalf("decode filtered object search response: %v", err)
	}
	if !filteredEnvelope.Success || filteredEnvelope.Data.Count != 3 {
		t.Fatalf("unexpected filtered object search response: %s", filteredStdout.String())
	}
	for _, match := range filteredEnvelope.Data.Matches {
		if match.Type != "table" && match.Type != "textbox" {
			t.Fatalf("unexpected filtered object type: %+v", match)
		}
	}
}

func TestFindByTagWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--format", "json")

	tableStdout := runCLI(t, "find-by-tag", editableDir, "--tag", "hp:tbl", "--format", "json")
	var tableEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag  string `json:"tag"`
				Text string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(tableStdout.Bytes(), &tableEnvelope); err != nil {
		t.Fatalf("decode tag search response: %v", err)
	}
	if !tableEnvelope.Success || tableEnvelope.Data.Count != 2 {
		t.Fatalf("unexpected table tag search response: %s", tableStdout.String())
	}
	for _, match := range tableEnvelope.Data.Matches {
		if !strings.HasSuffix(match.Tag, "tbl") {
			t.Fatalf("unexpected table tag: %+v", match)
		}
	}

	drawTextStdout := runCLI(t, "find-by-tag", editableDir, "--tag", "drawText", "--format", "json")
	var drawTextEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag  string `json:"tag"`
				Text string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(drawTextStdout.Bytes(), &drawTextEnvelope); err != nil {
		t.Fatalf("decode drawText search response: %v", err)
	}
	if !drawTextEnvelope.Success || drawTextEnvelope.Data.Count != 1 {
		t.Fatalf("unexpected drawText search response: %s", drawTextStdout.String())
	}
	if !strings.HasSuffix(drawTextEnvelope.Data.Matches[0].Tag, "drawText") || !strings.Contains(drawTextEnvelope.Data.Matches[0].Text, "글상자 첫 줄") {
		t.Fatalf("unexpected drawText match: %+v", drawTextEnvelope.Data.Matches[0])
	}
}

func TestFindByAttrWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--format", "json")

	idStdout := runCLI(t, "find-by-attr", editableDir, "--attr", "id", "--tag", "tbl", "--format", "json")
	var idEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag   string `json:"tag"`
				Attr  string `json:"attr"`
				Value string `json:"value"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(idStdout.Bytes(), &idEnvelope); err != nil {
		t.Fatalf("decode attr search response: %v", err)
	}
	if !idEnvelope.Success || idEnvelope.Data.Count != 2 {
		t.Fatalf("unexpected id attr search response: %s", idStdout.String())
	}
	for _, match := range idEnvelope.Data.Matches {
		if !strings.HasSuffix(match.Tag, "tbl") || match.Attr != "id" || strings.TrimSpace(match.Value) == "" {
			t.Fatalf("unexpected id attr match: %+v", match)
		}
	}

	editableStdout := runCLI(t, "find-by-attr", editableDir, "--attr", "editable", "--tag", "drawText", "--value", "0", "--format", "json")
	var editableEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag   string `json:"tag"`
				Attr  string `json:"attr"`
				Value string `json:"value"`
				Text  string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(editableStdout.Bytes(), &editableEnvelope); err != nil {
		t.Fatalf("decode editable attr search response: %v", err)
	}
	if !editableEnvelope.Success || editableEnvelope.Data.Count != 1 {
		t.Fatalf("unexpected editable attr search response: %s", editableStdout.String())
	}
	match := editableEnvelope.Data.Matches[0]
	if !strings.HasSuffix(match.Tag, "drawText") || match.Attr != "editable" || match.Value != "0" || !strings.Contains(match.Text, "글상자 첫 줄") {
		t.Fatalf("unexpected editable attr match: %+v", match)
	}
}

func TestFindByXPathWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--format", "json")

	tableStdout := runCLI(t, "find-by-xpath", editableDir, "--expr", ".//hp:tbl[@id]", "--format", "json")
	var tableEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Paragraph int    `json:"paragraph"`
				Run       int    `json:"run"`
				Tag       string `json:"tag"`
				Text      string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(tableStdout.Bytes(), &tableEnvelope); err != nil {
		t.Fatalf("decode xpath search response: %v", err)
	}
	if !tableEnvelope.Success || tableEnvelope.Data.Count != 2 {
		t.Fatalf("unexpected table xpath response: %s", tableStdout.String())
	}
	for _, match := range tableEnvelope.Data.Matches {
		if !strings.HasSuffix(match.Tag, "tbl") || match.Paragraph != 0 || match.Run != 0 {
			t.Fatalf("unexpected table xpath match: %+v", match)
		}
	}

	drawTextStdout := runCLI(t, "find-by-xpath", editableDir, "--expr", ".//hp:drawText[@editable='0']", "--format", "json")
	var drawTextEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				Tag  string `json:"tag"`
				Text string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(drawTextStdout.Bytes(), &drawTextEnvelope); err != nil {
		t.Fatalf("decode drawText xpath response: %v", err)
	}
	if !drawTextEnvelope.Success || drawTextEnvelope.Data.Count != 1 {
		t.Fatalf("unexpected drawText xpath response: %s", drawTextStdout.String())
	}
	if !strings.HasSuffix(drawTextEnvelope.Data.Matches[0].Tag, "drawText") || !strings.Contains(drawTextEnvelope.Data.Matches[0].Text, "글상자 첫 줄") {
		t.Fatalf("unexpected drawText xpath match: %+v", drawTextEnvelope.Data.Matches[0])
	}
}

func TestFindByTagAcrossSectionsIncludesCoordinates(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")
	copyFirstTableParagraphToSection(t, editableDir, 1)

	searchStdout := runCLI(t, "find-by-tag", editableDir, "--tag", "hp:tc", "--all-sections", "true", "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Matches []struct {
				SectionIndex int  `json:"sectionIndex"`
				Paragraph    int  `json:"paragraph"`
				TableIndex   *int `json:"tableIndex"`
				Cell         *struct {
					Row int `json:"row"`
					Col int `json:"col"`
				} `json:"cell"`
				Tag  string `json:"tag"`
				Text string `json:"text"`
			} `json:"matches"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode multi-section tag search response: %v", err)
	}
	if !envelope.Success || envelope.Data.Count != 8 {
		t.Fatalf("unexpected multi-section tag response: %s", searchStdout.String())
	}

	sectionCounts := map[int]int{}
	foundSectionOneCell := false
	for _, match := range envelope.Data.Matches {
		if !strings.HasSuffix(match.Tag, "tc") {
			t.Fatalf("unexpected tag match: %+v", match)
		}
		if match.TableIndex == nil || *match.TableIndex != 0 {
			t.Fatalf("expected section-local table index 0: %+v", match)
		}
		if match.Cell == nil {
			t.Fatalf("expected cell coordinates: %+v", match)
		}
		sectionCounts[match.SectionIndex]++
		if match.SectionIndex == 1 && match.Cell.Row == 1 && match.Cell.Col == 1 && strings.Contains(match.Text, "D") {
			foundSectionOneCell = true
		}
	}
	if sectionCounts[0] != 4 || sectionCounts[1] != 4 {
		t.Fatalf("unexpected section counts: %+v", sectionCounts)
	}
	if !foundSectionOneCell {
		t.Fatalf("expected to find section 1 bottom-right cell: %+v", envelope.Data.Matches)
	}
}

func TestSectionAwareParagraphAndRunEditWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 section 본문", "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")
	copyFirstEditableParagraphToSection(t, editableDir, 1)

	setParagraphStdout := runCLI(t, "set-paragraph-text", editableDir, "--section", "1", "--paragraph", "0", "--text", "둘째 section 초안", "--format", "json")
	var setParagraphEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			SectionIndex int `json:"sectionIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(setParagraphStdout.Bytes(), &setParagraphEnvelope); err != nil {
		t.Fatalf("decode set-paragraph-text response: %v", err)
	}
	if !setParagraphEnvelope.Success || setParagraphEnvelope.Data.SectionIndex != 1 {
		t.Fatalf("unexpected set-paragraph-text response: %s", setParagraphStdout.String())
	}

	setRunStdout := runCLI(t, "set-run-text", editableDir, "--section", "1", "--paragraph", "0", "--run", "0", "--text", "둘째 section 최종", "--format", "json")
	var setRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			SectionIndex int `json:"sectionIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(setRunStdout.Bytes(), &setRunEnvelope); err != nil {
		t.Fatalf("decode set-run-text response: %v", err)
	}
	if !setRunEnvelope.Success || setRunEnvelope.Data.SectionIndex != 1 {
		t.Fatalf("unexpected set-run-text response: %s", setRunStdout.String())
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionZeroBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section0 xml: %v", err)
	}
	if !strings.Contains(string(sectionZeroBytes), "첫 section 본문") {
		t.Fatalf("expected section0 text to remain unchanged: %s", string(sectionZeroBytes))
	}

	sectionOneBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section1.xml"))
	if err != nil {
		t.Fatalf("read section1 xml: %v", err)
	}
	if !strings.Contains(string(sectionOneBytes), "둘째 section 최종") {
		t.Fatalf("expected section1 text to be updated: %s", string(sectionOneBytes))
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success || !strings.Contains(textEnvelope.Data.Text, "첫 section 본문") || !strings.Contains(textEnvelope.Data.Text, "둘째 section 최종") {
		t.Fatalf("unexpected text output after section-aware edits: %s", textStdout.String())
	}
}

func TestSectionAwareParagraphMutationCommands(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 section 본문", "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")

	appendStdout := runCLI(t, "append-text", editableDir, "--section", "1", "--text", "둘째 section 첫 문단\n둘째 section 둘째 문단", "--format", "json")
	var appendEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			SectionIndex int `json:"sectionIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(appendStdout.Bytes(), &appendEnvelope); err != nil {
		t.Fatalf("decode append-text response: %v", err)
	}
	if !appendEnvelope.Success || appendEnvelope.Data.SectionIndex != 1 {
		t.Fatalf("unexpected append-text response: %s", appendStdout.String())
	}

	addRunStdout := runCLI(t, "add-run-text", editableDir, "--section", "1", "--paragraph", "0", "--text", " / 추가", "--format", "json")
	var addRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			SectionIndex int `json:"sectionIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(addRunStdout.Bytes(), &addRunEnvelope); err != nil {
		t.Fatalf("decode add-run-text response: %v", err)
	}
	if !addRunEnvelope.Success || addRunEnvelope.Data.SectionIndex != 1 {
		t.Fatalf("unexpected add-run-text response: %s", addRunStdout.String())
	}

	runCLI(t, "set-paragraph-layout", editableDir, "--section", "1", "--paragraph", "0", "--align", "CENTER", "--format", "json")
	runCLI(t, "set-paragraph-list", editableDir, "--section", "1", "--paragraph", "1", "--kind", "bullet", "--format", "json")
	runCLI(t, "set-text-style", editableDir, "--section", "1", "--paragraph", "0", "--bold", "true", "--format", "json")

	deleteStdout := runCLI(t, "delete-paragraph", editableDir, "--section", "1", "--paragraph", "1", "--format", "json")
	var deleteEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			SectionIndex int `json:"sectionIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(deleteStdout.Bytes(), &deleteEnvelope); err != nil {
		t.Fatalf("decode delete-paragraph response: %v", err)
	}
	if !deleteEnvelope.Success || deleteEnvelope.Data.SectionIndex != 1 {
		t.Fatalf("unexpected delete-paragraph response: %s", deleteStdout.String())
	}

	sectionZeroBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section0 xml: %v", err)
	}
	sectionZeroText := string(sectionZeroBytes)
	if !strings.Contains(sectionZeroText, "첫 section 본문") {
		t.Fatalf("expected section0 text to remain: %s", sectionZeroText)
	}
	if strings.Contains(sectionZeroText, "둘째 section 첫 문단") {
		t.Fatalf("expected section0 to exclude section1 paragraph edits: %s", sectionZeroText)
	}

	sectionOneDoc := etree.NewDocument()
	if err := sectionOneDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section1.xml")); err != nil {
		t.Fatalf("read section1 xml: %v", err)
	}
	sectionOneRoot := sectionOneDoc.Root()
	if sectionOneRoot == nil {
		t.Fatal("expected section1 root")
	}
	paragraphs := sectionOneRoot.FindElements("./hp:p")
	if len(paragraphs) < 2 {
		t.Fatalf("expected section1 paragraphs including section properties: %d", len(paragraphs))
	}
	if got := paragraphPlainTextForTest(paragraphs[1]); !strings.Contains(got, "둘째 section 첫 문단 / 추가") {
		t.Fatalf("expected updated section1 paragraph text, got %q", got)
	}
	if strings.Contains(sectionOneRoot.Text(), "둘째 section 둘째 문단") {
		t.Fatalf("expected deleted paragraph to be removed from section1 xml: %s", sectionOneRoot.Text())
	}
}

func TestSectionAwareTableMutationCommands(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "첫,섹션", "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")

	addTableStdout := runCLI(t, "add-table", editableDir, "--section", "1", "--cells", "A,B;C,D", "--format", "json")
	var addTableEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			SectionIndex int `json:"sectionIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(addTableStdout.Bytes(), &addTableEnvelope); err != nil {
		t.Fatalf("decode add-table response: %v", err)
	}
	if !addTableEnvelope.Success || addTableEnvelope.Data.SectionIndex != 1 {
		t.Fatalf("unexpected add-table response: %s", addTableStdout.String())
	}

	addNestedStdout := runCLI(t, "add-nested-table", editableDir, "--section", "1", "--table", "0", "--row", "1", "--col", "1", "--cells", "N", "--format", "json")
	var addNestedEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			SectionIndex int `json:"sectionIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(addNestedStdout.Bytes(), &addNestedEnvelope); err != nil {
		t.Fatalf("decode add-nested-table response: %v", err)
	}
	if !addNestedEnvelope.Success || addNestedEnvelope.Data.SectionIndex != 1 {
		t.Fatalf("unexpected add-nested-table response: %s", addNestedStdout.String())
	}

	runCLI(t, "set-table-cell-layout", editableDir, "--section", "1", "--table", "0", "--row", "0", "--col", "0", "--paragraph", "0", "--align", "CENTER", "--format", "json")
	runCLI(t, "set-table-cell-text-style", editableDir, "--section", "1", "--table", "0", "--row", "0", "--col", "1", "--paragraph", "0", "--bold", "true", "--format", "json")
	runCLI(t, "merge-table-cells", editableDir, "--section", "1", "--table", "0", "--start-row", "0", "--start-col", "0", "--end-row", "0", "--end-col", "1", "--format", "json")
	runCLI(t, "split-table-cell", editableDir, "--section", "1", "--table", "0", "--row", "0", "--col", "0", "--format", "json")

	normalizeStdout := runCLI(t, "normalize-table-borders", editableDir, "--section", "1", "--table", "0", "--format", "json")
	var normalizeEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			SectionIndex int `json:"sectionIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(normalizeStdout.Bytes(), &normalizeEnvelope); err != nil {
		t.Fatalf("decode normalize-table-borders response: %v", err)
	}
	if !normalizeEnvelope.Success || normalizeEnvelope.Data.SectionIndex != 1 {
		t.Fatalf("unexpected normalize-table-borders response: %s", normalizeStdout.String())
	}

	sectionZeroBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section0 xml: %v", err)
	}
	if strings.Contains(string(sectionZeroBytes), ">A<") || strings.Contains(string(sectionZeroBytes), ">N<") {
		t.Fatalf("expected section0 to exclude section1 table mutations: %s", string(sectionZeroBytes))
	}

	sectionOneBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section1.xml"))
	if err != nil {
		t.Fatalf("read section1 xml: %v", err)
	}
	sectionOneText := string(sectionOneBytes)
	for _, needle := range []string{">A<", ">C<", ">D<", ">N<", "<hp:tbl"} {
		if !strings.Contains(sectionOneText, needle) {
			t.Fatalf("expected %q in section1 xml: %s", needle, sectionOneText)
		}
	}
}

func TestReplaceRunsByStyleAcrossAllSections(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "Alpha", "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")
	copyFirstEditableParagraphToSection(t, editableDir, 1)
	runCLI(t, "set-paragraph-text", editableDir, "--section", "1", "--paragraph", "0", "--text", "Beta", "--format", "json")

	replaceStdout := runCLI(t, "replace-runs-by-style", editableDir, "--font-size-pt", "10", "--all-sections", "true", "--text", "[본문]", "--format", "json")
	var replaceEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count        int `json:"count"`
			Replacements []struct {
				SectionIndex int `json:"sectionIndex"`
			} `json:"replacements"`
		} `json:"data"`
	}
	if err := json.Unmarshal(replaceStdout.Bytes(), &replaceEnvelope); err != nil {
		t.Fatalf("decode replace-runs-by-style response: %v", err)
	}
	if !replaceEnvelope.Success || replaceEnvelope.Data.Count != 2 {
		t.Fatalf("unexpected replace-runs-by-style response: %s", replaceStdout.String())
	}
	sectionHits := map[int]int{}
	for _, replacement := range replaceEnvelope.Data.Replacements {
		sectionHits[replacement.SectionIndex]++
	}
	if sectionHits[0] != 1 || sectionHits[1] != 1 {
		t.Fatalf("expected replacements in both sections: %+v", sectionHits)
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success || textEnvelope.Data.Text != "[본문]\n[본문]" {
		t.Fatalf("unexpected text output after all-section replace: %s", textStdout.String())
	}
}

func TestSectionAddAndDeleteWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 section 본문", "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")
	runCLI(t, "add-section", editableDir, "--format", "json")
	runCLI(t, "delete-section", editableDir, "--section", "1", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	contentBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "content.hpf"))
	if err != nil {
		t.Fatalf("read content.hpf: %v", err)
	}
	contentText := string(contentBytes)
	if !strings.Contains(contentText, "section1.xml") {
		t.Fatalf("expected remaining section to be renumbered: %s", contentText)
	}
	if strings.Contains(contentText, "section2.xml") {
		t.Fatalf("expected stale section path to be removed: %s", contentText)
	}

	headerBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "header.xml"))
	if err != nil {
		t.Fatalf("read header.xml: %v", err)
	}
	if !strings.Contains(string(headerBytes), "secCnt=\"2\"") {
		t.Fatalf("expected header section count to be updated: %s", string(headerBytes))
	}

	if _, err := os.Stat(filepath.Join(editableDir, "Contents", "section2.xml")); err == nil || !os.IsNotExist(err) {
		t.Fatalf("expected stale section file to be removed: %v", err)
	}

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section1.xml"))
	if err != nil {
		t.Fatalf("read new section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"<hp:secPr", "<hp:t"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in new section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid   bool `json:"valid"`
				Summary struct {
					SectionPath []string `json:"sectionPaths"`
				} `json:"summary"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
	if want := []string{"Contents/section0.xml", "Contents/section1.xml"}; strings.Join(inspectEnvelope.Data.Report.Summary.SectionPath, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected section paths: %v", inspectEnvelope.Data.Report.Summary.SectionPath)
	}
}

func TestInsertImageCreatesVisiblePictureXML(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	imagePath := filepath.Join(workDir, "pixel.png")
	archivePath := filepath.Join(workDir, "result.hwpx")

	if err := os.WriteFile(imagePath, mustTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	insertStdout := runCLI(t, "insert-image", editableDir, "--image", imagePath, "--width-mm", "40", "--format", "json")

	var insertEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			ItemID       string `json:"itemId"`
			PlacedWidth  int    `json:"placedWidth"`
			PlacedHeight int    `json:"placedHeight"`
		} `json:"data"`
	}
	if err := json.Unmarshal(insertStdout.Bytes(), &insertEnvelope); err != nil {
		t.Fatalf("decode insert-image response: %v", err)
	}
	if !insertEnvelope.Success || insertEnvelope.Data.ItemID == "" {
		t.Fatalf("unexpected insert-image response: %s", insertStdout.String())
	}
	if insertEnvelope.Data.PlacedWidth <= 0 || insertEnvelope.Data.PlacedHeight <= 0 {
		t.Fatalf("unexpected inserted size: %+v", insertEnvelope.Data)
	}

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "<hp:pic") {
		t.Fatalf("expected visible picture xml: %s", sectionText)
	}
	if !strings.Contains(sectionText, "binaryItemIDRef=\""+insertEnvelope.Data.ItemID+"\"") {
		t.Fatalf("expected picture to reference embedded image id: %s", sectionText)
	}

	contentBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "content.hpf"))
	if err != nil {
		t.Fatalf("read content.hpf: %v", err)
	}
	contentText := string(contentBytes)
	if !strings.Contains(contentText, "isEmbeded=\"1\"") {
		t.Fatalf("expected embedded image manifest flag: %s", contentText)
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestHeaderFooterAndPageNumberWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "set-header", editableDir, "--text", "문서 머리말", "--format", "json")
	runCLI(t, "set-footer", editableDir, "--text", "문서 꼬리말", "--format", "json")
	runCLI(t, "set-page-number", editableDir, "--position", "BOTTOM_CENTER", "--type", "DIGIT", "--side-char", "-", "--start-page", "3", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:header",
		"문서 머리말",
		"<hp:footer",
		"문서 꼬리말",
		"<hp:pageNum",
		"pos=\"BOTTOM_CENTER\"",
		"formatType=\"DIGIT\"",
		"sideChar=\"-\"",
		"pageStartsOn=\"BOTH\"",
		"page=\"3\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestRemoveHeaderFooterPreservesPageNumber(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "set-header", editableDir, "--text", "문서 머리말", "--format", "json")
	runCLI(t, "set-footer", editableDir, "--text", "문서 꼬리말", "--format", "json")
	runCLI(t, "set-page-number", editableDir, "--position", "BOTTOM_CENTER", "--type", "DIGIT", "--format", "json")
	runCLI(t, "remove-header", editableDir, "--format", "json")
	runCLI(t, "remove-footer", editableDir, "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"<hp:header", "<hp:footer", "문서 머리말", "문서 꼬리말"} {
		if strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q to be removed from section xml: %s", needle, sectionText)
		}
	}
	for _, needle := range []string{"<hp:pageNum", "pos=\"BOTTOM_CENTER\"", "formatType=\"DIGIT\""} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q to remain in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestSetColumnsWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "set-columns", editableDir, "--count", "2", "--gap-mm", "8", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"spaceColumns=\"2268\"",
		"<hp:colPr",
		"colCount=\"2\"",
		"sameGap=\"2268\"",
		"<hp:colLine",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestSetPageLayoutWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(
		t,
		"set-page-layout", editableDir,
		"--orientation", "LANDSCAPE",
		"--width-mm", "297",
		"--height-mm", "210",
		"--left-margin-mm", "15",
		"--right-margin-mm", "15",
		"--top-margin-mm", "10",
		"--bottom-margin-mm", "10",
		"--header-margin-mm", "5",
		"--footer-margin-mm", "5",
		"--gutter-margin-mm", "3",
		"--gutter-type", "LEFT_ONLY",
		"--border-fill-id-ref", "2",
		"--border-text-border", "CONTENT",
		"--border-fill-area", "BORDER",
		"--border-header-inside", "true",
		"--border-footer-inside", "false",
		"--border-offset-left-mm", "2",
		"--border-offset-right-mm", "2",
		"--border-offset-top-mm", "2",
		"--border-offset-bottom-mm", "2",
		"--format", "json",
	)
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:pagePr",
		"landscape=\"WIDELY\"",
		"width=\"84189\"",
		"height=\"59528\"",
		"gutterType=\"LEFT_ONLY\"",
		"<hp:margin",
		"left=\"4252\"",
		"right=\"4252\"",
		"top=\"2835\"",
		"bottom=\"2835\"",
		"header=\"1417\"",
		"footer=\"1417\"",
		"gutter=\"850\"",
		"textBorder=\"CONTENT\"",
		"fillArea=\"BORDER\"",
		"headerInside=\"1\"",
		"footerInside=\"0\"",
		"borderFillIDRef=\"2\"",
		"<hp:offset left=\"567\" right=\"567\" top=\"567\" bottom=\"567\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}
	if count := strings.Count(sectionText, "borderFillIDRef=\"2\""); count != 3 {
		t.Fatalf("expected border fill to update all page types, got %d: %s", count, sectionText)
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestTableMergeAndSplitWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "merge-table-cells", editableDir, "--table", "0", "--start-row", "0", "--start-col", "0", "--end-row", "1", "--end-col", "1", "--format", "json")
	runCLI(t, "set-table-cell", editableDir, "--table", "0", "--row", "1", "--col", "1", "--text", "병합 셀", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"rowSpan=\"2\"",
		"colSpan=\"2\"",
		"병합 셀",
		"width=\"0\"",
		"height=\"0\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after merge: %s", needle, sectionText)
		}
	}

	runCLI(t, "split-table-cell", editableDir, "--table", "0", "--row", "0", "--col", "0", "--format", "json")
	runCLI(t, "set-table-cell", editableDir, "--table", "0", "--row", "1", "--col", "1", "--text", "분리 후 D", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err = os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml after split: %v", err)
	}
	sectionText = string(sectionBytes)
	if strings.Contains(sectionText, "rowSpan=\"2\"") || strings.Contains(sectionText, "colSpan=\"2\"") {
		t.Fatalf("expected merged span to be removed after split: %s", sectionText)
	}
	if !strings.Contains(sectionText, "분리 후 D") {
		t.Fatalf("expected split cell text in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{"병합 셀", "분리 후 D"} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestAddTableWithGeometryOptions(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(
		t,
		"add-table", editableDir,
		"--cells", "A,B,C;D,E,F",
		"--width-mm", "60",
		"--height-mm", "12",
		"--col-widths-mm", "10,20,30",
		"--row-heights-mm", "5,7",
		"--margin-left-mm", "1.5",
		"--margin-right-mm", "2.5",
		"--margin-top-mm", "3.5",
		"--margin-bottom-mm", "4.5",
		"--format", "json",
	)

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}

	root := sectionDoc.Root()
	if root == nil {
		t.Fatal("expected section root")
	}
	table := root.FindElement(".//hp:tbl")
	if table == nil {
		t.Fatal("expected table in section xml")
	}

	size := table.FindElement("./hp:sz")
	if size == nil {
		t.Fatal("expected table size")
	}
	if got := size.SelectAttrValue("width", ""); got != "17008" {
		t.Fatalf("unexpected table width: %s", got)
	}
	if got := size.SelectAttrValue("height", ""); got != "3401" {
		t.Fatalf("unexpected table height: %s", got)
	}

	outMargin := table.FindElement("./hp:outMargin")
	if outMargin == nil {
		t.Fatal("expected table outMargin")
	}
	expectedMargins := map[string]string{
		"left":   "425",
		"right":  "709",
		"top":    "992",
		"bottom": "1276",
	}
	for key, want := range expectedMargins {
		if got := outMargin.SelectAttrValue(key, ""); got != want {
			t.Fatalf("unexpected table outMargin %s: %s", key, got)
		}
	}

	rows := root.FindElements(".//hp:tbl/hp:tr")
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}

	firstRowCells := rows[0].FindElements("./hp:tc")
	if len(firstRowCells) != 3 {
		t.Fatalf("expected 3 cells in first row, got %d", len(firstRowCells))
	}
	expectedWidths := []string{"2835", "5669", "8504"}
	for index, want := range expectedWidths {
		cellSize := firstRowCells[index].FindElement("./hp:cellSz")
		if cellSize == nil {
			t.Fatalf("expected cell size for column %d", index)
		}
		if got := cellSize.SelectAttrValue("width", ""); got != want {
			t.Fatalf("unexpected width for column %d: %s", index, got)
		}
		if got := cellSize.SelectAttrValue("height", ""); got != "1417" {
			t.Fatalf("unexpected height for first row column %d: %s", index, got)
		}
	}

	secondRowCells := rows[1].FindElements("./hp:tc")
	for index, cell := range secondRowCells {
		cellSize := cell.FindElement("./hp:cellSz")
		if cellSize == nil {
			t.Fatalf("expected second-row cell size for column %d", index)
		}
		if got := cellSize.SelectAttrValue("height", ""); got != "1984" {
			t.Fatalf("unexpected second-row height for column %d: %s", index, got)
		}
	}
}

func TestTableCellStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,내용;이름,홍길동", "--format", "json")

	styleStdout := runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--text", "신청인",
		"--vert-align", "TOP",
		"--margin-left-mm", "1.5",
		"--margin-right-mm", "1.5",
		"--margin-top-mm", "0.8",
		"--margin-bottom-mm", "0.8",
		"--border-color", "#2F5597",
		"--border-width-mm", "0.3",
		"--fill-color", "#FFF2CC",
		"--format", "json",
	)
	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "1",
		"--col", "0",
		"--background-color", "#D9EAD3",
		"--vert-align", "BOTTOM",
		"--format", "json",
	)
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text        *string  `json:"text"`
			VertAlign   string   `json:"vertAlign"`
			FillColor   string   `json:"fillColor"`
			BorderColor string   `json:"borderColor"`
			BorderWidth *float64 `json:"borderWidthMm"`
		} `json:"data"`
	}
	if err := json.Unmarshal(styleStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode set-table-cell response: %v", err)
	}
	if !envelope.Success || envelope.Data.Text == nil || *envelope.Data.Text != "신청인" {
		t.Fatalf("unexpected set-table-cell response: %s", styleStdout.String())
	}
	if envelope.Data.VertAlign != "TOP" || envelope.Data.FillColor != "#FFF2CC" || envelope.Data.BorderColor != "#2F5597" {
		t.Fatalf("unexpected style response: %s", styleStdout.String())
	}
	if envelope.Data.BorderWidth == nil || *envelope.Data.BorderWidth != 0.3 {
		t.Fatalf("expected border width in response: %s", styleStdout.String())
	}

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	firstCell := sectionDoc.FindElement("//hp:tbl/hp:tr/hp:tc")
	if firstCell == nil {
		t.Fatalf("styled cell not found")
	}
	if firstCell.SelectAttrValue("hasMargin", "") != "1" {
		t.Fatalf("expected cell margin flag")
	}
	if firstCell.SelectAttrValue("borderFillIDRef", "") == "" || firstCell.SelectAttrValue("borderFillIDRef", "") == "3" {
		t.Fatalf("expected custom borderFill on styled cell")
	}
	cellMargin := firstCell.FindElement("./hp:cellMargin")
	if cellMargin == nil {
		t.Fatalf("cell margin element missing")
	}
	for key, want := range map[string]string{
		"left":   "425",
		"right":  "425",
		"top":    "227",
		"bottom": "227",
	} {
		if got := cellMargin.SelectAttrValue(key, ""); got != want {
			t.Fatalf("unexpected %s margin: got=%s want=%s", key, got, want)
		}
	}
	firstSubList := firstCell.FindElement("./hp:subList")
	if firstSubList == nil || firstSubList.SelectAttrValue("vertAlign", "") != "TOP" {
		t.Fatalf("expected TOP vertAlign on styled cell")
	}
	secondRowFirstCell := sectionDoc.FindElement("//hp:tbl/hp:tr[2]/hp:tc")
	if secondRowFirstCell == nil {
		t.Fatalf("second styled cell not found")
	}
	secondSubList := secondRowFirstCell.FindElement("./hp:subList")
	if secondSubList == nil || secondSubList.SelectAttrValue("vertAlign", "") != "BOTTOM" {
		t.Fatalf("expected BOTTOM vertAlign on second styled cell")
	}
	if secondRowFirstCell.SelectAttrValue("borderFillIDRef", "") == "3" {
		t.Fatalf("expected custom background borderFill on second styled cell")
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	firstBorderFillID := firstCell.SelectAttrValue("borderFillIDRef", "")
	borderFill := headerDoc.FindElement("//hh:borderFill[@id='" + firstBorderFillID + "']")
	if borderFill == nil {
		t.Fatalf("styled borderFill not found: %s", firstBorderFillID)
	}
	leftBorder := borderFill.FindElement("./hh:leftBorder")
	if leftBorder == nil {
		t.Fatalf("left border missing")
	}
	if leftBorder.SelectAttrValue("type", "") != "SOLID" ||
		leftBorder.SelectAttrValue("width", "") != "0.3 mm" ||
		leftBorder.SelectAttrValue("color", "") != "#2F5597" {
		t.Fatalf("unexpected border styling")
	}
	winBrush := borderFill.FindElement("./hc:fillBrush/hc:winBrush")
	if winBrush == nil || winBrush.SelectAttrValue("faceColor", "") != "#FFF2CC" {
		t.Fatalf("unexpected fill styling")
	}

	secondBorderFillID := secondRowFirstCell.SelectAttrValue("borderFillIDRef", "")
	secondFill := headerDoc.FindElement("//hh:borderFill[@id='" + secondBorderFillID + "']/hc:fillBrush/hc:winBrush")
	if secondFill == nil || secondFill.SelectAttrValue("faceColor", "") != "#D9EAD3" {
		t.Fatalf("unexpected background-color alias styling")
	}
}

func TestTableCellSideBorderWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")

	styleStdout := runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--border-style", "NONE",
		"--border-color", "#808080",
		"--border-left-style", "SOLID",
		"--border-left-color", "#000000",
		"--border-left-width-mm", "0.4",
		"--border-top-style", "SOLID",
		"--border-top-color", "#000000",
		"--border-top-width-mm", "0.4",
		"--border-right-width-mm", "0.12",
		"--border-bottom-width-mm", "0.12",
		"--format", "json",
	)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			BorderStyle         string   `json:"borderStyle"`
			BorderColor         string   `json:"borderColor"`
			BorderLeftStyle     string   `json:"borderLeftStyle"`
			BorderLeftColor     string   `json:"borderLeftColor"`
			BorderLeftWidthMM   *float64 `json:"borderLeftWidthMm"`
			BorderTopStyle      string   `json:"borderTopStyle"`
			BorderTopColor      string   `json:"borderTopColor"`
			BorderTopWidthMM    *float64 `json:"borderTopWidthMm"`
			BorderRightWidthMM  *float64 `json:"borderRightWidthMm"`
			BorderBottomWidthMM *float64 `json:"borderBottomWidthMm"`
		} `json:"data"`
	}
	if err := json.Unmarshal(styleStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode set-table-cell side border response: %v", err)
	}
	if !envelope.Success {
		t.Fatalf("unexpected side border response: %s", styleStdout.String())
	}
	if envelope.Data.BorderStyle != "NONE" || envelope.Data.BorderLeftStyle != "SOLID" || envelope.Data.BorderTopStyle != "SOLID" {
		t.Fatalf("unexpected side border styles: %s", styleStdout.String())
	}
	if envelope.Data.BorderLeftColor != "#000000" || envelope.Data.BorderTopColor != "#000000" || envelope.Data.BorderColor != "#808080" {
		t.Fatalf("unexpected side border colors: %s", styleStdout.String())
	}
	if envelope.Data.BorderLeftWidthMM == nil || *envelope.Data.BorderLeftWidthMM != 0.4 {
		t.Fatalf("expected left border width in response: %s", styleStdout.String())
	}
	if envelope.Data.BorderTopWidthMM == nil || *envelope.Data.BorderTopWidthMM != 0.4 {
		t.Fatalf("expected top border width in response: %s", styleStdout.String())
	}
	if envelope.Data.BorderRightWidthMM == nil || *envelope.Data.BorderRightWidthMM != 0.12 {
		t.Fatalf("expected right border width in response: %s", styleStdout.String())
	}
	if envelope.Data.BorderBottomWidthMM == nil || *envelope.Data.BorderBottomWidthMM != 0.12 {
		t.Fatalf("expected bottom border width in response: %s", styleStdout.String())
	}

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	firstCell := sectionDoc.FindElement("//hp:tbl/hp:tr/hp:tc")
	if firstCell == nil {
		t.Fatalf("styled cell not found")
	}
	borderFillID := firstCell.SelectAttrValue("borderFillIDRef", "")
	if borderFillID == "" || borderFillID == "3" {
		t.Fatalf("expected custom borderFill on side-styled cell")
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	borderFill := headerDoc.FindElement("//hh:borderFill[@id='" + borderFillID + "']")
	if borderFill == nil {
		t.Fatalf("side borderFill not found: %s", borderFillID)
	}

	assertBorder := func(tag, wantType, wantWidth, wantColor string) {
		t.Helper()
		line := borderFill.FindElement("./" + tag)
		if line == nil {
			t.Fatalf("%s missing", tag)
		}
		if line.SelectAttrValue("type", "") != wantType ||
			line.SelectAttrValue("width", "") != wantWidth ||
			line.SelectAttrValue("color", "") != wantColor {
			t.Fatalf("unexpected %s styling: type=%s width=%s color=%s", tag, line.SelectAttrValue("type", ""), line.SelectAttrValue("width", ""), line.SelectAttrValue("color", ""))
		}
	}

	assertBorder("hh:leftBorder", "SOLID", "0.4 mm", "#000000")
	assertBorder("hh:topBorder", "SOLID", "0.4 mm", "#000000")
	assertBorder("hh:rightBorder", "NONE", "0.12 mm", "#808080")
	assertBorder("hh:bottomBorder", "NONE", "0.12 mm", "#808080")
}

func TestTableCellRichBorderStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B", "--format", "json")

	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--border-style", "DOUBLE",
		"--border-top-style", "DASH",
		"--format", "json",
	)

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	firstCell := sectionDoc.FindElement("//hp:tbl/hp:tr/hp:tc")
	if firstCell == nil {
		t.Fatalf("styled cell not found")
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	borderFillID := firstCell.SelectAttrValue("borderFillIDRef", "")
	borderFill := headerDoc.FindElement("//hh:borderFill[@id='" + borderFillID + "']")
	if borderFill == nil {
		t.Fatalf("rich borderFill not found: %s", borderFillID)
	}

	assertBorder := func(tag, wantType string) {
		t.Helper()
		line := borderFill.FindElement("./" + tag)
		if line == nil {
			t.Fatalf("%s missing", tag)
		}
		if line.SelectAttrValue("type", "") != wantType {
			t.Fatalf("unexpected %s type: %s", tag, line.SelectAttrValue("type", ""))
		}
	}

	assertBorder("hh:leftBorder", "DOUBLE_SLIM")
	assertBorder("hh:rightBorder", "DOUBLE_SLIM")
	assertBorder("hh:bottomBorder", "DOUBLE_SLIM")
	assertBorder("hh:topBorder", "DASH")
}

func TestNormalizeTableBordersWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--rows", "2", "--cols", "3", "--format", "json")
	runCLI(t, "merge-table-cells", editableDir, "--table", "0", "--start-row", "0", "--start-col", "0", "--end-row", "0", "--end-col", "1", "--format", "json")
	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--border-right-style", "SOLID",
		"--border-right-color", "#111111",
		"--border-right-width-mm", "0.4",
		"--format", "json",
	)
	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "2",
		"--border-left-style", "SOLID",
		"--border-left-color", "#808080",
		"--border-left-width-mm", "0.12",
		"--format", "json",
	)

	normalizeStdout := runCLI(
		t,
		"normalize-table-borders", editableDir,
		"--table", "0",
		"--format", "json",
	)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			TableIndex int `json:"tableIndex"`
		} `json:"data"`
	}
	if err := json.Unmarshal(normalizeStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode normalize-table-borders response: %v", err)
	}
	if !envelope.Success || envelope.Data.TableIndex != 0 {
		t.Fatalf("unexpected normalize-table-borders response: %s", normalizeStdout.String())
	}

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	leftCell := sectionDoc.FindElement("//hp:tbl/hp:tr/hp:tc[1]")
	rightCell := sectionDoc.FindElement("//hp:tbl/hp:tr/hp:tc[3]")
	if leftCell == nil || rightCell == nil {
		t.Fatalf("table cells not found after normalization")
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}

	assertBorder := func(borderFillID, tag, wantType, wantWidth, wantColor string) {
		t.Helper()
		borderFill := headerDoc.FindElement("//hh:borderFill[@id='" + borderFillID + "']")
		if borderFill == nil {
			t.Fatalf("borderFill not found: %s", borderFillID)
		}
		line := borderFill.FindElement("./" + tag)
		if line == nil {
			t.Fatalf("%s missing on %s", tag, borderFillID)
		}
		if line.SelectAttrValue("type", "") != wantType ||
			line.SelectAttrValue("width", "") != wantWidth ||
			line.SelectAttrValue("color", "") != wantColor {
			t.Fatalf("unexpected %s styling on %s: type=%s width=%s color=%s", tag, borderFillID, line.SelectAttrValue("type", ""), line.SelectAttrValue("width", ""), line.SelectAttrValue("color", ""))
		}
	}

	assertBorder(leftCell.SelectAttrValue("borderFillIDRef", ""), "hh:rightBorder", "SOLID", "0.4 mm", "#111111")
	assertBorder(rightCell.SelectAttrValue("borderFillIDRef", ""), "hh:leftBorder", "SOLID", "0.4 mm", "#111111")
}

func TestNormalizeTableBordersPerimeterWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--rows", "2", "--cols", "2", "--format", "json")
	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--border-top-style", "DOUBLE",
		"--border-top-width-mm", "0.5",
		"--border-top-color", "#000000",
		"--format", "json",
	)
	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--border-left-style", "SOLID",
		"--border-left-width-mm", "0.4",
		"--border-left-color", "#000000",
		"--format", "json",
	)
	runCLI(t, "normalize-table-borders", editableDir, "--table", "0", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	topRightCell := sectionDoc.FindElement("//hp:tbl/hp:tr[1]/hp:tc[2]")
	bottomLeftCell := sectionDoc.FindElement("//hp:tbl/hp:tr[2]/hp:tc[1]")
	if topRightCell == nil || bottomLeftCell == nil {
		t.Fatalf("perimeter cells not found")
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}

	assertEdgeType := func(cell *etree.Element, tag, wantType string) {
		t.Helper()
		borderFillID := cell.SelectAttrValue("borderFillIDRef", "")
		borderFill := headerDoc.FindElement("//hh:borderFill[@id='" + borderFillID + "']")
		if borderFill == nil {
			t.Fatalf("borderFill not found: %s", borderFillID)
		}
		line := borderFill.FindElement("./" + tag)
		if line == nil {
			t.Fatalf("%s missing", tag)
		}
		if got := line.SelectAttrValue("type", ""); got != wantType {
			t.Fatalf("unexpected %s type: %s", tag, got)
		}
	}

	assertEdgeType(topRightCell, "hh:topBorder", "DOUBLE_SLIM")
	assertEdgeType(bottomLeftCell, "hh:leftBorder", "SOLID")
}

func TestMergeTableCellsPromotesMergedStyleAndSplitClonesAnchorStyle(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--rows", "2", "--cols", "2", "--format", "json")
	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "1",
		"--fill-color", "#D6D6D6",
		"--border-top-style", "DOUBLE",
		"--border-top-width-mm", "0.5",
		"--border-top-color", "#000000",
		"--border-right-style", "SOLID",
		"--border-right-width-mm", "0.4",
		"--border-right-color", "#000000",
		"--format", "json",
	)
	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "1",
		"--col", "0",
		"--border-left-style", "SOLID",
		"--border-left-width-mm", "0.4",
		"--border-left-color", "#000000",
		"--border-bottom-style", "SOLID",
		"--border-bottom-width-mm", "0.4",
		"--border-bottom-color", "#000000",
		"--format", "json",
	)
	runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--text", "대표 셀",
		"--font-name", "맑은 고딕",
		"--font-size-pt", "13",
		"--format", "json",
	)
	runCLI(t, "merge-table-cells", editableDir, "--table", "0", "--start-row", "0", "--start-col", "0", "--end-row", "1", "--end-col", "1", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml after merge: %v", err)
	}
	anchorCell := sectionDoc.FindElement("//hp:tbl/hp:tr[1]/hp:tc[1]")
	if anchorCell == nil {
		t.Fatalf("merged anchor cell not found")
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml after merge: %v", err)
	}

	borderFillID := anchorCell.SelectAttrValue("borderFillIDRef", "")
	borderFill := headerDoc.FindElement("//hh:borderFill[@id='" + borderFillID + "']")
	if borderFill == nil {
		t.Fatalf("merged borderFill not found: %s", borderFillID)
	}

	assertMergedBorder := func(tag, wantType, wantWidth string) {
		t.Helper()
		line := borderFill.FindElement("./" + tag)
		if line == nil {
			t.Fatalf("%s missing on merged borderFill", tag)
		}
		if got := line.SelectAttrValue("type", ""); got != wantType {
			t.Fatalf("unexpected %s type: %s", tag, got)
		}
		if wantWidth != "" && line.SelectAttrValue("width", "") != wantWidth {
			t.Fatalf("unexpected %s width: %s", tag, line.SelectAttrValue("width", ""))
		}
	}

	assertMergedBorder("hh:topBorder", "DOUBLE_SLIM", "0.5 mm")
	assertMergedBorder("hh:rightBorder", "SOLID", "0.4 mm")
	assertMergedBorder("hh:leftBorder", "SOLID", "0.4 mm")
	assertMergedBorder("hh:bottomBorder", "SOLID", "0.4 mm")

	winBrush := borderFill.FindElement("./hc:fillBrush/hc:winBrush")
	if winBrush == nil || winBrush.SelectAttrValue("faceColor", "") != "#D6D6D6" {
		t.Fatalf("expected merged fill color to be promoted: %v", winBrush)
	}

	anchorParagraph := anchorCell.FindElement("./hp:subList/hp:p")
	if anchorParagraph == nil {
		t.Fatalf("merged anchor paragraph missing")
	}
	anchorParaPrID := anchorParagraph.SelectAttrValue("paraPrIDRef", "")
	anchorRun := anchorParagraph.FindElement("./hp:run")
	if anchorRun == nil {
		t.Fatalf("merged anchor run missing")
	}
	anchorCharPrID := anchorRun.SelectAttrValue("charPrIDRef", "")

	runCLI(t, "split-table-cell", editableDir, "--table", "0", "--row", "0", "--col", "0", "--format", "json")

	sectionDoc = etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml after split: %v", err)
	}
	clonedCell := sectionDoc.FindElement("//hp:tbl/hp:tr[2]/hp:tc[2]")
	if clonedCell == nil {
		t.Fatalf("split cloned cell not found")
	}

	if clonedCell.SelectAttrValue("borderFillIDRef", "") != borderFillID {
		t.Fatalf("expected split cell to clone merged borderFill: %s", clonedCell.SelectAttrValue("borderFillIDRef", ""))
	}

	clonedParagraph := clonedCell.FindElement("./hp:subList/hp:p")
	if clonedParagraph == nil {
		t.Fatalf("split cloned paragraph missing")
	}
	if clonedParagraph.SelectAttrValue("paraPrIDRef", "") != anchorParaPrID {
		t.Fatalf("expected split paragraph style to match anchor: %s", clonedParagraph.SelectAttrValue("paraPrIDRef", ""))
	}
	clonedRun := clonedParagraph.FindElement("./hp:run")
	if clonedRun == nil {
		t.Fatalf("split cloned run missing")
	}
	if clonedRun.SelectAttrValue("charPrIDRef", "") != anchorCharPrID {
		t.Fatalf("expected split run style to match anchor: %s", clonedRun.SelectAttrValue("charPrIDRef", ""))
	}
}

func TestTableCellParagraphAndTextStyleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "styled.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "초기", "--format", "json")

	cellStdout := runCLI(
		t,
		"set-table-cell", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--text", "라벨\n본문 내용\n안내 문구",
		"--align", "CENTER",
		"--bold", "true",
		"--text-color", "#1F4E79",
		"--font-name", "맑은 고딕",
		"--font-size-pt", "13",
		"--format", "json",
	)

	var cellEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			ParagraphCount int      `json:"paragraphCount"`
			ParaPrIDRef    string   `json:"paraPrIdRef"`
			AppliedRuns    int      `json:"appliedRuns"`
			CharPrIDs      []string `json:"charPrIds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(cellStdout.Bytes(), &cellEnvelope); err != nil {
		t.Fatalf("decode set-table-cell response: %v", err)
	}
	if !cellEnvelope.Success || cellEnvelope.Data.ParagraphCount != 3 || cellEnvelope.Data.ParaPrIDRef == "" || cellEnvelope.Data.AppliedRuns != 3 || len(cellEnvelope.Data.CharPrIDs) != 1 {
		t.Fatalf("unexpected set-table-cell response: %s", cellStdout.String())
	}

	layoutStdout := runCLI(
		t,
		"set-table-cell-layout", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--paragraph", "1",
		"--align", "LEFT",
		"--space-after-mm", "2",
		"--line-spacing-percent", "160",
		"--format", "json",
	)

	var layoutEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			ParaPrIDRef string `json:"paraPrIdRef"`
		} `json:"data"`
	}
	if err := json.Unmarshal(layoutStdout.Bytes(), &layoutEnvelope); err != nil {
		t.Fatalf("decode set-table-cell-layout response: %v", err)
	}
	if !layoutEnvelope.Success || layoutEnvelope.Data.ParaPrIDRef == "" {
		t.Fatalf("unexpected set-table-cell-layout response: %s", layoutStdout.String())
	}

	styleStdout := runCLI(
		t,
		"set-table-cell-text-style", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--paragraph", "2",
		"--italic", "true",
		"--underline", "true",
		"--text-color", "#C00000",
		"--font-name", "맑은 고딕",
		"--font-size-pt", "11",
		"--format", "json",
	)

	var styleEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			AppliedRuns int      `json:"appliedRuns"`
			CharPrIDs   []string `json:"charPrIds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(styleStdout.Bytes(), &styleEnvelope); err != nil {
		t.Fatalf("decode set-table-cell-text-style response: %v", err)
	}
	if !styleEnvelope.Success || styleEnvelope.Data.AppliedRuns != 1 || len(styleEnvelope.Data.CharPrIDs) != 1 {
		t.Fatalf("unexpected set-table-cell-text-style response: %s", styleStdout.String())
	}

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	cellParagraphs := sectionDoc.FindElements("//hp:tbl/hp:tr/hp:tc/hp:subList/hp:p")
	if len(cellParagraphs) < 3 {
		t.Fatalf("expected cell paragraphs in section xml")
	}
	if got := cellParagraphs[0].SelectAttrValue("paraPrIDRef", ""); got != cellEnvelope.Data.ParaPrIDRef {
		t.Fatalf("unexpected first cell paraPrIDRef: %s", got)
	}
	if got := cellParagraphs[1].SelectAttrValue("paraPrIDRef", ""); got != layoutEnvelope.Data.ParaPrIDRef {
		t.Fatalf("unexpected second cell paraPrIDRef: %s", got)
	}
	noteRun := cellParagraphs[2].FindElement("./hp:run")
	if noteRun == nil {
		t.Fatalf("expected note run in third cell paragraph")
	}
	if got := noteRun.SelectAttrValue("charPrIDRef", ""); got != styleEnvelope.Data.CharPrIDs[0] {
		t.Fatalf("unexpected third cell charPrIDRef: %s", got)
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	baseParaPr := headerDoc.FindElement("//hh:paraPr[@id='" + cellEnvelope.Data.ParaPrIDRef + "']")
	if baseParaPr == nil {
		t.Fatalf("cell paraPr not found: %s", cellEnvelope.Data.ParaPrIDRef)
	}
	if align := baseParaPr.FindElement("./hh:align"); align == nil || align.SelectAttrValue("horizontal", "") != "CENTER" {
		t.Fatalf("expected center aligned cell paraPr")
	}
	bodyParaPr := headerDoc.FindElement("//hh:paraPr[@id='" + layoutEnvelope.Data.ParaPrIDRef + "']")
	if bodyParaPr == nil {
		t.Fatalf("body paraPr not found: %s", layoutEnvelope.Data.ParaPrIDRef)
	}
	if align := bodyParaPr.FindElement("./hh:align"); align == nil || align.SelectAttrValue("horizontal", "") != "LEFT" {
		t.Fatalf("expected left aligned body paraPr")
	}
	noteCharPr := headerDoc.FindElement("//hh:charPr[@id='" + styleEnvelope.Data.CharPrIDs[0] + "']")
	if noteCharPr == nil {
		t.Fatalf("note charPr not found: %s", styleEnvelope.Data.CharPrIDs[0])
	}
	if noteCharPr.SelectAttrValue("textColor", "") != "#C00000" {
		t.Fatalf("expected note text color in charPr")
	}
	if noteCharPr.SelectAttrValue("height", "") != "1100" {
		t.Fatalf("expected note font size in charPr")
	}
	if underline := noteCharPr.FindElement("./hh:underline"); underline == nil || underline.SelectAttrValue("type", "") != "BOTTOM" {
		t.Fatalf("expected underline in note charPr")
	}
	headerText, err := headerDoc.WriteToString()
	if err != nil {
		t.Fatalf("serialize header xml: %v", err)
	}
	if !strings.Contains(headerText, "face=\"맑은 고딕\"") {
		t.Fatalf("expected font face in header xml")
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{"라벨", "본문 내용", "안내 문구"} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in packed text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestFillTemplateYAMLParagraphRepeat(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for yaml fill-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "팀 소개")
	appendSectionParagraphForTest(t, sectionPath, "기존 팀 문단 1")
	appendSectionParagraphForTest(t, sectionPath, "기존 팀 문단 2")
	appendSectionParagraphForTest(t, sectionPath, "기존 팀 문단 3")

	mappingPath := filepath.Join(t.TempDir(), "mapping.yaml")
	mapping := `replacements:
  - nearText: "팀 소개"
    values:
      - "팀 문단 1"
      - "팀 문단 2"
    mode: "paragraph-next-repeat"
`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write yaml mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Changes []struct {
				Mode string `json:"mode"`
				Text string `json:"text"`
			} `json:"changes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode yaml fill-template dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 2 {
		t.Fatalf("expected two yaml paragraph repeat changes: %s", dryRunStdout.String())
	}
	if dryRunEnvelope.Data.Changes[0].Mode != "paragraph-next-repeat" || dryRunEnvelope.Data.Changes[1].Mode != "paragraph-next-repeat" {
		t.Fatalf("expected paragraph-next-repeat mode for yaml mapping: %+v", dryRunEnvelope.Data.Changes)
	}

	runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after yaml fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"팀 문단 1", "팀 문단 2", "기존 팀 문단 3"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after yaml fill-template apply: %q", needle, sectionText)
		}
	}
}

func TestFillTemplateRejectsInvalidMapping(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for invalid fill-template mapping: %v stderr=%s", err, stderr.String())
	}

	mappingPath := filepath.Join(t.TempDir(), "invalid-mapping.json")
	mapping := `{
  "replacements": [
    {"anchor": "과제명", "value": "프로젝트 X", "values": ["중복"], "mode": "table-right"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write invalid mapping file: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	err := Run([]string{"fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("fill-template should fail for invalid mapping")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode invalid fill-template response: %v", decodeErr)
	}
	if envelope.Success || envelope.Error.Code != "invalid_arguments" || !strings.Contains(envelope.Error.Message, "exactly one of value, values, grid, or records") {
		t.Fatalf("unexpected invalid fill-template response: %s", stdout.String())
	}
}

func TestFillTemplateReportsMisses(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for missed fill-template mapping: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "제한 참여기관 표")
	runCLI(t, "add-table", outputDir, "--cells", "참여기관;기존 참여기관 1", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "miss-mapping.json")
	mapping := `{
  "replacements": [
    {"anchor": "참여기관", "tableLabel": "제한 참여기관 표", "values": ["기관1", "기관2"], "mode": "table-down-repeat"},
    {"anchor": "없는 앵커", "tableLabel": "없는 표", "value": "실패"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write miss mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count     int `json:"count"`
			MissCount int `json:"missCount"`
			Misses    []struct {
				Selector  string `json:"selector"`
				Reason    string `json:"reason"`
				Requested int    `json:"requested"`
				Matched   int    `json:"matched"`
				Partial   bool   `json:"partial"`
			} `json:"misses"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode missed fill-template dry-run response: %v", err)
	}
	if !envelope.Success || envelope.Data.Count != 1 || envelope.Data.MissCount != 2 {
		t.Fatalf("expected one change and two misses: %s", dryRunStdout.String())
	}
	if envelope.Data.Misses[0].Selector != "참여기관" || envelope.Data.Misses[0].Reason != "insufficient-target-capacity" || !envelope.Data.Misses[0].Partial {
		t.Fatalf("expected partial miss for limited repeat capacity: %+v", envelope.Data.Misses)
	}
	if envelope.Data.Misses[1].Selector != "없는 앵커" || envelope.Data.Misses[1].Reason != "table-label-not-found" {
		t.Fatalf("expected table-label miss for missing anchor scope: %+v", envelope.Data.Misses)
	}
}

func TestFillTemplateReportsRequiredMisses(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for required miss fill-template: %v stderr=%s", err, stderr.String())
	}

	mappingPath := filepath.Join(t.TempDir(), "required-miss.json")
	mapping := `{
  "replacements": [
    {"anchor": "없는 앵커", "tableLabel": "없는 표", "value": "실패", "required": true}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write required miss mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			MissCount int `json:"missCount"`
			Misses    []struct {
				Required bool   `json:"required"`
				Reason   string `json:"reason"`
			} `json:"misses"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode required miss response: %v", err)
	}
	if !envelope.Success || envelope.Data.MissCount != 1 || len(envelope.Data.Misses) != 1 {
		t.Fatalf("expected one required miss: %s", dryRunStdout.String())
	}
	if !envelope.Data.Misses[0].Required || envelope.Data.Misses[0].Reason != "required-target-not-found" {
		t.Fatalf("expected required-target-not-found miss payload: %+v", envelope.Data.Misses)
	}
}

func TestFillTemplateFailOnMiss(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for fail-on-miss fill-template: %v stderr=%s", err, stderr.String())
	}

	mappingPath := filepath.Join(t.TempDir(), "fail-on-miss.json")
	mapping := `{
  "replacements": [
    {"anchor": "없는 앵커", "tableLabel": "없는 표", "value": "실패"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write fail-on-miss mapping file: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	err := Run([]string{"fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--fail-on-miss", "true", "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("fill-template should fail when miss exists and fail-on-miss is true")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code string `json:"code"`
		} `json:"error"`
		Data struct {
			FailOnMiss bool `json:"failOnMiss"`
			Applied    bool `json:"applied"`
			MissCount  int  `json:"missCount"`
			Misses     []struct {
				Reason string `json:"reason"`
			} `json:"misses"`
		} `json:"data"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode fail-on-miss response: %v", decodeErr)
	}
	if envelope.Success || envelope.Error.Code != "fill_template_miss" {
		t.Fatalf("unexpected fail-on-miss response: %s", stdout.String())
	}
	if !envelope.Data.FailOnMiss || envelope.Data.Applied || envelope.Data.MissCount != 1 || len(envelope.Data.Misses) != 1 {
		t.Fatalf("expected fail-on-miss data in response: %+v", envelope.Data)
	}
	if envelope.Data.Misses[0].Reason != "table-label-not-found" {
		t.Fatalf("expected specific miss reason in fail-on-miss response: %+v", envelope.Data.Misses)
	}
}

func TestFillTemplateUniqueRejectsAmbiguousMatches(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "표 1", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,값1", "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "표 2", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,값2", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "unique-ambiguous.json")
	mapping := `{
  "replacements": [
    {"anchor": "항목", "value": "최종값", "mode": "table-right", "unique": true}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write unique ambiguous mapping file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("fill-template should fail for ambiguous unique selector")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode unique ambiguous response: %v", decodeErr)
	}
	if envelope.Success || !strings.Contains(envelope.Error.Message, "ambiguous selector") {
		t.Fatalf("unexpected unique ambiguous response: %s", stdout.String())
	}
}

func TestFillTemplateTargetsNestedTableByIndex(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "중첩 표 테스트", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,외부값;기타,Z", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "항목,내부값", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "nested-target.json")
	mapping := `{
  "replacements": [
    {"anchor": "항목", "tableIndex": 1, "value": "내부 최종값", "mode": "table-right"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write nested target mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Changes []struct {
				TableIndex *int   `json:"tableIndex"`
				Text       string `json:"text"`
			} `json:"changes"`
			MissCount int `json:"missCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode nested target dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 1 || dryRunEnvelope.Data.MissCount != 0 {
		t.Fatalf("unexpected nested target dry-run response: %s", dryRunStdout.String())
	}
	if dryRunEnvelope.Data.Changes[0].TableIndex == nil || *dryRunEnvelope.Data.Changes[0].TableIndex != 1 || dryRunEnvelope.Data.Changes[0].Text != "내부 최종값" {
		t.Fatalf("expected nested table index targeting: %+v", dryRunEnvelope.Data.Changes)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section after nested target apply: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "내부 최종값") || !strings.Contains(sectionText, "외부값") {
		t.Fatalf("expected nested target apply to preserve outer value and update inner value: %q", sectionText)
	}
}

func TestFillTemplateOccurrenceScopesDuplicateMatches(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for occurrence fill-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "요약")
	appendSectionParagraphForTest(t, sectionPath, "첫 번째 요약 본문")
	appendSectionParagraphForTest(t, sectionPath, "요약")
	appendSectionParagraphForTest(t, sectionPath, "두 번째 요약 본문")
	appendSectionParagraphForTest(t, sectionPath, "표 1")
	runCLI(t, "add-table", outputDir, "--cells", "항목,외부값", "--format", "json")
	appendSectionParagraphForTest(t, sectionPath, "표 2")
	runCLI(t, "add-table", outputDir, "--cells", "항목,내부값", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "occurrence.json")
	mapping := `{
  "replacements": [
    {"nearText": "요약", "occurrence": 2, "value": "두 번째 요약 최종본", "mode": "paragraph-next"},
    {"anchor": "항목", "occurrence": 2, "value": "두 번째 표 값", "mode": "table-right"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write occurrence mapping file: %v", err)
	}

	runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after occurrence fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "첫 번째 요약 본문") || !strings.Contains(sectionText, "두 번째 요약 최종본") {
		t.Fatalf("expected occurrence to target only second near-text block: %q", sectionText)
	}
	if !strings.Contains(sectionText, "외부값") || !strings.Contains(sectionText, "두 번째 표 값") {
		t.Fatalf("expected occurrence to target only second anchor table: %q", sectionText)
	}
}

func TestFillTemplateReportsMissingOccurrence(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for occurrence miss fill-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "요약")
	appendSectionParagraphForTest(t, sectionPath, "유일한 요약 본문")

	mappingPath := filepath.Join(t.TempDir(), "occurrence-miss.json")
	mapping := `{
  "replacements": [
    {"nearText": "요약", "occurrence": 3, "value": "도달 불가", "mode": "paragraph-next"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write occurrence miss mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count     int `json:"count"`
			MissCount int `json:"missCount"`
			Misses    []struct {
				Reason     string `json:"reason"`
				Occurrence *int   `json:"occurrence"`
			} `json:"misses"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode occurrence miss response: %v", err)
	}
	if !envelope.Success || envelope.Data.Count != 0 || envelope.Data.MissCount != 1 {
		t.Fatalf("expected one occurrence miss: %s", dryRunStdout.String())
	}
	if envelope.Data.Misses[0].Reason != "occurrence-not-found" || envelope.Data.Misses[0].Occurrence == nil || *envelope.Data.Misses[0].Occurrence != 3 {
		t.Fatalf("expected occurrence-not-found miss payload: %+v", envelope.Data.Misses)
	}
}

func TestFillTemplatePreservesTableCellStyle(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "스타일 보존 표", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,기존 값", "--format", "json")

	styleStdout := runCLI(
		t,
		"set-table-cell-text-style", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "1",
		"--paragraph", "0",
		"--bold", "true",
		"--text-color", "#C00000",
		"--font-name", "맑은 고딕",
		"--font-size-pt", "11",
		"--format", "json",
	)

	var styleEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			CharPrIDs []string `json:"charPrIds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(styleStdout.Bytes(), &styleEnvelope); err != nil {
		t.Fatalf("decode fill-template preserve style response: %v", err)
	}
	if !styleEnvelope.Success || len(styleEnvelope.Data.CharPrIDs) != 1 {
		t.Fatalf("unexpected set-table-cell-text-style response: %s", styleStdout.String())
	}

	beforeDoc := etree.NewDocument()
	if err := beforeDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section before preserve style fill-template: %v", err)
	}
	beforeParagraph := beforeDoc.FindElement("//hp:tbl/hp:tr/hp:tc[2]/hp:subList/hp:p")
	if beforeParagraph == nil {
		t.Fatalf("expected target paragraph before fill-template apply")
	}
	beforeLineSegAttrs := map[string]string{
		"textpos":    "0",
		"vertpos":    "0",
		"vertsize":   "1100",
		"textheight": "1100",
		"baseline":   "935",
		"spacing":    "550",
		"horzpos":    "0",
		"horzsize":   "42520",
		"flags":      "393216",
	}
	ensureLineSegAttrsForTest(beforeParagraph, beforeLineSegAttrs)
	if err := beforeDoc.WriteToFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("write section before preserve style fill-template: %v", err)
	}

	mappingPath := filepath.Join(t.TempDir(), "preserve-style.json")
	mapping := `{
  "replacements": [
    {"anchor": "항목", "value": "최종 값", "mode": "table-right"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write preserve style mapping file: %v", err)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section after preserve style fill-template: %v", err)
	}
	targetRun := sectionDoc.FindElement("//hp:tbl/hp:tr/hp:tc[2]/hp:subList/hp:p/hp:run")
	if targetRun == nil {
		t.Fatalf("expected target run after fill-template apply")
	}
	if got := targetRun.SelectAttrValue("charPrIDRef", ""); got != styleEnvelope.Data.CharPrIDs[0] {
		t.Fatalf("expected fill-template to preserve cell charPrIDRef: %s", got)
	}
	afterLineSegAttrs := firstLineSegAttrsForTest(targetRun.Parent())
	if afterLineSegAttrs["vertsize"] != beforeLineSegAttrs["vertsize"] || afterLineSegAttrs["spacing"] != beforeLineSegAttrs["spacing"] {
		t.Fatalf("expected fill-template to preserve cell lineseg attrs: before=%v after=%v", beforeLineSegAttrs, afterLineSegAttrs)
	}
}

func TestFillTemplatePreservesParagraphStyle(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "요약\n기존 본문", "--format", "json")

	layoutStdout := runCLI(
		t,
		"set-paragraph-layout",
		editableDir,
		"--paragraph", "1",
		"--align", "center",
		"--left-margin-mm", "6",
		"--right-margin-mm", "4",
		"--format", "json",
	)

	var layoutEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			ParaPrIDRef string `json:"paraPrIdRef"`
		} `json:"data"`
	}
	if err := json.Unmarshal(layoutStdout.Bytes(), &layoutEnvelope); err != nil {
		t.Fatalf("decode fill-template preserve paragraph layout response: %v", err)
	}
	if !layoutEnvelope.Success || layoutEnvelope.Data.ParaPrIDRef == "" {
		t.Fatalf("unexpected set-paragraph-layout response: %s", layoutStdout.String())
	}

	styleStdout := runCLI(
		t,
		"set-text-style",
		editableDir,
		"--paragraph", "1",
		"--bold", "true",
		"--underline", "true",
		"--text-color", "#C00000",
		"--format", "json",
	)

	var styleEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			CharPrIDs []string `json:"charPrIds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(styleStdout.Bytes(), &styleEnvelope); err != nil {
		t.Fatalf("decode fill-template preserve paragraph style response: %v", err)
	}
	if !styleEnvelope.Success || len(styleEnvelope.Data.CharPrIDs) != 1 {
		t.Fatalf("unexpected set-text-style response: %s", styleStdout.String())
	}

	beforeDoc := etree.NewDocument()
	if err := beforeDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section before preserve paragraph style fill-template: %v", err)
	}
	beforeParagraphs := beforeDoc.FindElements("//hp:p")
	if len(beforeParagraphs) < 3 {
		t.Fatalf("expected editable paragraphs before fill-template apply")
	}
	beforeLineSegAttrs := map[string]string{
		"textpos":    "0",
		"vertpos":    "0",
		"vertsize":   "1100",
		"textheight": "1100",
		"baseline":   "935",
		"spacing":    "550",
		"horzpos":    "0",
		"horzsize":   "42520",
		"flags":      "393216",
	}
	ensureLineSegAttrsForTest(beforeParagraphs[2], beforeLineSegAttrs)
	if err := beforeDoc.WriteToFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("write section before preserve paragraph style fill-template: %v", err)
	}

	mappingPath := filepath.Join(t.TempDir(), "preserve-paragraph-style.json")
	mapping := `{
  "replacements": [
    {"nearText": "요약", "value": "최종 요약 본문", "mode": "paragraph-next"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write preserve paragraph style mapping file: %v", err)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section after preserve paragraph style fill-template: %v", err)
	}
	paragraphs := sectionDoc.FindElements("//hp:p")
	if len(paragraphs) < 3 {
		t.Fatalf("expected editable paragraphs after fill-template apply")
	}
	targetParagraph := paragraphs[2]
	targetRun := targetParagraph.FindElement("./hp:run")
	if targetRun == nil {
		t.Fatalf("expected target run after paragraph fill-template apply")
	}
	if got := targetParagraph.SelectAttrValue("paraPrIDRef", ""); got != layoutEnvelope.Data.ParaPrIDRef {
		t.Fatalf("expected fill-template to preserve paragraph paraPrIDRef: %s", got)
	}
	if got := targetRun.SelectAttrValue("charPrIDRef", ""); got != styleEnvelope.Data.CharPrIDs[0] {
		t.Fatalf("expected fill-template to preserve paragraph charPrIDRef: %s", got)
	}
	targetText := targetParagraph.FindElement("./hp:run/hp:t")
	if targetText == nil {
		t.Fatalf("expected paragraph text node after fill-template apply")
	}
	if got := strings.TrimSpace(targetText.Text()); got != "최종 요약 본문" {
		t.Fatalf("expected fill-template to update paragraph text: %q", got)
	}
	afterLineSegAttrs := firstLineSegAttrsForTest(targetParagraph)
	if afterLineSegAttrs["vertsize"] != beforeLineSegAttrs["vertsize"] || afterLineSegAttrs["spacing"] != beforeLineSegAttrs["spacing"] {
		t.Fatalf("expected fill-template to preserve paragraph lineseg attrs: before=%v after=%v", beforeLineSegAttrs, afterLineSegAttrs)
	}
}

func TestFillTemplateSupportsLeftAndUpModes(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for directional fill-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "왼쪽 표")
	runCLI(t, "add-table", outputDir, "--cells", "기존 왼쪽,라벨", "--format", "json")
	appendSectionParagraphForTest(t, sectionPath, "위쪽 표")
	runCLI(t, "add-table", outputDir, "--cells", "기존 위;라벨", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "directional.json")
	mapping := `{
  "replacements": [
    {"anchor": "라벨", "tableLabel": "왼쪽 표", "value": "왼쪽 최종값", "mode": "table-left"},
    {"anchor": "라벨", "tableLabel": "위쪽 표", "value": "위쪽 최종값", "mode": "table-up"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write directional mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Changes []struct {
				Mode       string `json:"mode"`
				TableLabel string `json:"tableLabel"`
				Text       string `json:"text"`
			} `json:"changes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode directional fill-template dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 2 {
		t.Fatalf("expected two directional changes: %s", dryRunStdout.String())
	}

	runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after directional fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"왼쪽 최종값", "위쪽 최종값"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after directional fill-template apply: %q", needle, sectionText)
		}
	}
}

func TestFillTemplateSupportsExactMatchMode(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for exact match fill-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "요약서")
	appendSectionParagraphForTest(t, sectionPath, "요약서 본문")
	appendSectionParagraphForTest(t, sectionPath, "요약")
	appendSectionParagraphForTest(t, sectionPath, "요약 본문")
	appendSectionParagraphForTest(t, sectionPath, "정확 표")
	runCLI(t, "add-table", outputDir, "--cells", "세부항목,건드리면 안 됨", "--format", "json")
	runCLI(t, "add-table", outputDir, "--cells", "항목,정확히 변경", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "exact-match.json")
	mapping := `{
  "replacements": [
    {"nearText": "요약", "value": "정확 요약 본문", "mode": "paragraph-next", "matchMode": "exact"},
    {"anchor": "항목", "value": "정확 표 값", "mode": "table-right", "matchMode": "exact"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write exact match mapping file: %v", err)
	}

	runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after exact match fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "요약서 본문") || !strings.Contains(sectionText, "정확 요약 본문") {
		t.Fatalf("expected exact near-text matching to preserve 요약서 block: %q", sectionText)
	}
	if !strings.Contains(sectionText, "건드리면 안 됨") || !strings.Contains(sectionText, "정확 표 값") {
		t.Fatalf("expected exact anchor matching to preserve 세부항목 table: %q", sectionText)
	}
}

func TestFillTemplateSupportsGridMode(t *testing.T) {
	archivePath := fixtureArchive(t)
	outputDir := filepath.Join(t.TempDir(), "unpacked")

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run([]string{"unpack", archivePath, "--output", outputDir, "--format", "json"}, stdout, stderr); err != nil {
		t.Fatalf("unpack for grid fill-template: %v stderr=%s", err, stderr.String())
	}

	sectionPath := filepath.Join(outputDir, "Contents", "section0.xml")
	appendSectionParagraphForTest(t, sectionPath, "참여인력 블록")
	runCLI(t, "add-table", outputDir, "--cells", "참여인력,기존 이름1,기존 소속1;비고,기존 이름2,기존 소속2", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "grid.json")
	mapping := `{
  "replacements": [
    {
      "anchor": "참여인력",
      "tableLabel": "참여인력 블록",
      "grid": [
        ["홍길동", "기관 A"],
        ["김영희", "기관 B"]
      ],
      "mode": "table-right-grid"
    }
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write grid mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Changes []struct {
				Mode string `json:"mode"`
				Text string `json:"text"`
			} `json:"changes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode grid fill-template dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 4 {
		t.Fatalf("expected four grid changes: %s", dryRunStdout.String())
	}

	runCLI(t, "fill-template", outputDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(sectionPath)
	if err != nil {
		t.Fatalf("read section after grid fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"홍길동", "기관 A", "김영희", "기관 B"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after grid fill-template apply: %q", needle, sectionText)
		}
	}
}

func TestFillTemplateSupportsDownRepeatExpand(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "참여기관 확장 표", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "참여기관,비고;기존 기관1,기존 비고1", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "down-repeat-expand.json")
	mapping := `{
  "replacements": [
    {"anchor": "참여기관", "tableLabel": "참여기관 확장 표", "values": ["기관1", "기관2", "기관3"], "mode": "table-down-repeat", "expand": true}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write down repeat expand mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count     int `json:"count"`
			MissCount int `json:"missCount"`
			Changes   []struct {
				Mode   string `json:"mode"`
				Expand bool   `json:"expand"`
				Cell   struct {
					Row int `json:"row"`
					Col int `json:"col"`
				} `json:"cell"`
				Text string `json:"text"`
			} `json:"changes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode down repeat expand dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 3 || dryRunEnvelope.Data.MissCount != 0 {
		t.Fatalf("expected three expanded repeat changes without misses: %s", dryRunStdout.String())
	}
	if !dryRunEnvelope.Data.Changes[1].Expand || dryRunEnvelope.Data.Changes[1].Cell.Row != 2 {
		t.Fatalf("expected second expanded change to target appended row: %+v", dryRunEnvelope.Data.Changes)
	}
	if !dryRunEnvelope.Data.Changes[2].Expand || dryRunEnvelope.Data.Changes[2].Cell.Row != 3 {
		t.Fatalf("expected third expanded change to target appended row: %+v", dryRunEnvelope.Data.Changes)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section after down repeat expand fill-template apply: %v", err)
	}
	sectionText, err := sectionDoc.WriteToString()
	if err != nil {
		t.Fatalf("serialize section after down repeat expand fill-template apply: %v", err)
	}
	for _, needle := range []string{"기관1", "기관2", "기관3"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after down repeat expand apply: %q", needle, sectionText)
		}
	}
	if strings.Count(sectionText, "<hp:tr") < 4 {
		t.Fatalf("expected appended table rows after down repeat expand apply: %q", sectionText)
	}
}

func TestFillTemplateSupportsDownGridExpand(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "참여인력 확장 표", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "참여인력,소속,비고;기존 이름1,기존 소속1,기존 비고1", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "down-grid-expand.json")
	mapping := `{
  "replacements": [
    {
      "anchor": "참여인력",
      "tableLabel": "참여인력 확장 표",
      "grid": [
        ["홍길동", "기관 A"],
        ["김영희", "기관 B"],
        ["박철수", "기관 C"]
      ],
      "mode": "table-down-grid",
      "expand": true
    }
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write down grid expand mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count     int `json:"count"`
			MissCount int `json:"missCount"`
			Changes   []struct {
				Mode   string `json:"mode"`
				Expand bool   `json:"expand"`
				Cell   struct {
					Row int `json:"row"`
					Col int `json:"col"`
				} `json:"cell"`
				Text string `json:"text"`
			} `json:"changes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode down grid expand dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 6 || dryRunEnvelope.Data.MissCount != 0 {
		t.Fatalf("expected six expanded grid changes without misses: %s", dryRunStdout.String())
	}
	if !dryRunEnvelope.Data.Changes[2].Expand || dryRunEnvelope.Data.Changes[2].Cell.Row != 2 {
		t.Fatalf("expected third expanded grid change to target appended row: %+v", dryRunEnvelope.Data.Changes)
	}
	if !dryRunEnvelope.Data.Changes[4].Expand || dryRunEnvelope.Data.Changes[4].Cell.Row != 3 {
		t.Fatalf("expected fifth expanded grid change to target appended row: %+v", dryRunEnvelope.Data.Changes)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section after down grid expand fill-template apply: %v", err)
	}
	sectionText, err := sectionDoc.WriteToString()
	if err != nil {
		t.Fatalf("serialize section after down grid expand fill-template apply: %v", err)
	}
	for _, needle := range []string{"홍길동", "기관 A", "김영희", "기관 B", "박철수", "기관 C"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after down grid expand apply: %q", needle, sectionText)
		}
	}
	if strings.Count(sectionText, "<hp:tr") < 4 {
		t.Fatalf("expected appended table rows after down grid expand apply: %q", sectionText)
	}
}

func TestFillTemplatePreservesExpandedRowStyle(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "확장 스타일 표", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "참여인력,소속;기존 이름1,기존 소속1", "--format", "json")

	layoutStdout := runCLI(
		t,
		"set-table-cell-layout", editableDir,
		"--table", "0",
		"--row", "1",
		"--col", "0",
		"--paragraph", "0",
		"--align", "CENTER",
		"--space-after-mm", "2",
		"--format", "json",
	)
	var layoutEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			ParaPrIDRef string `json:"paraPrIdRef"`
		} `json:"data"`
	}
	if err := json.Unmarshal(layoutStdout.Bytes(), &layoutEnvelope); err != nil {
		t.Fatalf("decode expanded row layout response: %v", err)
	}
	if !layoutEnvelope.Success || layoutEnvelope.Data.ParaPrIDRef == "" {
		t.Fatalf("unexpected set-table-cell-layout response: %s", layoutStdout.String())
	}

	styleStdout := runCLI(
		t,
		"set-table-cell-text-style", editableDir,
		"--table", "0",
		"--row", "1",
		"--col", "1",
		"--paragraph", "0",
		"--bold", "true",
		"--text-color", "#C00000",
		"--font-name", "맑은 고딕",
		"--font-size-pt", "11",
		"--format", "json",
	)
	var styleEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			CharPrIDs []string `json:"charPrIds"`
		} `json:"data"`
	}
	if err := json.Unmarshal(styleStdout.Bytes(), &styleEnvelope); err != nil {
		t.Fatalf("decode expanded row style response: %v", err)
	}
	if !styleEnvelope.Success || len(styleEnvelope.Data.CharPrIDs) != 1 {
		t.Fatalf("unexpected set-table-cell-text-style response: %s", styleStdout.String())
	}

	mappingPath := filepath.Join(t.TempDir(), "expanded-row-style.json")
	mapping := `{
  "replacements": [
    {
      "anchor": "참여인력",
      "tableLabel": "확장 스타일 표",
      "grid": [
        ["홍길동", "기관 A"],
        ["김영희", "기관 B"],
        ["박철수", "기관 C"]
      ],
      "mode": "table-down-grid",
      "expand": true
    }
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write expanded row style mapping file: %v", err)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section after expanded row style fill-template apply: %v", err)
	}

	var row3Col0Paragraph *etree.Element
	var row3Col1Run *etree.Element
	for _, cell := range sectionDoc.FindElements("//hp:tbl/hp:tr/hp:tc") {
		addr := cell.FindElement("./hp:cellAddr")
		if addr == nil {
			continue
		}
		row := addr.SelectAttrValue("rowAddr", "")
		col := addr.SelectAttrValue("colAddr", "")
		switch {
		case row == "3" && col == "0":
			row3Col0Paragraph = cell.FindElement("./hp:subList/hp:p")
		case row == "3" && col == "1":
			row3Col1Run = cell.FindElement("./hp:subList/hp:p/hp:run")
		}
	}
	if row3Col0Paragraph == nil || row3Col1Run == nil {
		t.Fatalf("expected expanded row cells after fill-template apply")
	}
	if got := row3Col0Paragraph.SelectAttrValue("paraPrIDRef", ""); got != layoutEnvelope.Data.ParaPrIDRef {
		t.Fatalf("expected expanded row to preserve paraPrIDRef: %s", got)
	}
	if got := row3Col1Run.SelectAttrValue("charPrIDRef", ""); got != styleEnvelope.Data.CharPrIDs[0] {
		t.Fatalf("expected expanded row to preserve charPrIDRef: %s", got)
	}
}

func TestFillTemplateSupportsRecordMode(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "참여인력 레코드 표", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "참여인력,소속,역할;기존 이름1,기존 소속1,기존 역할1", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "records.json")
	mapping := `{
  "replacements": [
    {
      "anchor": "참여인력",
      "tableLabel": "참여인력 레코드 표",
      "fields": ["name", "org", "role"],
      "records": [
        {"name": "홍길동", "org": "기관 A", "role": "PM"},
        {"name": "김영희", "org": "기관 B", "role": "연구원"},
        {"name": "박철수", "org": "기관 C", "role": "개발"}
      ],
      "mode": "table-down-records",
      "expand": true
    }
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write records mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count     int `json:"count"`
			MissCount int `json:"missCount"`
			Changes   []struct {
				Mode string `json:"mode"`
				Text string `json:"text"`
				Cell struct {
					Row int `json:"row"`
					Col int `json:"col"`
				} `json:"cell"`
			} `json:"changes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode records dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 9 || dryRunEnvelope.Data.MissCount != 0 {
		t.Fatalf("expected nine record changes without misses: %s", dryRunStdout.String())
	}
	if dryRunEnvelope.Data.Changes[0].Mode != "table-down-records" || dryRunEnvelope.Data.Changes[3].Cell.Row != 2 || dryRunEnvelope.Data.Changes[6].Cell.Row != 3 {
		t.Fatalf("expected record mode changes to expand by row: %+v", dryRunEnvelope.Data.Changes)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section after records fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{"홍길동", "기관 A", "PM", "김영희", "기관 B", "연구원", "박철수", "기관 C", "개발"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml after record mode apply: %q", needle, sectionText)
		}
	}
}

func TestFillTemplateSupportsRecordFallbackValue(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "참여인력 fallback 표", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "참여인력,소속,역할;기존 이름1,기존 소속1,기존 역할1", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "records-fallback.json")
	mapping := `{
  "replacements": [
    {
      "anchor": "참여인력",
      "tableLabel": "참여인력 fallback 표",
      "fields": ["name", "org", "role"],
      "records": [
        {"name": "홍길동", "org": "기관 A", "role": "PM"},
        {"name": "김영희", "org": "기관 B"}
      ],
      "fallbackValue": "미정",
      "mode": "table-down-records",
      "expand": true
    }
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write records fallback mapping file: %v", err)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section after records fallback fill-template apply: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "김영희") || !strings.Contains(sectionText, "기관 B") || !strings.Contains(sectionText, "미정") {
		t.Fatalf("expected fallback value in section xml after record mode apply: %q", sectionText)
	}
}

func TestFillTemplateSkipsNestedTablesWithoutExplicitTableIndex(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "항목,외부값;보조,중첩 셀", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "항목,내부값", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "skip-nested.json")
	mapping := `{
  "replacements": [
    {"anchor": "항목", "value": "외부 최종값", "mode": "table-right"}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write skip nested mapping file: %v", err)
	}

	dryRunStdout := runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "true", "--format", "json")
	var dryRunEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Count   int `json:"count"`
			Changes []struct {
				TableIndex *int   `json:"tableIndex"`
				Text       string `json:"text"`
			} `json:"changes"`
			MissCount int `json:"missCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dryRunStdout.Bytes(), &dryRunEnvelope); err != nil {
		t.Fatalf("decode skip nested dry-run response: %v", err)
	}
	if !dryRunEnvelope.Success || dryRunEnvelope.Data.Count != 1 || dryRunEnvelope.Data.MissCount != 0 {
		t.Fatalf("unexpected skip nested dry-run response: %s", dryRunStdout.String())
	}
	if dryRunEnvelope.Data.Changes[0].TableIndex == nil || *dryRunEnvelope.Data.Changes[0].TableIndex != 0 {
		t.Fatalf("expected default fill-template selection to stay on outer table: %+v", dryRunEnvelope.Data.Changes)
	}

	runCLI(t, "fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section after skip nested apply: %v", err)
	}
	sectionText := string(sectionBytes)
	if !strings.Contains(sectionText, "외부 최종값") {
		t.Fatalf("expected outer table to be updated: %q", sectionText)
	}
	if !strings.Contains(sectionText, "내부값") {
		t.Fatalf("expected nested table to remain untouched without explicit tableIndex: %q", sectionText)
	}
}

func TestFillTemplateExpandFailsForMergedTemplateRow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "병합 확장 표", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "참여기관,비고;기존 기관1,기존 비고1", "--format", "json")
	runCLI(t, "merge-table-cells", editableDir, "--table", "0", "--start-row", "1", "--start-col", "0", "--end-row", "1", "--end-col", "1", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "merged-expand-error.json")
	mapping := `{
  "replacements": [
    {"anchor": "참여기관", "tableLabel": "병합 확장 표", "values": ["기관1", "기관2"], "mode": "table-down-repeat", "expand": true}
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write merged expand error mapping file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("fill-template should fail for merged template row expansion")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode merged expand error response: %v", decodeErr)
	}
	if envelope.Success || envelope.Error.Code == "" || !strings.Contains(envelope.Error.Message, "merged template rows") {
		t.Fatalf("unexpected merged expand error response: %s", stdout.String())
	}
}

func TestFillTemplateExpandFailsForNestedTemplateRow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "중첩 확장 표", "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "참여인력,소속;기존 이름1,중첩 셀", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2", "--format", "json")

	mappingPath := filepath.Join(t.TempDir(), "nested-expand-error.json")
	mapping := `{
  "replacements": [
    {
      "anchor": "참여인력",
      "tableLabel": "중첩 확장 표",
      "grid": [
        ["홍길동", "기관 A"],
        ["김영희", "기관 B"]
      ],
      "mode": "table-down-grid",
      "expand": true
    }
  ]
}`
	if err := os.WriteFile(mappingPath, []byte(mapping), 0o644); err != nil {
		t.Fatalf("write nested expand error mapping file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := Run([]string{"fill-template", editableDir, "--mapping", mappingPath, "--dry-run", "false", "--format", "json"}, stdout, stderr)
	if err == nil {
		t.Fatal("fill-template should fail for nested template row expansion")
	}

	var envelope struct {
		Success bool `json:"success"`
		Error   struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if decodeErr := json.Unmarshal(stdout.Bytes(), &envelope); decodeErr != nil {
		t.Fatalf("decode nested expand error response: %v", decodeErr)
	}
	if envelope.Success || envelope.Error.Code == "" || !strings.Contains(envelope.Error.Message, "nested tables") {
		t.Fatalf("unexpected nested expand error response: %s", stdout.String())
	}
}

func TestNestedTableWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A,B;C,D", "--format", "json")
	runCLI(t, "add-nested-table", editableDir, "--table", "0", "--row", "1", "--col", "1", "--cells", "내부1,내부2;내부3,내부4", "--format", "json")

	analyzeStdout := runCLI(t, "analyze-template", editableDir, "--format", "json")
	var analyzeEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Analysis struct {
				TableCount int `json:"tableCount"`
				Tables     []struct {
					TableIndex       int  `json:"tableIndex"`
					ParentTableIndex *int `json:"parentTableIndex"`
					NestedDepth      int  `json:"nestedDepth"`
				} `json:"tables"`
			} `json:"analysis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(analyzeStdout.Bytes(), &analyzeEnvelope); err != nil {
		t.Fatalf("decode analyze-template nested table response: %v", err)
	}
	if !analyzeEnvelope.Success || analyzeEnvelope.Data.Analysis.TableCount != 2 {
		t.Fatalf("unexpected analyze-template nested table response: %s", analyzeStdout.String())
	}
	if analyzeEnvelope.Data.Analysis.Tables[0].ParentTableIndex != nil || analyzeEnvelope.Data.Analysis.Tables[0].NestedDepth != 0 {
		t.Fatalf("expected outer table hierarchy metadata: %+v", analyzeEnvelope.Data.Analysis.Tables)
	}
	if analyzeEnvelope.Data.Analysis.Tables[1].ParentTableIndex == nil || *analyzeEnvelope.Data.Analysis.Tables[1].ParentTableIndex != 0 || analyzeEnvelope.Data.Analysis.Tables[1].NestedDepth != 1 {
		t.Fatalf("expected nested table hierarchy metadata: %+v", analyzeEnvelope.Data.Analysis.Tables)
	}

	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	if strings.Count(sectionText, "<hp:tbl") != 2 {
		t.Fatalf("expected outer and nested table in section xml: %s", sectionText)
	}
	for _, needle := range []string{"내부1", "내부4"} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{"A", "D", "내부1", "내부4"} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestAddNestedTableWithGeometryOptions(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-table", editableDir, "--cells", "A", "--format", "json")
	runCLI(
		t,
		"add-nested-table", editableDir,
		"--table", "0",
		"--row", "0",
		"--col", "0",
		"--rows", "2",
		"--cols", "2",
		"--col-widths-mm", "15,25",
		"--row-heights-mm", "6,8",
		"--margin-left-mm", "1",
		"--margin-top-mm", "2",
		"--format", "json",
	)

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}

	tables := sectionDoc.FindElements(".//hp:tbl")
	if len(tables) != 2 {
		t.Fatalf("expected outer and nested table, got %d", len(tables))
	}

	nested := tables[1]
	size := nested.FindElement("./hp:sz")
	if size == nil {
		t.Fatal("expected nested table size")
	}
	if got := size.SelectAttrValue("width", ""); got != "11339" {
		t.Fatalf("unexpected nested table width: %s", got)
	}
	if got := size.SelectAttrValue("height", ""); got != "3969" {
		t.Fatalf("unexpected nested table height: %s", got)
	}

	outMargin := nested.FindElement("./hp:outMargin")
	if outMargin == nil {
		t.Fatal("expected nested outMargin")
	}
	if got := outMargin.SelectAttrValue("left", ""); got != "283" {
		t.Fatalf("unexpected nested left margin: %s", got)
	}
	if got := outMargin.SelectAttrValue("top", ""); got != "567" {
		t.Fatalf("unexpected nested top margin: %s", got)
	}

	rows := nested.FindElements("./hp:tr")
	if len(rows) != 2 {
		t.Fatalf("expected 2 nested rows, got %d", len(rows))
	}
	firstCell := rows[0].FindElement("./hp:tc/hp:cellSz")
	if firstCell == nil {
		t.Fatal("expected first nested cell size")
	}
	if got := firstCell.SelectAttrValue("width", ""); got != "4252" {
		t.Fatalf("unexpected nested first col width: %s", got)
	}
	if got := firstCell.SelectAttrValue("height", ""); got != "1701" {
		t.Fatalf("unexpected nested first row height: %s", got)
	}
}

func TestFooterSupportsInlinePageTokens(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "set-footer", editableDir, "--text", "- {{PAGE}} / {{TOTAL_PAGE}} -", "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:footer",
		"<hp:autoNum num=\"1\" numType=\"PAGE\">",
		"<hp:autoNum num=\"1\" numType=\"TOTAL_PAGE\">",
		"<hp:autoNumFormat type=\"DIGIT\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}
}

func TestFootnoteAndEndnoteWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-footnote", editableDir, "--anchor-text", "각주가 있는 본문", "--text", "첫 번째 각주", "--format", "json")
	runCLI(t, "add-endnote", editableDir, "--anchor-text", "미주가 있는 본문", "--text", "첫 번째 미주", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:footNote",
		"<hp:endNote",
		"각주가 있는 본문",
		"첫 번째 각주",
		"미주가 있는 본문",
		"첫 번째 미주",
		"numType=\"FOOTNOTE\"",
		"numType=\"ENDNOTE\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"각주가 있는 본문",
		"첫 번째 각주",
		"미주가 있는 본문",
		"첫 번째 미주",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestMemoWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-memo", editableDir, "--anchor-text", "검토가 필요한 문장", "--text", "첫 번째 메모\n두 번째 메모", "--author", "홍길동", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	headerBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "header.xml"))
	if err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	headerText := string(headerBytes)
	for _, needle := range []string{
		"<hh:memoProperties",
		"<hh:memoPr id=\"0\"",
		"fillColor=\"#CCFF99\"",
	} {
		if !strings.Contains(headerText, needle) {
			t.Fatalf("expected %q in header xml: %s", needle, headerText)
		}
	}

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:memogroup>",
		"<hp:memo id=\"",
		"memoShapeIDRef=\"0\"",
		"type=\"MEMO\"",
		"<hp:stringParam name=\"Author\">홍길동</hp:stringParam>",
		"<hp:stringParam name=\"MemoShapeID\">0</hp:stringParam>",
		"검토가 필요한 문장",
		"첫 번째 메모",
		"두 번째 메모",
		"<hp:fieldEnd",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"검토가 필요한 문장",
		"첫 번째 메모",
		"두 번째 메모",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestBookmarkAndHyperlinkWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-bookmark", editableDir, "--name", "intro", "--text", "소개 위치", "--format", "json")
	runCLI(t, "add-hyperlink", editableDir, "--target", "#intro", "--text", "소개로 이동", "--format", "json")
	runCLI(t, "add-hyperlink", editableDir, "--target", "https://example.com", "--text", "외부 링크", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:bookmark name=\"intro\"",
		"type=\"HYPERLINK\"",
		"name=\"#intro\"",
		"name=\"https://example.com\"",
		"fieldid=\"",
		"<hp:parameters count=\"1\" name=\"\">",
		"<hp:stringParam name=\"Command\">#intro</hp:stringParam>",
		"<hp:stringParam name=\"Command\">https://example.com</hp:stringParam>",
		"<hp:fieldEnd",
		"소개 위치",
		"소개로 이동",
		"외부 링크",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"소개 위치",
		"소개로 이동",
		"외부 링크",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestHeadingTOCAndCrossReferenceWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")

	headingStdout := runCLI(t, "add-heading", editableDir, "--kind", "heading", "--level", "1", "--text", "소개", "--format", "json")
	outlineStdout := runCLI(t, "add-heading", editableDir, "--kind", "outline", "--level", "2", "--text", "세부 항목", "--format", "json")

	var headingEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			BookmarkName string `json:"bookmarkName"`
		} `json:"data"`
	}
	if err := json.Unmarshal(headingStdout.Bytes(), &headingEnvelope); err != nil {
		t.Fatalf("decode add-heading response: %v", err)
	}
	if !headingEnvelope.Success || headingEnvelope.Data.BookmarkName == "" {
		t.Fatalf("unexpected heading response: %s", headingStdout.String())
	}

	var outlineEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			BookmarkName string `json:"bookmarkName"`
		} `json:"data"`
	}
	if err := json.Unmarshal(outlineStdout.Bytes(), &outlineEnvelope); err != nil {
		t.Fatalf("decode outline response: %v", err)
	}
	if !outlineEnvelope.Success || outlineEnvelope.Data.BookmarkName == "" {
		t.Fatalf("unexpected outline response: %s", outlineStdout.String())
	}

	runCLI(t, "insert-toc", editableDir, "--title", "목차", "--max-level", "2", "--format", "json")
	runCLI(t, "add-cross-reference", editableDir, "--bookmark", headingEnvelope.Data.BookmarkName, "--text", "소개로 이동", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"목차",
		"소개",
		"세부 항목",
		"소개로 이동",
		headingEnvelope.Data.BookmarkName,
		outlineEnvelope.Data.BookmarkName,
		"#" + headingEnvelope.Data.BookmarkName,
		"#" + outlineEnvelope.Data.BookmarkName,
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}
	if strings.Count(sectionText, "type=\"HYPERLINK\"") < 3 {
		t.Fatalf("expected toc and cross reference hyperlinks in section xml: %s", sectionText)
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"목차",
		"소개",
		"세부 항목",
		"소개로 이동",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestEquationWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-equation", editableDir, "--script", "a+b", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:equation",
		"numberingType=\"EQUATION\"",
		"version=\"Equation Version 60\"",
		"<hp:script>a+b</hp:script>",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	if !strings.Contains(textEnvelope.Data.Text, "a+b") {
		t.Fatalf("expected equation script in extracted text: %s", textEnvelope.Data.Text)
	}
}

func TestRectangleWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-rectangle", editableDir, "--width-mm", "40", "--height-mm", "20", "--fill-color", "#FFF2CC", "--line-color", "#333333", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:rect",
		"numberingType=\"NONE\"",
		"<hp:lineShape color=\"#333333\"",
		"faceColor=\"#FFF2CC\"",
		"<hc:pt2",
		"<hp:sz width=\"",
		"<hp:pos treatAsChar=\"1\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestLineWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-line", editableDir, "--width-mm", "50", "--height-mm", "10", "--line-color", "#2F5597", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:line",
		"numberingType=\"NONE\"",
		"<hp:lineShape color=\"#2F5597\"",
		"<hc:startPt x=\"0\" y=\"0\">",
		"<hc:endPt x=\"",
		"<hp:sz width=\"",
		"<hp:pos treatAsChar=\"1\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestEllipseWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-ellipse", editableDir, "--width-mm", "40", "--height-mm", "20", "--fill-color", "#FFF2CC", "--line-color", "#333333", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:ellipse",
		"hasArcPr=\"false\"",
		"arcType=\"Normal\"",
		"<hc:center x=\"",
		"<hc:ax1 x=\"",
		"<hc:ax2 x=\"",
		"<hp:lineShape color=\"#333333\"",
		"faceColor=\"#FFF2CC\"",
		"<hp:sz width=\"",
		"<hp:pos treatAsChar=\"1\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}
}

func TestTextBoxWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	archivePath := filepath.Join(workDir, "result.hwpx")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "60", "--height-mm", "25", "--text", "글상자 첫 줄\n글상자 둘째 줄", "--fill-color", "#FFF2CC", "--line-color", "#333333", "--format", "json")
	runCLI(t, "pack", editableDir, "--output", archivePath, "--format", "json")

	sectionBytes, err := os.ReadFile(filepath.Join(editableDir, "Contents", "section0.xml"))
	if err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	sectionText := string(sectionBytes)
	for _, needle := range []string{
		"<hp:rect",
		"<hp:drawText",
		"<hp:textMargin left=\"283\"",
		"<hp:subList",
		"글상자 첫 줄",
		"글상자 둘째 줄",
		"faceColor=\"#FFF2CC\"",
		"<hp:lineShape color=\"#333333\"",
	} {
		if !strings.Contains(sectionText, needle) {
			t.Fatalf("expected %q in section xml: %s", needle, sectionText)
		}
	}

	inspectStdout := runCLI(t, "inspect", archivePath, "--format", "json")
	var inspectEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Report struct {
				Valid bool `json:"valid"`
			} `json:"report"`
		} `json:"data"`
	}
	if err := json.Unmarshal(inspectStdout.Bytes(), &inspectEnvelope); err != nil {
		t.Fatalf("decode inspect response: %v", err)
	}
	if !inspectEnvelope.Success || !inspectEnvelope.Data.Report.Valid {
		t.Fatalf("unexpected inspect response: %s", inspectStdout.String())
	}

	textStdout := runCLI(t, "text", archivePath, "--format", "json")
	var textEnvelope struct {
		Success bool `json:"success"`
		Data    struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(textStdout.Bytes(), &textEnvelope); err != nil {
		t.Fatalf("decode text response: %v", err)
	}
	if !textEnvelope.Success {
		t.Fatalf("unexpected text response: %s", textStdout.String())
	}
	for _, needle := range []string{
		"글상자 첫 줄",
		"글상자 둘째 줄",
	} {
		if !strings.Contains(textEnvelope.Data.Text, needle) {
			t.Fatalf("expected %q in extracted text: %s", needle, textEnvelope.Data.Text)
		}
	}
}

func TestSetParagraphLayoutWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")

	layoutStdout := runCLI(
		t,
		"set-paragraph-layout", editableDir,
		"--paragraph", "1",
		"--align", "CENTER",
		"--indent-mm", "3",
		"--left-margin-mm", "8",
		"--space-after-mm", "4",
		"--line-spacing-percent", "180",
		"--format", "json",
	)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			ParaPrIDRef string `json:"paraPrIdRef"`
		} `json:"data"`
	}
	if err := json.Unmarshal(layoutStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode set-paragraph-layout response: %v", err)
	}
	if !envelope.Success || envelope.Data.ParaPrIDRef == "" {
		t.Fatalf("unexpected set-paragraph-layout response: %s", layoutStdout.String())
	}

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}
	paragraphs := sectionDoc.FindElements("//hp:p")
	if len(paragraphs) < 3 {
		t.Fatalf("expected editable paragraphs in section xml")
	}
	if got := paragraphs[2].SelectAttrValue("paraPrIDRef", ""); got != envelope.Data.ParaPrIDRef {
		t.Fatalf("unexpected paragraph paraPrIDRef: %s", got)
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	paraPr := headerDoc.FindElement("//hh:paraPr[@id='" + envelope.Data.ParaPrIDRef + "']")
	if paraPr == nil {
		t.Fatalf("styled paraPr not found: %s", envelope.Data.ParaPrIDRef)
	}
	align := paraPr.FindElement("./hh:align")
	if align == nil || align.SelectAttrValue("horizontal", "") != "CENTER" {
		t.Fatalf("unexpected align element")
	}
	lineSpacing := paraPr.FindElement(".//hh:lineSpacing")
	if lineSpacing == nil || lineSpacing.SelectAttrValue("value", "") != "180" {
		t.Fatalf("expected updated line spacing in paraPr")
	}
}

func TestSetParagraphListWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "append-text", editableDir, "--text", "첫 문단\n둘째 문단", "--format", "json")

	listStdout := runCLI(
		t,
		"set-paragraph-list", editableDir,
		"--paragraph", "1",
		"--kind", "number",
		"--level", "1",
		"--start-number", "3",
		"--format", "json",
	)

	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			ParaPrIDRef string `json:"paraPrIdRef"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listStdout.Bytes(), &envelope); err != nil {
		t.Fatalf("decode set-paragraph-list response: %v", err)
	}
	if !envelope.Success || envelope.Data.ParaPrIDRef == "" {
		t.Fatalf("unexpected set-paragraph-list response: %s", listStdout.String())
	}

	headerDoc := etree.NewDocument()
	if err := headerDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "header.xml")); err != nil {
		t.Fatalf("read header xml: %v", err)
	}
	paraPr := headerDoc.FindElement("//hh:paraPr[@id='" + envelope.Data.ParaPrIDRef + "']")
	if paraPr == nil {
		t.Fatalf("styled paraPr not found: %s", envelope.Data.ParaPrIDRef)
	}
	heading := paraPr.FindElement("./hh:heading")
	if heading == nil {
		t.Fatalf("heading not found")
	}
	if heading.SelectAttrValue("type", "") != "NUMBER" || heading.SelectAttrValue("level", "") != "1" {
		t.Fatalf("unexpected heading attrs")
	}
	numberingID := heading.SelectAttrValue("idRef", "")
	if numberingID == "" {
		t.Fatalf("numbering id missing")
	}
	numbering := headerDoc.FindElement("//hh:numbering[@id='" + numberingID + "']")
	if numbering == nil || numbering.SelectAttrValue("start", "") != "3" {
		t.Fatalf("unexpected numbering start")
	}
}

func TestSetObjectPositionWorkflow(t *testing.T) {
	workDir := t.TempDir()
	editableDir := filepath.Join(workDir, "editable")
	imagePath := filepath.Join(workDir, "tiny.png")
	if err := os.WriteFile(imagePath, mustTinyPNG(t), 0o644); err != nil {
		t.Fatalf("write image fixture: %v", err)
	}

	runCLI(t, "create", "--output", editableDir, "--format", "json")
	runCLI(t, "insert-image", editableDir, "--image", imagePath, "--width-mm", "20", "--format", "json")
	runCLI(t, "add-textbox", editableDir, "--width-mm", "40", "--height-mm", "20", "--text", "위치 테스트", "--format", "json")
	runCLI(t, "set-object-position", editableDir, "--type", "image", "--index", "0", "--treat-as-char", "false", "--x-mm", "10", "--y-mm", "5", "--horz-align", "CENTER", "--format", "json")
	runCLI(t, "set-object-position", editableDir, "--type", "textbox", "--index", "0", "--x-mm", "12", "--y-mm", "7", "--vert-align", "BOTTOM", "--format", "json")

	sectionDoc := etree.NewDocument()
	if err := sectionDoc.ReadFromFile(filepath.Join(editableDir, "Contents", "section0.xml")); err != nil {
		t.Fatalf("read section xml: %v", err)
	}

	picturePos := sectionDoc.FindElement("//hp:pic/hp:pos")
	if picturePos == nil {
		t.Fatalf("picture position not found")
	}
	if picturePos.SelectAttrValue("treatAsChar", "") != "0" || picturePos.SelectAttrValue("horzAlign", "") != "CENTER" {
		t.Fatalf("unexpected picture position attrs")
	}
	if picturePos.SelectAttrValue("horzOffset", "") == "0" || picturePos.SelectAttrValue("vertOffset", "") == "0" {
		t.Fatalf("expected updated picture offsets")
	}

	textboxPos := sectionDoc.FindElement("//hp:rect[hp:drawText]/hp:pos")
	if textboxPos == nil {
		t.Fatalf("textbox position not found")
	}
	if textboxPos.SelectAttrValue("vertAlign", "") != "BOTTOM" {
		t.Fatalf("unexpected textbox vertAlign")
	}
}

func runCLI(t *testing.T, args ...string) *bytes.Buffer {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	if err := Run(args, stdout, stderr); err != nil {
		t.Fatalf("run %v: %v stderr=%s", args, err, stderr.String())
	}
	return stdout
}

func mustTinyPNG(t *testing.T) []byte {
	t.Helper()

	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO7Z0QAAAABJRU5ErkJggg==")
	if err != nil {
		t.Fatalf("decode tiny png: %v", err)
	}
	return data
}
