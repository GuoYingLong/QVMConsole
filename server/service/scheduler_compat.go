package service

import schedpkg "kvm_console/service/scheduler"

// ── Type aliases（向后兼容，让 service 根包和外部调用方可直接使用类型名） ──

type SchedulerDefinition = schedpkg.SchedulerDefinition
type SchedulerListItem = schedpkg.SchedulerListItem
type SchedulerEventMessage = schedpkg.SchedulerEventMessage
type SchedulerEventStartInput = schedpkg.SchedulerEventStartInput
