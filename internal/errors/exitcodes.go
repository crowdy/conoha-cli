package errors

const (
	ExitOK             = 0
	ExitGeneral        = 1
	ExitAuth           = 2
	ExitNotFound       = 3
	ExitValidation     = 4
	ExitAPI            = 5
	ExitNetwork        = 6
	ExitModeConflict   = 7 // cmd/app: --proxy/--no-proxy disagrees with server marker
	ExitNotInitialized = 8 // cmd/app: a command requiring a server-side marker found none
	ExitCancelled      = 10
)
