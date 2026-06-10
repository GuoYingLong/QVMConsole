package service

import (
	"kvm_console/model"
	"kvm_console/service/vm/memory"
)

// scheduler_register.go — 将 service 根包函数注入到 scheduler 子包（如有需要），
// 并将 scheduler 子包函数注入到其他子包的 Hook 变量中。

func init() {
	// ── 向 memory 子包注册 scheduler 函数（替换原 hooks_init.go 中的逻辑） ──
	memory.HookMemoryRegisterScheduler = func(def memory.SchedulerDefinition) {
		RegisterScheduler(SchedulerDefinition{
			Key:         def.Key,
			Name:        def.Name,
			Group:       def.Group,
			Description: def.Description,
			Enabled:     def.Enabled,
		})
	}
	memory.HookMemoryStartSchedulerEvent = func(input memory.SchedulerEventStartInput) (interface{}, error) {
		return StartSchedulerEvent(SchedulerEventStartInput{
			SchedulerKey:   input.SchedulerKey,
			SchedulerName:  input.SchedulerName,
			SchedulerGroup: input.SchedulerGroup,
			VMName:         input.VMName,
			VMBackend:      input.VMBackend,
			TriggerReason:  input.TriggerReason,
		})
	}
	memory.HookMemoryFinishSchedulerEventOk = func(event interface{}, msg string) error {
		if e, ok := event.(*model.SchedulerEvent); ok {
			return FinishSchedulerEventSuccess(e, msg)
		}
		return nil
	}
	memory.HookMemoryFinishSchedulerEventFail = func(event interface{}, msg string) error {
		if e, ok := event.(*model.SchedulerEvent); ok {
			return FinishSchedulerEventFailed(e, msg)
		}
		return nil
	}
}
