package taskqueue

import (
	"context"
	"errors"
	"testing"
	"time"

	"kvm_console/model"
)

func resetTaskQueueStateForTest() {
	taskStoreMu.Lock()
	defer taskStoreMu.Unlock()

	taskStore = make(map[uint]*model.Task)
	taskCancelFn = make(map[uint]context.CancelFunc)
	taskIDSeq = 0
}

func seedTaskForTest(id uint, taskType, status, createdBy string, createdAt time.Time) {
	taskStore[id] = &model.Task{
		ID:        id,
		Type:      taskType,
		Status:    status,
		CreatedBy: createdBy,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func TestGetTaskListFilteredForUserScopesTasksByCreator(t *testing.T) {
	resetTaskQueueStateForTest()

	now := time.Now()

	taskStoreMu.Lock()
	seedTaskForTest(1, model.TaskTypeClone, model.TaskStatusPending, "alice", now.Add(-1*time.Minute))
	seedTaskForTest(2, model.TaskTypeDelete, model.TaskStatusRunning, "bob", now.Add(-2*time.Minute))
	seedTaskForTest(3, model.TaskTypeClone, model.TaskStatusSuccess, "admin", now.Add(-3*time.Minute))
	taskStoreMu.Unlock()

	tasks, total, err := GetTaskListFilteredForUser(1, 10, "", "", "alice", "user")
	if err != nil {
		t.Fatalf("GetTaskListFilteredForUser returned error: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected alice to see 1 task, got %d", total)
	}
	if len(tasks) != 1 || tasks[0].CreatedBy != "alice" {
		t.Fatalf("expected alice to only see her own task, got %+v", tasks)
	}

	adminTasks, adminTotal, err := GetTaskListFilteredForUser(1, 10, "", "", "admin", "admin")
	if err != nil {
		t.Fatalf("admin GetTaskListFilteredForUser returned error: %v", err)
	}
	if adminTotal != 3 || len(adminTasks) != 3 {
		t.Fatalf("expected admin to see all 3 tasks, total=%d len=%d", adminTotal, len(adminTasks))
	}

	filteredTasks, filteredTotal, err := GetTaskListFilteredForUser(1, 10, model.TaskStatusSuccess, model.TaskTypeClone, "admin", "admin")
	if err != nil {
		t.Fatalf("filtered admin query returned error: %v", err)
	}
	if filteredTotal != 1 || len(filteredTasks) != 1 || filteredTasks[0].ID != 3 {
		t.Fatalf("expected filtered admin query to only return task 3, got total=%d tasks=%+v", filteredTotal, filteredTasks)
	}
}

func TestTaskAccessAndMutationRespectCreatorScope(t *testing.T) {
	resetTaskQueueStateForTest()

	now := time.Now()

	taskStoreMu.Lock()
	seedTaskForTest(1, model.TaskTypeClone, model.TaskStatusPending, "alice", now)
	seedTaskForTest(2, model.TaskTypeDelete, model.TaskStatusSuccess, "alice", now)
	seedTaskForTest(3, model.TaskTypeCreate, model.TaskStatusFailed, "bob", now)
	seedTaskForTest(4, model.TaskTypeSnapshot, model.TaskStatusRunning, "bob", now)
	taskStoreMu.Unlock()

	if _, err := GetTaskForUser(3, "alice", "user"); !errors.Is(err, ErrTaskAccessDenied) {
		t.Fatalf("expected alice task detail access to be denied, got err=%v", err)
	}

	if err := CancelTaskForUser(4, "alice", "user"); !errors.Is(err, ErrTaskAccessDenied) {
		t.Fatalf("expected alice cancel access to be denied, got err=%v", err)
	}

	if err := CancelTaskForUser(1, "alice", "user"); err != nil {
		t.Fatalf("expected alice to cancel her own task, got err=%v", err)
	}

	task, err := GetTaskForUser(1, "alice", "user")
	if err != nil {
		t.Fatalf("expected alice to read her own task after cancel, got err=%v", err)
	}
	if task.Status != model.TaskStatusCanceled {
		t.Fatalf("expected task 1 to be canceled, got status=%s", task.Status)
	}

	cleared, err := ClearFinishedTasksForUser("alice", "user")
	if err != nil {
		t.Fatalf("expected alice clear to succeed, got err=%v", err)
	}
	if cleared != 2 {
		t.Fatalf("expected alice clear to remove 2 finished tasks, got %d", cleared)
	}

	if _, exists := getTask(3); !exists {
		t.Fatalf("expected bob finished task to remain after alice clear")
	}
	if _, exists := getTask(4); !exists {
		t.Fatalf("expected bob running task to remain after alice clear")
	}
}
