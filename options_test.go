package main

import (
	"path/filepath"
	"testing"
)

// TestSetAbsDirOverride 验证 -o/--out 覆盖：非空时按进程工作目录解析为绝对路径，
// 不再拼接程序所在目录。
func TestSetAbsDirOverride(t *testing.T) {
	want, err := filepath.Abs(filepath.Join(t.TempDir(), "my-backup"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	o := &Options{Dir: "backup"}
	o.setAbsDir(want)
	if o.absDir != want {
		t.Errorf("setAbsDir override got=%s want=%s", o.absDir, want)
	}
}

// TestSetAbsDirRelativeOverride 验证相对路径的 -o 也被解析为绝对路径（基于 CWD）。
func TestSetAbsDirRelativeOverride(t *testing.T) {
	o := &Options{Dir: "backup"}
	o.setAbsDir("rel-out")
	want, err := filepath.Abs("rel-out")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if o.absDir != want {
		t.Errorf("setAbsDir relative override got=%s want=%s", o.absDir, want)
	}
}

// TestSetAbsDirDefault 验证不传 -o 时保持历史行为：程序目录 + 配置 dir。
func TestSetAbsDirDefault(t *testing.T) {
	o := &Options{Dir: "backup"}
	o.setAbsDir("")
	want := filepath.Join(GetCurrentDirectory(), "backup")
	if o.absDir != want {
		t.Errorf("setAbsDir default got=%s want=%s", o.absDir, want)
	}
}

// TestResolveDragDropNoArgs 验证无位置参数时原样返回。
func TestResolveDragDropNoArgs(t *testing.T) {
	c, o := resolveDragDrop("config.yaml", "", map[string]bool{}, nil)
	if c != "config.yaml" || o != "" {
		t.Errorf("no args should be unchanged: c=%s o=%s", c, o)
	}
}

// TestResolveDragDropDropped 验证拖拽：位置参数作为配置，导出目录=配置所在目录。
func TestResolveDragDropDropped(t *testing.T) {
	dir := t.TempDir()
	dropped := filepath.Join(dir, "config.yaml")
	c, o := resolveDragDrop("config.yaml", "", map[string]bool{}, []string{dropped})
	if c != dropped {
		t.Errorf("config should be dropped file: got=%s", c)
	}
	if o != dir {
		t.Errorf("out should be dir of config: got=%s want=%s", o, dir)
	}
}

// TestResolveDragDropExplicitFlags 验证显式 -c/-o 时，位置参数不覆盖。
func TestResolveDragDropExplicitFlags(t *testing.T) {
	explicit := map[string]bool{"c": true, "o": true}
	c, o := resolveDragDrop("my.yaml", "D:\\explicit-out", explicit, []string{filepath.Join(t.TempDir(), "config.yaml")})
	if c != "my.yaml" {
		t.Errorf("explicit -c should win: got=%s", c)
	}
	if o != `D:\explicit-out` {
		t.Errorf("explicit -o should win: got=%s", o)
	}
}

// TestResolveDragDropExplicitCOnly 验证只显式 -c 时，导出目录仍取拖入文件所在目录。
func TestResolveDragDropExplicitCOnly(t *testing.T) {
	explicit := map[string]bool{"c": true}
	dir := t.TempDir()
	dropped := filepath.Join(dir, "config.yaml")
	c, o := resolveDragDrop("my.yaml", "", explicit, []string{dropped})
	if c != "my.yaml" {
		t.Errorf("explicit -c should win: got=%s", c)
	}
	if o != dir {
		t.Errorf("out should be dir of dropped: got=%s", o)
	}
}
