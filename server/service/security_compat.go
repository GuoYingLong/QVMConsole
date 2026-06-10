package service

// Security compatibility types - delegate to service/security subpackage
import securitypkg "kvm_console/service/security"

// Type aliases for security subpackage types used by handler layer.
// These make service.XXX and security.XXX interchangeable, ensuring zero changes in handler code.

type SMTPConfigView = securitypkg.SMTPConfigView
type SMTPTestConfig = securitypkg.SMTPTestConfig
type TOTPSetupInfo = securitypkg.TOTPSetupInfo
type TOTPRecoverySetup = securitypkg.TOTPRecoverySetup
type InviteDetail = securitypkg.InviteDetail
type PasswordResetAccountCandidate = securitypkg.PasswordResetAccountCandidate
type SecurityState = securitypkg.SecurityState
