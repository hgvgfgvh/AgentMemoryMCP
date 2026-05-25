package filter

import "testing"

func TestShouldSkipStore_chitchat(t *testing.T) {
	skip, reason := ShouldSkipStore("你好")
	if !skip || reason != "chitchat" {
		t.Fatalf("got skip=%v reason=%q", skip, reason)
	}
}

func TestShouldSkipStore_soulBoundary_whoAmI(t *testing.T) {
	content := `[source=agenttest-plan turn=t1]
## 用户诉求
我是谁

## 门户回复
您是老王，关注项目 Alpha。`
	skip, reason := ShouldSkipStore(content)
	if !skip || reason != "soul_boundary" {
		t.Fatalf("got skip=%v reason=%q", skip, reason)
	}
}

func TestShouldSkipStore_allowsExecutionEpisode(t *testing.T) {
	content := `[source=agenttest-plan turn=t2]
## 用户诉求
列出 WorkSpace 根目录前 5 个条目

## 门户回复
已完成

## 计划终态 (TodoList)
step1: [1] 列出 tier=2 status=completed
  tools_called: filesystem__list_directory
  artifacts: boundary_plan_dir_list.txt`
	skip, reason := ShouldSkipStore(content)
	if skip {
		t.Fatalf("should accept execution episode, skip=%v reason=%q", skip, reason)
	}
}

func TestShouldSkipStore_planWithoutTools(t *testing.T) {
	content := `[source=agenttest-plan]
## 用户诉求
你好 我是谁

## 门户回复
您好，我是助手。`
	skip, reason := ShouldSkipStore(content)
	if !skip || (reason != "soul_boundary" && reason != "not_execution_scope" && reason != "chitchat") {
		t.Fatalf("got skip=%v reason=%q", skip, reason)
	}
}

func TestHasExecutionSignals(t *testing.T) {
	if !hasExecutionSignals("tools_called: filesystem__list_directory") {
		t.Fatal("expected tools marker")
	}
	if hasExecutionSignals("我是谁") {
		t.Fatal("identity only should not match")
	}
}
