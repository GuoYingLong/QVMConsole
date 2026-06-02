package service

import (
	"strings"
	"testing"
)

func TestNormalizeTemplateDeleteMode(t *testing.T) {
	if got := normalizeTemplateDeleteMode(TemplateDeleteModePromote); got != TemplateDeleteModePromote {
		t.Fatalf("期望提升模式，实际为 %s", got)
	}
	if got := normalizeTemplateDeleteMode(TemplateDeleteModePromoteHot); got != TemplateDeleteModePromoteHot {
		t.Fatalf("期望热提升模式，实际为 %s", got)
	}
	if got := normalizeTemplateDeleteMode(""); got != TemplateDeleteModeCascade {
		t.Fatalf("空删除模式应回退为级联删除，实际为 %s", got)
	}
	if got := normalizeTemplateDeleteMode("unknown"); got != TemplateDeleteModeCascade {
		t.Fatalf("未知删除模式应回退为级联删除，实际为 %s", got)
	}
}

func TestBuildTemplatePromoteHotBlockersAllowsRunningVMs(t *testing.T) {
	parent := &TemplateInfo{Name: "base"}
	preview := &DeleteTemplatePreview{
		ParentTemplate:    parent,
		PromotedTemplates: []TemplateInfo{{Name: "child"}},
		RelatedVMs:        []TemplateRelatedVM{{Name: "demo", Status: "running"}},
	}

	if blockers := buildTemplatePromoteHotBlockers(preview); len(blockers) != 0 {
		t.Fatalf("运行中 VM 应允许进入热提升，实际阻断原因: %#v", blockers)
	}
}

func TestBuildTemplatePromoteBlockersRequiresParent(t *testing.T) {
	preview := &DeleteTemplatePreview{
		PromotedTemplates: []TemplateInfo{{Name: "child"}},
		RelatedVMs:        []TemplateRelatedVM{{Name: "demo", Status: "shut off"}},
	}

	blockers := buildTemplatePromoteBlockers(preview)
	if len(blockers) == 0 || !strings.Contains(blockers[0], "根模板没有上级节点") {
		t.Fatalf("应阻止根模板提升，实际阻断原因: %#v", blockers)
	}
}

func TestBuildTemplatePromoteBlockersRequiresStoppedVMs(t *testing.T) {
	parent := &TemplateInfo{Name: "base"}
	preview := &DeleteTemplatePreview{
		ParentTemplate:    parent,
		PromotedTemplates: []TemplateInfo{{Name: "child"}},
		RelatedVMs:        []TemplateRelatedVM{{Name: "demo", Status: "running"}},
	}

	blockers := buildTemplatePromoteBlockers(preview)
	if len(blockers) == 0 || !strings.Contains(blockers[0], "请先关机") {
		t.Fatalf("应阻止运行中 VM 提升，实际阻断原因: %#v", blockers)
	}
}

func TestBuildTemplatePromoteBlockersAllowsStoppedVMs(t *testing.T) {
	parent := &TemplateInfo{Name: "base"}
	preview := &DeleteTemplatePreview{
		ParentTemplate:    parent,
		PromotedTemplates: []TemplateInfo{{Name: "child"}},
		RelatedVMs:        []TemplateRelatedVM{{Name: "demo", Status: "shut off"}},
	}

	if blockers := buildTemplatePromoteBlockers(preview); len(blockers) != 0 {
		t.Fatalf("关机 VM 不应阻止提升，实际阻断原因: %#v", blockers)
	}
}
